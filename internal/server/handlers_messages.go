package server

import (
	"encoding/json"
	"fmt"
	"goback/internal/models"
	"goback/internal/utils"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lxzan/gws"
)

type CreateMessage struct {
	Author         models.User     `json:"author"`
	ChannelId      string          `json:"channel_id"`
	Content        json.RawMessage `json:"content"`
	PrivateMessage bool            `json:"private_message"`
}

func (s *Server) HandlerPrivateMessages(c echo.Context) error {
	resp := make(map[string]any)

	channelId := c.Param("channelId")
	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	messages, err := s.db.GetPrivateMessages(userId, channelId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["messages"] = messages

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChannelMessages(c echo.Context) error {
	resp := make(map[string]any)

	channelId := c.Param("channelId")

	messages, err := s.db.GetChannelMessages(channelId)
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
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occurred when sending your message."

		return c.JSON(http.StatusBadRequest, resp)
	}

	message := models.Message{
		Author:    body.Author,
		ChannelId: body.ChannelId,
		Content:   body.Content,
		Edited:    false,
	}

	mess, err := s.db.CreateMessage(message)
	if err != nil {
		log.Println("error when creating a message", err)
		resp["message"] = "An error occured when sending your message."

		return c.JSON(http.StatusBadRequest, resp)
	}

	if body.PrivateMessage {
		id, _ := utils.GenerateRandomId(10)
		notif := models.MessageNotif{
			ID:        id,
			Type:      "new_message",
			Counter:   1,
			UserId:    "users:" + body.ChannelId,
			ChannelId: "channels:" + strings.Split(body.Author.ID, ":")[1],
		}
		wsMess := models.WSMessage{
			Type:    "text_message",
			Content: mess,
			Notif:   notif,
		}
		data, err := json.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}

		if conn, ok := s.ws.sessions.Load(strings.Split(body.Author.ID, ":")[1]); ok {
			conn.WriteMessage(gws.OpcodeText, data)
		}

		connFriend, ok := s.ws.sessions.Load(body.ChannelId)
		if ok {
			connFriend.WriteMessage(gws.OpcodeText, data)
		}
	} else {
		id, _ := utils.GenerateRandomId(10)
		notif := models.MessageNotif{
			ID:        id,
			Type:      "new_message",
			UserId:    body.Author.ID,
			ChannelId: "channels:" + body.ChannelId,
		}
		wsMess := models.WSMessage{
			Type:    "text_message",
			Content: mess,
			Notif:   notif,
		}
		data, err := json.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}
		Pub(globalEmitter, "channels:"+body.ChannelId, gws.OpcodeText, data)
	}

	return c.JSON(http.StatusOK, resp)
}
