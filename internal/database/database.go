package database

import (
	"fmt"
	"goback/internal/models"
	"log"
	"os"

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
	res, err := s.db.Query(`SELECT VALUE array::distinct((SELECT id, username, display_name, status, avatar FROM <->friends<->users WHERE id != $userId)) FROM ONLY $userId;`,
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
