package server

import (
	"encoding/json"
	"fmt"
	"goback/internal/models"
	"goback/internal/utils"
	"goback/proto/protoMess"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/lxzan/gws"
	"google.golang.org/protobuf/proto"
)

type CreateMessage struct {
	Author         models.User `json:"author"`
	ChannelId      string      `json:"channel_id"`
	Content        string      `json:"content"`
	PrivateMessage bool        `json:"private_message"`
	ServerId       string      `json:"server_id,omitempty"`
	Reply          string      `json:"reply,omitempty"`
	Mentions       []string    `json:"mentions,omitempty"`
}

type EditMessage struct {
	ChannelId      string   `json:"channel_id"`
	Content        string   `json:"content"`
	MessageId      string   `json:"message_id,omitempty"`
	AuthorId       string   `json:"author_id"`
	Mentions       []string `json:"mentions,omitempty"`
	PrivateMessage bool     `json:"private_message"`
}

type DeleteMessage struct {
	MessageId      string `json:"message_id,omitempty"`
	ChannelId      string `json:"channel_id"`
	AuthorId       string `json:"author_id"`
	PrivateMessage bool   `json:"private_message"`
}

func (s *Server) HandlerPrivateMessages(c echo.Context) error {
	resp := make(map[string]any)

	channelId := c.Param("channelId")
	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	messages, err := s.db.GetPrivateMessages(userId, channelId)
	if err != nil {
		resp["error"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["messages"] = messages

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChannelMessages(c echo.Context) error {
	resp := make(map[string]any)

	channelId := c.Param("channelId")
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	before, _ := strconv.Atoi(c.QueryParam("before"))

	messages, err := s.db.GetChannelMessages(channelId, limit, before)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["messages"] = messages

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerSendMessage(c echo.Context) error {
	resp := make(map[string]any)

	body := new(CreateMessage)
	bodyStr := c.FormValue("body")
	if err := json.Unmarshal([]byte(bodyStr), body); err != nil {
		log.Println("Error parsing JSON body:", err)
		return c.JSON(http.StatusBadRequest, resp)
	}

	message := models.Message{
		Author:    body.Author,
		ChannelId: body.ChannelId,
		Content:   body.Content,
		Reply:     models.Reply{ID: body.Reply},
		Edited:    false,
		Images:    make([]string, 0),
		Mentions:  make([]string, 0),
	}

	form, err := c.MultipartForm()
	if err != nil {
		log.Println("Error parsing form data:", err)
		return c.JSON(http.StatusBadRequest, resp)
	}

	files := form.File
	if len(files) > 0 {
		var wg sync.WaitGroup
		for _, fileHeader := range files {
			for _, file := range fileHeader {
				wg.Add(1)
				go func(message *models.Message, file *multipart.FileHeader) {
					defer wg.Done()
					if file.Size > 8*1024*1024 {
						resp["error"] = "File size exceeds 8MB limit"
					}

					src, err := file.Open()
					if err != nil {
						log.Println(err)
						resp["error"] = "File size exceeds 8MB limit"
						return
					}
					defer src.Close()

					randId, _ := utils.GenerateRandomId(10)
					imageKey := randId + "-" + file.Filename
					_, err = s.s3.PutObject(&s3.PutObjectInput{
						Bucket: aws.String("Hudori"),
						Key:    aws.String(imageKey),
						Body:   src,
					})
					if err != nil {
						resp["error"] = "File size exceeds 8MB limit"
						return
					}

					message.Images = append(message.Images, os.Getenv("B2_URL")+imageKey)
				}(&message, file)
			}
		}
		wg.Wait()
	}

	message.Mentions = append(message.Mentions, body.Mentions...)
	mess, err := s.db.CreateMessage(message)
	if err != nil {
		log.Println("error when creating a message", err)

		return c.JSON(http.StatusBadRequest, resp)
	}

	go s.SendMessageNotifications(body.PrivateMessage, body.Author.ID, body.ChannelId, body.ServerId, body.Mentions)

	authorObj := &protoMess.User{
		Id:            mess.Author.ID,
		DisplayName:   mess.Author.DisplayName,
		Avatar:        mess.Author.Avatar,
		UsernameColor: mess.Author.UsernameColor,
	}

	messObj := &protoMess.Message{
		Id:        mess.ID,
		Author:    authorObj,
		ChannelId: mess.ChannelId,
		Content:   mess.Content,
		Images:    mess.Images,
		Mentions:  mess.Mentions,
		UpdatedAt: mess.UpdatedAt,
		CreatedAt: mess.CreatedAt,
	}
	if mess.Reply.ID != "" {
		messObj.Replies = &protoMess.Reply{
			Id: mess.Reply.ID,
			Author: &protoMess.User{
				DisplayName: mess.Reply.Author.DisplayName,
			},
			Content: mess.Reply.Content,
		}
	}

	wsMess := &protoMess.WSMessage{
		Type: "text_message",
		Content: &protoMess.WSMessage_Mess{
			Mess: messObj,
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)

	if body.PrivateMessage {
		if conn, ok := s.ws.sessions.Load(strings.Split(body.Author.ID, ":")[1]); ok {
			conn.WriteMessage(gws.OpcodeBinary, compMess)
		}
		if connFriend, ok := s.ws.sessions.Load(body.ChannelId); ok {
			connFriend.WriteMessage(gws.OpcodeBinary, compMess)
		}
	} else {
		Pub(globalEmitter, "channels:"+body.ChannelId, gws.OpcodeBinary, compMess)
	}

	return nil
}

func (s *Server) SendMessageNotifications(privateMessage bool, authorId, channelId, serverId string, mentions []string) {
	if privateMessage {
		notif, err := s.db.CreateMessageNotification("users:"+channelId, "users:"+strings.Split(authorId, ":")[1])
		if err != nil {
			log.Println(err)
		}

		wsMess := &protoMess.WSMessage{
			Type: "new_notification",
			Content: &protoMess.WSMessage_Notification{
				Notification: &protoMess.MessageNotif{
					Id:        notif.ID,
					Type:      "new_message",
					UserId:    notif.UserId,
					ChannelId: notif.ChannelId,
					Counter:   int32(notif.Counter),
					Read:      false,
				},
			},
		}
		data, err := proto.Marshal(wsMess)
		if err != nil {
			log.Println(err)
		}

		compMess := utils.CompressMess(data)
		if conn, ok := s.ws.sessions.Load(strings.Split(authorId, ":")[1]); ok {
			conn.WriteMessage(gws.OpcodeBinary, compMess)
		}

		connFriend, ok := s.ws.sessions.Load(channelId)
		if ok {
			connFriend.WriteMessage(gws.OpcodeBinary, compMess)
		}
	} else {
		users, err := s.db.CreateMessageNotifications(channelId, serverId, authorId, mentions)
		if err != nil {
			log.Println("error when creating a message", err)
		}

		for _, u := range users {
			id, _ := utils.GenerateRandomId(10)
			wsMess := &protoMess.WSMessage{
				Type: "new_notification",
				Content: &protoMess.WSMessage_Notification{
					Notification: &protoMess.MessageNotif{
						Id:        id,
						Type:      "new_message",
						UserId:    u,
						ChannelId: "channels:" + channelId,
						ServerId:  serverId,
						Mentions:  mentions,
						Read:      false,
					},
				},
			}

			data, err := proto.Marshal(wsMess)
			if err != nil {
				log.Println(err)
			}

			compMess := utils.CompressMess(data)
			if conn, ok := s.ws.sessions.Load(strings.Split(u, ":")[1]); ok {
				conn.WriteMessage(gws.OpcodeBinary, compMess)
			}
		}
	}
}

func (s *Server) HandlerEditMessage(c echo.Context) error {
	resp := make(map[string]any)

	body := new(EditMessage)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.EditMessage(body.MessageId, body.Content, body.Mentions)
	if err != nil {
		log.Println("error when editing a message", err)

		return c.JSON(http.StatusBadRequest, resp)
	}

	if body.PrivateMessage {
		author := &protoMess.User{
			Id: body.AuthorId,
		}

		messObj := &protoMess.Message{
			Id:        body.MessageId,
			ChannelId: body.ChannelId,
			Content:   body.Content,
			Mentions:  body.Mentions,
			Edited:    true,
			Author:    author,
		}

		wsMess := &protoMess.WSMessage{
			Type: "edit_message",
			Content: &protoMess.WSMessage_Mess{
				Mess: messObj,
			},
		}
		data, err := proto.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}

		compMess := utils.CompressMess(data)
		if connFriend, ok := s.ws.sessions.Load(body.ChannelId); ok {
			connFriend.WriteMessage(gws.OpcodeBinary, compMess)
		}
	} else {
		messObj := &protoMess.Message{
			Id:        body.MessageId,
			ChannelId: body.ChannelId,
			Content:   body.Content,
			Mentions:  body.Mentions,
			Edited:    true,
		}

		wsMess := &protoMess.WSMessage{
			Type: "edit_message",
			Content: &protoMess.WSMessage_Mess{
				Mess: messObj,
			},
		}
		data, err := proto.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}

		compMess := utils.CompressMess(data)
		Pub(globalEmitter, "channels:"+body.ChannelId, gws.OpcodeBinary, compMess)
	}

	resp["message"] = "success"
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerDeleteMessage(c echo.Context) error {
	resp := make(map[string]any)

	body := new(DeleteMessage)
	if err := c.Bind(body); err != nil {
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.DeleteMessage(body.MessageId)
	if err != nil {
		log.Println("error when deleting a message", err)
		return c.JSON(http.StatusBadRequest, resp)
	}

	if body.PrivateMessage {
		author := &protoMess.User{
			Id: body.AuthorId,
		}

		messObj := &protoMess.Message{
			Id:        body.MessageId,
			ChannelId: body.ChannelId,
			Author:    author,
		}

		wsMess := &protoMess.WSMessage{
			Type: "delete_message",
			Content: &protoMess.WSMessage_Mess{
				Mess: messObj,
			},
		}
		data, err := proto.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}

		compMess := utils.CompressMess(data)
		if connFriend, ok := s.ws.sessions.Load(body.ChannelId); ok {
			connFriend.WriteMessage(gws.OpcodeBinary, compMess)
		}
	} else {
		messObj := &protoMess.Message{
			Id:        body.MessageId,
			ChannelId: body.ChannelId,
		}

		wsMess := &protoMess.WSMessage{
			Type: "delete_message",
			Content: &protoMess.WSMessage_Mess{
				Mess: messObj,
			},
		}
		data, err := proto.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}

		compMess := utils.CompressMess(data)
		Pub(globalEmitter, "channels:"+body.ChannelId, gws.OpcodeBinary, compMess)
	}

	resp["message"] = "success"
	return c.JSON(http.StatusOK, resp)
}
