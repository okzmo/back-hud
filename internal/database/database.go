package database

import (
	"fmt"
	"goback/internal/models"
	"log"
	"os"
	"slices"
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
	GetServer(userId, serverId string) (models.Server, error)
	GetPrivateMessages(userId, channelId string) ([]models.Message, error)
	GetChannelMessages(channelId string) ([]models.Message, error)
	CreateMessage(message models.Message) (models.Message, error)
	RelateFriends(initiatorId, initiatorUsername, receiverUsername string) (models.FriendRequest, error)
	AcceptFriend(requestId, notifId string) ([]models.User, error)
	RefuseFriend(requestId, notifId string) error
	RemoveFriend(userId, FriendId string) error
	GetNotifications(userId string) (interface{}, error)
	JoinServer(userId, serverId string) (jcServerReturn, error)
	GetSubscribedChannels(userId string) ([]models.Channel, error)
	CreateServer(userId, name string) (jcServerReturn, error)
	DeleteServer(userId, serverId string) error
	LeaveServer(userId, serverId string) error
	CreateChannel(serverId, categoryName, channelType, name string) (createChannelReturn, error)
	RemoveChannel(serverId, categoryName, channelId string) error
	CreateCategory(serverId, name string) error
	RemoveCategory(serverId, name string) error
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
	res, err := s.db.Query(`SELECT VALUE array::distinct((SELECT id, username, display_name, status, avatar, about_me FROM <->(friends WHERE accepted=true)<->users WHERE id != $userId)) FROM ONLY $userId;`,
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
	res, err := s.db.Query(`
      SELECT
        roles,
        out.id AS id,
        out.name AS name,
        out.icon AS icon,
        out.created_at AS created_at
      FROM member WHERE in = $userId ORDER BY created_at ASC FETCH out;
    `, map[string]string{
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

func (s *service) GetServer(userId, serverId string) (models.Server, error) {
	res, err := s.db.Query("SELECT * FROM ONLY $serverId FETCH categories.channels", map[string]string{
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

	res, err = s.db.Query("SELECT VALUE roles FROM ONLY member WHERE in = $userId AND out = $serverId LIMIT 1;", map[string]string{
		"userId":   userId,
		"serverId": serverId,
	})
	if err != nil {
		log.Println(err)
		return models.Server{}, err
	}

	roles, err := surrealdb.SmartUnmarshal[[]string](res, err)
	if err != nil {
		log.Println(err)
		return models.Server{}, err
	}

	server.Roles = roles

	return server, nil
}

func (s *service) GetPrivateMessages(userId, channelId string) ([]models.Message, error) {
	res, err := s.db.Query(`
      SELECT author.id, author.username, author.display_name, author.avatar, channel_id, content, id, edited, updated_at, created_at
      FROM messages 
      WHERE (channel_id = $channelId AND author = $userId) OR (channel_id = $userId2 AND author = $channelId2) ORDER BY created_at ASC FETCH author;
    `, map[string]string{
		"userId":     userId,
		"channelId":  "channels:" + channelId,
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
	res, err := s.db.Query(`SELECT author.id, author.username, author.display_name, author.avatar, channel_id, content, id, edited, updated_at, created_at FROM messages WHERE channel_id=$channelId ORDER BY created_at ASC FETCH author;`, map[string]string{
		"channelId": "channels:" + channelId,
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
		"channelId": "channels:" + message.ChannelId,
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

type FriendStruct struct {
	Accepted bool   `json:"accepted"`
	Id       string `json:"id"`
	In       string `json:"in"`
	Out      string `json:"out"`
}

func (s *service) RelateFriends(initiatorId, initiatorUsername, receiverUsername string) (models.FriendRequest, error) {
	existing, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $friendId = SELECT VALUE id FROM users WHERE username=$receiverUsername;
      LET $existingRequest = SELECT VALUE (SELECT id, accepted FROM <->friends WHERE out=$friendId[0] OR in=$friendId[0]) FROM ONLY $initiatorId;

      IF COUNT($existingRequest) == 0 {
          THROW "No existing request"
      };

      RETURN $existingRequest[0];
      COMMIT TRANSACTION;
    `, map[string]string{
		"initiatorId":       initiatorId,
		"initiatorUsername": initiatorUsername,
		"receiverUsername":  receiverUsername,
	})
	if err != nil {
		log.Println(err)
		return models.FriendRequest{}, err
	}

	friend, err := surrealdb.SmartUnmarshal[FriendStruct](existing, err)
	if err == nil {
		if friend.Accepted {
			return models.FriendRequest{}, fmt.Errorf("you're already friend with this person")
		} else if !friend.Accepted {
			return models.FriendRequest{}, fmt.Errorf("a friend request is already pending")
		}
	}

	relateFriends, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $friend = SELECT id FROM users WHERE username=$receiverUsername;
      LET $request = RELATE ONLY $initiatorId->friends->$friend;

      LET $notif = CREATE ONLY notifications CONTENT {
          "type": "friend_request",
          "user_id": $friend[0].id,
          "initiator_id": $initiatorId,
          "request_id": $request.id,
          "message": $initiatorUsername + " sent you a friend request.",
          "created_at": time::now()
      };

      RETURN $notif;
      COMMIT TRANSACTION;
    `, map[string]string{
		"initiatorId":       initiatorId,
		"initiatorUsername": initiatorUsername,
		"receiverUsername":  receiverUsername,
	})
	if err != nil {
		log.Println(err)
		return models.FriendRequest{}, fmt.Errorf("an error occured when adding your friend")
	}

	notif, err := surrealdb.SmartUnmarshal[models.FriendRequest](relateFriends, err)
	if err != nil {
		log.Println(err)
		return models.FriendRequest{}, err
	}

	return notif, nil
}

func (s *service) AcceptFriend(requestId, notifId string) ([]models.User, error) {
	res, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $users = SELECT initiator_id, user_id FROM ONLY $notifId;
      LET $initiator = SELECT id, username, display_name, status, avatar, about_me FROM ONLY $users.initiator_id;  
      LET $receiver = SELECT id, username, display_name, status, avatar, about_me FROM ONLY $users.user_id;

      UPDATE $requestId SET accepted=true;
      DELETE $notifId;

      RETURN [$initiator, $receiver];
      COMMIT TRANSACTION;
    `, map[string]string{
		"requestId": requestId,
		"notifId":   notifId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	users, err := surrealdb.SmartUnmarshal[[]models.User](res, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return users, nil
}

func (s *service) RefuseFriend(requestId, notifId string) error {
	_, err := s.db.Query(`
    DELETE $requestId;
    DELETE $notifId;
    `, map[string]string{
		"requestId": requestId,
		"notifId":   notifId,
	})
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (s *service) RemoveFriend(userId, friendId string) error {
	_, err := s.db.Query(`
      DELETE friends WHERE (in=$userId AND out=$friendId) OR (in=$friendId AND out=$userId);
    `, map[string]string{
		"userId":   userId,
		"friendId": friendId,
	})
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

func (s *service) GetNotifications(userId string) (interface{}, error) {
	res, err := s.db.Query("SELECT * FROM notifications WHERE user_id=$userId ORDER BY created_at DESC", map[string]string{
		"userId": userId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	notifs, err := surrealdb.SmartUnmarshal[interface{}](res, err)
	if err != nil {
		log.Println(err)
		return models.FriendRequest{}, err
	}

	return notifs, err
}

func (s *service) GetSubscribedChannels(userId string) ([]models.Channel, error) {
	res, err := s.db.Query("SELECT VALUE (SELECT id FROM ->subscribed.out) FROM ONLY $userId;", map[string]string{
		"userId": "users:" + userId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	channels, err := surrealdb.SmartUnmarshal[[]models.Channel](res, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return channels, err
}

type jcServerReturn struct {
	Server         models.Server `json:"server"`
	ServerChannels []string      `json:"server_channels"`
}

func (s *service) JoinServer(userId, inviteId string) (jcServerReturn, error) {
	res, err := s.db.Query(`SELECT VALUE server_id FROM invites WHERE invite_id=$inviteId;`, map[string]string{
		"inviteId": inviteId,
	})
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, err
	}

	serverId, err := surrealdb.SmartUnmarshal[[]string](res, err)
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, err
	} else if len(serverId) == 0 {
		return jcServerReturn{}, fmt.Errorf("the invitation is either invalid or has expired")
	}

	existing, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $existingUser = (SELECT VALUE (SELECT id FROM <-member.in WHERE id = $userId) FROM ONLY $serverId);
      RETURN $existingUser[0].id;
      COMMIT TRANSACTION;
    `, map[string]interface{}{
		"serverId": serverId[0],
		"userId":   userId,
	})
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, fmt.Errorf("the invitation is either invalid or has expired")
	}

	existing, err = surrealdb.SmartUnmarshal[string](existing, err)
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, err
	} else if existing != "" {
		return jcServerReturn{}, fmt.Errorf("you already joined this community")
	}

	res, err = s.db.Query(`
	     BEGIN TRANSACTION;
	     LET $server = (SELECT id, icon, name FROM ONLY $serverId);
       LET $serverChannels = (SELECT VALUE array::flatten(categories.channels) FROM ONLY $serverId);
	     RELATE $userId->member->$serverId;
       RELATE $userId->subscribed->$serverChannels;
	     RETURN {
         server: $server,
         server_channels: $serverChannels
       };
	     COMMIT TRANSACTION;
	   `, map[string]string{
		"userId":   userId,
		"serverId": serverId[0],
	})
	fmt.Println(res)
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, fmt.Errorf("the invitation is either invalid or has expired")
	}

	server, err := surrealdb.SmartUnmarshal[jcServerReturn](res, err)
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, fmt.Errorf("the invitation is either invalid or has expired")
	}

	return server, nil
}

func (s *service) CreateServer(userId, name string) (jcServerReturn, error) {
	res, err := s.db.Query(`
        BEGIN TRANSACTION;
        LET $textChannel = (CREATE ONLY channels CONTENT {
          name: "Textual channel",
          type: "textual",
          private: false,
        } RETURN id);

        LET $voiceChannel = (CREATE ONLY channels CONTENT {
          name: "Voice channel",
          type: "voice",
          private: false,
        } RETURN id);

        LET $server = (CREATE ONLY servers CONTENT {
            banner: "",
            icon: "",
            categories: [
                {
                  channels: [
                    $textChannel.id,
                    $voiceChannel.id
                  ],
                  name: 'General'
                }
            ],
            name: $name,
        } RETURN AFTER);

        RELATE $userId->member->$server SET roles = ["owner"];
        RELATE $userId->subscribed->[$textChannel.id, $voiceChannel.id];

        RETURN { 
            server: {
              id: $server.id, 
              name: $server.name,
              icon: $server.icon,
            },
            server_channels: array::flatten($server.categories.channels)
        };
        COMMIT TRANSACTION;
	   `, map[string]string{
		"userId": userId,
		"name":   name,
	})
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, fmt.Errorf("the creation could not be completed, please retry later")
	}
	fmt.Println(res)

	server, err := surrealdb.SmartUnmarshal[jcServerReturn](res, err)
	if err != nil {
		log.Println(err)
		return jcServerReturn{}, fmt.Errorf("the creation could not be completed, please retry later")
	}

	return server, nil
}

func (s *service) DeleteServer(userId, serverId string) error {
	res, err := s.db.Query(`SELECT VALUE roles FROM ONLY member WHERE in = $userId AND out = $serverId LIMIT 1;`, map[string]string{
		"serverId": serverId,
		"userId":   userId,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while deleting the server")
	}

	roles, err := surrealdb.SmartUnmarshal[[]string](res, err)
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while deleting the server")
	} else if !slices.Contains(roles, "owner") {
		return fmt.Errorf("an error occured while deleting the server")
	}

	_, err = s.db.Query(`
      BEGIN TRANSACTION;
      LET $serverChannels = (SELECT VALUE array::flatten(categories.channels) FROM ONLY $serverId);
      DELETE $serverId;
      DELETE $serverChannels;
      DELETE messages WHERE channel_id IN $serverChannels;
      COMMIT TRANSACTION;
	   `, map[string]string{
		"serverId": serverId,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while deleting the server")
	}

	return nil
}

func (s *service) LeaveServer(userId, serverId string) error {
	_, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $serverChannels = (SELECT VALUE array::flatten(categories.channels) FROM ONLY $serverId);

      DELETE member WHERE in=$userId AND out=$serverId;
      DELETE subscribed WHERE in=$userId AND out IN $serverChannels;
      COMMIT TRANSACTION;
	   `, map[string]string{
		"userId":   userId,
		"serverId": serverId,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}

type createChannelReturn struct {
	Channel models.Channel `json:"channel"`
	Members []string       `json:"members"`
}

func (s *service) CreateChannel(serverId, categoryName, channelType, channelName string) (createChannelReturn, error) {
	res, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $channel = (CREATE ONLY channels CONTENT {
          name: $channelName,
          type: $channelType,
          private: false,
      } RETURN AFTER);

      UPDATE $serverId SET categories[WHERE name=$categoryName][0].channels += $channel.id;

      LET $users = (SELECT VALUE <-member.in FROM ONLY $serverId);
      RELATE $users->subscribed->$channel;

      RETURN {
        channel: $channel,
        members: $users
      };
      COMMIT TRANSACTION;
	   `, map[string]string{
		"channelName":  channelName,
		"channelType":  channelType,
		"categoryName": categoryName,
		"serverId":     serverId,
	})
	if err != nil {
		log.Println(err)
		return createChannelReturn{}, fmt.Errorf("an error occured while leaving the server")
	}

	channelAndMembers, err := surrealdb.SmartUnmarshal[createChannelReturn](res, err)
	if err != nil {
		log.Println(err)
		return createChannelReturn{}, fmt.Errorf("an error occured while deleting the server")
	}

	return channelAndMembers, nil
}

func (s *service) RemoveChannel(serverId, categoryName, channelId string) error {
	_, err := s.db.Query(`
      BEGIN TRANSACTION;
      DELETE $channelId;
      UPDATE $serverId SET categories[WHERE name=$categoryName][0].channels -= $channelId;
      DELETE subscribed WHERE out=$channelId;
      DELETE messages WHERE channel_id=$channelId;
      COMMIT TRANSACTION;
	   `, map[string]string{
		"categoryName": categoryName,
		"channelId":    channelId,
		"serverId":     serverId,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}

func (s *service) CreateCategory(serverId, categoryName string) error {
	_, err := s.db.Query(`UPDATE $serverId SET categories += { name: $categoryName, channels: []};`, map[string]string{
		"categoryName": categoryName,
		"serverId":     serverId,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}

func (s *service) RemoveCategory(serverId, categoryName string) error {
	_, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $category = SELECT VALUE categories[WHERE name=$categoryName][0] FROM ONLY $serverId;
      UPDATE $serverId SET categories -= $category;
      DELETE $category.channels;
      DELETE messages WHERE channel_id IN $category.channels;
      COMMIT TRANSACTION;
	   `, map[string]string{
		"categoryName": categoryName,
		"serverId":     serverId,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}
