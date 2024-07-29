package database

import (
	"fmt"
	"goback/internal/models"
	"goback/internal/utils"
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
	DeleteSession(id string) error
	GetFriends(userId string) ([]models.User, error)
	GetUsersFromChannel(channelId string) ([]string, error)
	GetUserServers(userId string) ([]models.Server, error)
	GetServer(userId, serverId string) (models.Server, error)
	GetPrivateMessages(userId, channelId string) ([]models.Message, error)
	GetChannelMessages(channelId string, limit, before int) ([]models.Message, error)
	CreateMessage(message models.Message) (models.Message, error)
	EditMessage(messageId, content string, mentions []string) error
	DeleteMessage(messageId string) error
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
	RemoveCategory(serverId, name string) ([]string, error)
	CreateInvitation(userId, serverId string) (string, error)
	CreateMessageNotification(userId, channelId string) (models.MessageNotif, error)
	CreateMessageNotifications(channelId, serverId, authorId string, mentions []string) ([]string, error)
	UpdateMessageNotifications(userId string, channels []string) error
	ChangeEmail(userId, email string) error
	ChangeUsername(userId, username string) error
	ChangeDisplayName(userId, displayName string) error
	ChangeNameColor(userId, usernameColor string) error
	UpdateBanner(userId, bannerLink string) (string, error)
	UpdateAvatar(userId, avatarLink string) (string, error)
	UpdateServerIcon(serverId, avatarLink string) (string, error)
	UpdateServerBanner(serverId, bannerLink string) (string, error)
	CheckInvitationValidity(InviteId string) (models.Invitation, error)
	UpdateUserStatus(userId string, status string) error
}

type service struct {
	db *surrealdb.DB
}

var (
	username  = os.Getenv("DB_USERNAME")
	password  = os.Getenv("DB_PASSWORD")
	namespace = os.Getenv("DB_NAMESPACE")
	database  = os.Getenv("DB_DATABASE")
	url       = os.Getenv("DB_URL")
)

