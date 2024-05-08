package database

import (
	"fmt"
	"goback/internal/models"
	"log"
	"os"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"github.com/surrealdb/surrealdb.go"
)

type Service interface {
	// Health() map[string]string
	CreateUser(user models.User) (string, error)
	CreateSession(session models.Session) (models.Session, error)
	GetUser(id, username, email string) (models.User, error)
	GetSession(id string) (models.Session, error)
	GetFriends(userId string) ([]models.User, error)
	GetUsersFromChannel(channelId string) ([]string, error)
	GetUserServers(userId string) ([]models.Server, error)
	GetServer(serverId string) (models.Server, error)
	GetPrivateMessages(userId, channelId string) ([]models.Message, error)
	GetChannelMessages(channelId string) ([]models.Message, error)
	CreateMessage(message models.Message) (models.Message, error)
}

type service struct {
	db *surrealdb.DB
}

var (
	username  = os.Getenv("DB_USERNAME")
	password  = os.Getenv("DB_PASSWORD")
	namespace = os.Getenv("DB_NAMESPACE")
	database  = os.Getenv("DB_DATABASE")
)

func New() Service {
	db, err := surrealdb.New("ws://localhost:8000/rpc")
	if err != nil {
		panic(err)
	}

	if _, err := db.Signin(map[string]interface{}{
		"user": username,
		"pass": password,
	}); err != nil {
		panic(err)
	}

	if _, err := db.Use(namespace, database); err != nil {
		panic(err)
	}

	s := &service{db: db}
	return s
}

func (s *service) GetUser(id, username, email string) (models.User, error) {
	var data interface{}
	var err error

	if id != "" {
		data, err = s.db.Select(id)
	} else if email != "" {
		data, err = s.db.Query("SELECT * FROM users WHERE email = $email", map[string]interface{}{
			"email": email,
		})
	} else if username != "" {
		data, err = s.db.Query("SELECT * FROM users WHERE username = $username", map[string]interface{}{
			"username": username,
		})
	}

	if err != nil {
		return models.User{}, err
	}

	if username != "" || email != "" {
		var users []models.User
		if ok, err := surrealdb.UnmarshalRaw(data, &users); !ok {
			if err != nil {
				return models.User{}, err
			}

			return models.User{}, fmt.Errorf("no user found")
		}
		return users[0], nil
	}

	var user models.User
	if err := surrealdb.Unmarshal(data, &user); err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s *service) CreateUser(user models.User) (string, error) {
	var users []models.User

	data, err := s.db.Create("users", user)
	if err != nil {
		return "", err
	}

	err = surrealdb.Unmarshal(data, &users)
	if err != nil {
		return "", err
	}

	return users[0].ID, nil
}

func (s *service) CreateSession(session models.Session) (models.Session, error) {
	var sess []models.Session
	data, err := s.db.Create("sessions", session)
	if err != nil {
		return models.Session{}, err
	}

	err = surrealdb.Unmarshal(data, &sess)
	if err != nil {
		return models.Session{}, err
	}

	return sess[0], nil
}

func (s *service) GetSession(sessionId string) (models.Session, error) {
	data, err := s.db.Select(sessionId)
	if err != nil {
		return models.Session{}, err
	}

	var session models.Session
	err = surrealdb.Unmarshal(data, &session)
	if err != nil {
		return models.Session{}, err
	}

	return session, nil
}

func (s *service) GetFriends(userId string) ([]models.User, error) {
	res, err := s.db.Query(`SELECT VALUE array::distinct((SELECT id, username, display_name, status, avatar, about_me FROM <->friends<->users WHERE id != $userId)) FROM ONLY $userId;`,
		map[string]interface{}{
			"userId": userId,
		})
	if err != nil {
		return nil, err
	}

	friends, err := surrealdb.SmartUnmarshal[[]models.User](res, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return friends, nil
}

type UsersId struct {
	Users []string `json:"users"`
}

func (s *service) GetUsersFromChannel(channelId string) ([]string, error) {
	res, err := s.db.Query("SELECT <-subscribed.in AS users FROM ONLY $channelId;", map[string]string{
		"channelId": channelId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	users, err := surrealdb.SmartUnmarshal[UsersId](res, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return users.Users, nil
}

func (s *service) GetUserServers(userId string) ([]models.Server, error) {
	res, err := s.db.Query("SELECT VALUE (SELECT id, name, icon FROM ->member.out) FROM ONLY $userId;", map[string]string{
		"userId": userId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	servers, err := surrealdb.SmartUnmarshal[[]models.Server](res, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return servers, nil
}

func (s *service) GetServer(serverId string) (models.Server, error) {
	res, err := s.db.Query("SELECT * FROM ONLY $serverId FETCH channels", map[string]string{
		"serverId": serverId,
	})
	if err != nil {
		log.Println(err)
		return models.Server{}, err
	}

	server, err := surrealdb.SmartUnmarshal[models.Server](res, err)
	if err != nil {
		log.Println(err)
		return models.Server{}, err
	}

	return server, nil
}

func (s *service) GetPrivateMessages(userId, channelId string) ([]models.Message, error) {
	res, err := s.db.Query(`SELECT author.id, author.username, author.display_name, author.avatar, channel_id, content, id, edited, updated_at 
	                        FROM messages 
													WHERE (channel_id = $channelId AND author = $userId) OR (channel_id = $userId2 AND author = $channelId2) ORDER BY updated_at ASC FETCH author;`, map[string]string{
		"userId":     userId,
		"channelId":  channelId,
		"userId2":    strings.Split(userId, ":")[1],
		"channelId2": "users:" + channelId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	messages, err := surrealdb.SmartUnmarshal[[]models.Message](res, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return messages, nil
}

func (s *service) GetChannelMessages(channelId string) ([]models.Message, error) {
	res, err := s.db.Query(`SELECT author.id, author.username, author.display_name, author.avatar, channel_id, content, id, edited, updated_at FROM messages WHERE channel_id=$channelId ORDER BY updated_at ASC FETCH author;`, map[string]string{
		"channelId": channelId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	messages, err := surrealdb.SmartUnmarshal[[]models.Message](res, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return messages, nil
}

type CreateMessage struct {
	ID string `json:"id"`
}

func (s *service) CreateMessage(message models.Message) (models.Message, error) {
	createRes, err := s.db.Query(`
    CREATE ONLY messages CONTENT {
      "author": $authorId,
      "channel_id": $channelId,
      "content": $content,
      "edited": $edited,
    } RETURN id;
    `, map[string]any{
		"authorId":  message.Author.ID,
		"channelId": message.ChannelId,
		"content":   message.Content,
		"edited":    message.Edited,
	})
	if err != nil {
		return models.Message{}, err
	}

	id, err := surrealdb.SmartUnmarshal[CreateMessage](createRes, err)
	if err != nil {
		log.Println(err)
		return models.Message{}, err
	}

	messageRes, err := s.db.Query(`
    SELECT author.id, author.username, author.display_name, author.avatar, channel_id, content, id, edited, updated_at FROM ONLY $id FETCH author;
    `, map[string]any{
		"id": id.ID,
	})
	if err != nil {
		return models.Message{}, err
	}

	messageCreated, err := surrealdb.SmartUnmarshal[models.Message](messageRes, err)
	if err != nil {
		log.Println(err)
		return models.Message{}, err
	}

	return messageCreated, nil
}