func New() Service {
	db, err := surrealdb.New(url)
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

func (s *service) DeleteSession(sessionId string) error {
	_, err := s.db.Delete(sessionId)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetFriends(userId string) ([]models.User, error) {
	res, err := s.db.Query(`SELECT VALUE array::distinct((SELECT id, username, display_name, status, avatar, about_me, username_color FROM <->(friends WHERE accepted=true)<->users WHERE id != $userId)) FROM ONLY $userId;`,
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
        out.banner AS banner,
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

	res, err = s.db.Query("SELECT VALUE (SELECT id, username, avatar, display_name, username_color FROM <-member.in) as member FROM ONLY $serverId FETCH member;", map[string]string{
		"serverId": serverId,
	})
	if err != nil {
		log.Println(err)
		return models.Server{}, err
	}

	members, err := surrealdb.SmartUnmarshal[[]models.User](res, err)
	if err != nil {
		log.Println(err)
		return models.Server{}, err
	}

	server.Members = members

	return server, nil
}

type PrivateMessages struct {
	Messages []models.Message `json:"messages"`
	Members  []models.User    `json:"members"`
}

func (s *service) GetPrivateMessages(userId, channelId string) ([]models.Message, error) {
	res, err := s.db.Query(`
      SELECT author.id, author.username, author.display_name, author.username_color, author.avatar, channel_id, content, id, edited, images, mentions, updated_at, created_at, replies.id, replies.content, replies.author.display_name
      FROM messages 
      WHERE (channel_id = $channelId AND author = $userId) OR (channel_id = $userId2 AND author = $channelId2) ORDER BY created_at ASC FETCH author, replies;
    `, map[string]string{
		"userId":     userId,
		"channelId":  "channels:" + channelId,
		"userId2":    "channels:" + strings.Split(userId, ":")[1],
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

	_, err = s.db.Query(`
     UPDATE notifications SET read = true WHERE user_id=$userId AND channel_id=$channelId;
    `, map[string]string{
		"userId":    userId,
		"channelId": "users:" + channelId,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return messages, nil
}

func (s *service) GetChannelMessages(channelId string, limit, before int) ([]models.Message, error) {
	res, err := s.db.Query(`SELECT author.id, author.username, author.display_name, author.username_color, author.avatar, channel_id, content, images, mentions, id, edited, updated_at, created_at, replies.id, replies.content, replies.author.display_name FROM messages WHERE channel_id=$channelId ORDER BY created_at DESC LIMIT $limit START $before FETCH author, replies;`, map[string]interface{}{
		"channelId": "channels:" + channelId,
		"before":    before,
		"limit":     limit,
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
	params := map[string]any{
		"authorId":  message.Author.ID,
		"channelId": "channels:" + message.ChannelId,
		"content":   message.Content,
		"edited":    message.Edited,
		"images":    message.Images,
		"mentions":  message.Mentions,
	}

	if message.Reply.ID != "" {
		params["reply"] = message.Reply.ID
	}

	createRes, err := s.db.Query(`
    CREATE ONLY messages CONTENT {
      "author": $authorId,
      "channel_id": $channelId,
      "content": $content,
      "edited": $edited,
      "images": $images,
      "mentions": $mentions,
      "replies": $reply,
    } RETURN id;
    `, params)
	if err != nil {
		return models.Message{}, err
	}

	id, err := surrealdb.SmartUnmarshal[CreateMessage](createRes, err)
	if err != nil {
		log.Println(err)
		return models.Message{}, err
	}

	messageRes, err := s.db.Query(`
    SELECT author.id, author.username, author.display_name, author.username_color, author.avatar, channel_id, content, images, mentions, id, edited, updated_at, created_at, replies.id, replies.content, replies.author.display_name FROM ONLY $id FETCH author, replies;
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

func (s *service) EditMessage(messageId, content string, mentions []string) error {
	_, err := s.db.Query(`
      UPDATE $messageId MERGE {
          content: $content,
          edited: true,
          mentions: $mentions,
          updated_at: time::now()
      }
    `, map[string]any{
		"messageId": messageId,
		"mentions":  mentions,
		"content":   content,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) DeleteMessage(messageId string) error {
	_, err := s.db.Query(`
      DELETE $messageId;
    `, map[string]any{
		"messageId": messageId,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) CreateMessageNotifications(channelId, serverId, authorId string, mentions []string) ([]string, error) {
	createRes, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $users = SELECT VALUE <-subscribed.in FROM ONLY $channelId;
      FOR $user IN $users {
        IF $user != $authorId
          {
            LET $existingNotif = (SELECT id FROM notifications WHERE user_id=$user AND channel_id=$channelId);
            IF $existingNotif {
              UPDATE $existingNotif.id MERGE { 
                  mentions: $mentions,
                  read: false,
              };
            } ELSE {
              CREATE ONLY notifications CONTENT {
                channel_id: $channelId,
                created_at: time::now(),
                mentions: $mentions,
                server_id: $serverId,
                type: 'new_message',
                user_id: $user,
                read: false
              };
            }
          }
        ;
      };
      RETURN $users;
      COMMIT TRANSACTION;
    `, map[string]any{
		"mentions":  mentions,
		"channelId": "channels:" + channelId,
		"serverId":  serverId,
		"authorId":  authorId,
	})
	if err != nil {
		return nil, err
	}

	users, err := surrealdb.SmartUnmarshal[[]string](createRes, err)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return users, nil
}

func (s *service) UpdateMessageNotifications(userId string, channels []string) error {
	_, err := s.db.Query(`
      UPDATE notifications SET read = true WHERE channel_id IN $channels AND user_id=$userId;
    `, map[string]any{
		"channels": channels,
		"userId":   userId,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) CreateMessageNotification(userId, channelId string) (models.MessageNotif, error) {
	createRes, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $existingNotif = (SELECT * FROM notifications WHERE channel_id = $channelId LIMIT 1);
      LET $result = IF $existingNotif {
          LET $update = UPDATE ONLY $existingNotif 
          SET counter = IF read == true THEN 1 ELSE IF counter < 10 THEN counter + 1 ELSE counter END,
          read = false;
              $update;
      } ELSE { 
          LET $create = CREATE ONLY notifications CONTENT {
            channel_id: $channelId,
            counter: 1,
            created_at: time::now(),
            type: 'new_message',
            user_id: $userId,
            read: false,
          };
          $create;
      };
      RETURN $result;
      COMMIT TRANSACTION;
    `, map[string]any{
		"userId":    userId,
		"channelId": channelId,
	})
	if err != nil {
		return models.MessageNotif{}, err
	}

	notif, err := surrealdb.SmartUnmarshal[models.MessageNotif](createRes, err)
	if err != nil {
		log.Println(err)
		return models.MessageNotif{}, err
	}

	return notif, nil
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
	res, err := s.db.Query("SELECT * FROM notifications WHERE user_id=$userId AND read=false ORDER BY created_at DESC", map[string]string{
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
	log.Println(notifs)

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

func (s *service) RemoveCategory(serverId, categoryName string) ([]string, error) {
	res, err := s.db.Query(`
      BEGIN TRANSACTION;
      LET $category = SELECT VALUE categories[WHERE name=$categoryName][0] FROM ONLY $serverId;
      UPDATE $serverId SET categories -= $category;
      DELETE $category.channels;
      DELETE messages WHERE channel_id IN $category.channels;

      RETURN $category.channels;
      COMMIT TRANSACTION;
	   `, map[string]string{
		"categoryName": categoryName,
		"serverId":     serverId,
	})
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("an error occured while leaving the server")
	}

	channels, err := surrealdb.SmartUnmarshal[[]string](res, err)
	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("an error occured while deleting the server")
	}

	return channels, nil
}

type inviteId struct {
	InviteId string `json:"invite_id"`
}

func (s *service) CreateInvitation(userId, serverId string) (string, error) {
	var invitationId inviteId
	res, err := s.db.Query(`SELECT invite_id FROM ONLY invites WHERE user_id=$userId AND server_id=$serverId LIMIT 1`, map[string]string{
		"userId":   userId,
		"serverId": serverId,
	})
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("no invitation found for this space")
	}

	invitationId, err = surrealdb.SmartUnmarshal[inviteId](res, err)
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("no invitation found for this space")
	}

	if len(invitationId.InviteId) > 0 {
		return invitationId.InviteId, nil
	}

	id, _ := utils.GenerateRandomId()

	_, err = s.db.Query(`
      CREATE invites CONTENT {
          invite_id: $inviteId,
          server_id: $serverId,
          user_id: $userId,
      }
	   `, map[string]string{
		"inviteId": id,
		"userId":   userId,
		"serverId": serverId,
	})
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("an error occured while creating an invitation")
	}

	return id, nil
}

func (s *service) ChangeEmail(userId, email string) error {
	_, err := s.db.Query(`UPDATE $userId SET email=$email`, map[string]string{
		"userId": userId,
		"email":  email,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}

func (s *service) ChangeUsername(userId, username string) error {
	_, err := s.db.Query(`UPDATE $userId SET username=$username`, map[string]string{
		"userId":   userId,
		"username": username,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}

func (s *service) ChangeDisplayName(userId, displayName string) error {
	_, err := s.db.Query(`UPDATE $userId SET display_name=$displayName`, map[string]string{
		"userId":      userId,
		"displayName": displayName,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}

func (s *service) ChangeNameColor(userId, usernameColor string) error {
	_, err := s.db.Query(`UPDATE $userId SET username_color=$usernameColor`, map[string]string{
		"userId":        userId,
		"usernameColor": usernameColor,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}

func (s *service) UpdateBanner(userId, bannerKey string) (string, error) {
	res, err := s.db.Query(`UPDATE ONLY $userId SET banner=$bannerLink RETURN banner`, map[string]string{
		"userId":     "users:" + userId,
		"bannerLink": "https://f003.backblazeb2.com/file/Hudori/" + bannerKey,
	})
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("an error occured while leaving the server")
	}

	user, err := surrealdb.SmartUnmarshal[models.User](res, err)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return user.Banner, nil
}

func (s *service) UpdateAvatar(userId, avatarKey string) (string, error) {
	res, err := s.db.Query(`UPDATE ONLY $userId SET avatar=$avatarLink RETURN avatar`, map[string]string{
		"userId":     "users:" + userId,
		"avatarLink": "https://f003.backblazeb2.com/file/Hudori/" + avatarKey,
	})
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("an error occured while leaving the server")
	}

	user, err := surrealdb.SmartUnmarshal[models.User](res, err)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return user.Avatar, nil
}

func (s *service) UpdateServerBanner(serverId, bannerKey string) (string, error) {
	res, err := s.db.Query(`UPDATE ONLY $serverId SET banner=$bannerLink RETURN banner`, map[string]string{
		"serverId":   serverId,
		"bannerLink": "https://f003.backblazeb2.com/file/Hudori/" + bannerKey,
	})
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("an error occured while leaving the server")
	}

	server, err := surrealdb.SmartUnmarshal[models.Server](res, err)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return server.Banner, nil
}

func (s *service) UpdateServerIcon(serverId, iconKey string) (string, error) {
	res, err := s.db.Query(`UPDATE ONLY $serverId SET icon=$iconLink RETURN icon`, map[string]string{
		"serverId": serverId,
		"iconLink": "https://f003.backblazeb2.com/file/Hudori/" + iconKey,
	})
	log.Println(res)
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("an error occured while leaving the server")
	}

	server, err := surrealdb.SmartUnmarshal[models.Server](res, err)
	if err != nil {
		log.Println(err)
		return "", err
	}

	return server.Icon, nil
}

func (s *service) CheckInvitationValidity(invitationId string) (models.Invitation, error) {
	res, err := s.db.Query(`
      BEGIN TRANSACTION;
      let $invitation = (SELECT id, number_of_use, initiator.display_name, initiator.banner FROM ONLY $invitationId FETCH initiator);

      IF !$invitation {
          THROW "This invitation does not exist."
      } ELSE IF $invitation.number_of_use <= 0 {
          THROW "This invitation has expired."
      };

      RETURN $invitation;
      COMMIT TRANSACTION;
    `, map[string]string{
		"invitationId": invitationId,
	})
	if err != nil {
		log.Println(err)
		return models.Invitation{}, fmt.Errorf("an error occured while leaving the server")
	}

	invitation, err := surrealdb.SmartUnmarshal[models.Invitation](res, err)
	if err != nil {
		log.Println(err)
		return models.Invitation{}, err
	}

	return invitation, nil
}

func (s *service) UpdateUserStatus(userId, status string) error {
	_, err := s.db.Query(`UPDATE ONLY $userId SET status=$status`, map[string]string{
		"userId": "users:" + userId,
		"status": status,
	})
	if err != nil {
		log.Println(err)
		return fmt.Errorf("an error occured while leaving the server")
	}

	return nil
}
