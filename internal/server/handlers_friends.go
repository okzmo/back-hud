package server

import (
	"encoding/json"
	"fmt"
	"goback/internal/models"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lxzan/gws"
)

type AddFriendBody struct {
	InitiatorId       string `json:"initiator_id"`
	InitiatorUsername string `json:"initiator_username"`
	ReceiverUsername  string `json:"receiver_username"`
}

type AcceptFriendBody struct {
	RequestId string `json:"request_id"`
	NotifId   string `json:"id"`
}

func (s *Server) HandlerFriends(c echo.Context) error {
	resp := make(map[string]any)

	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	friends, err := s.db.GetFriends(userId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["friends"] = friends

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerAddFriend(c echo.Context) error {
	resp := make(map[string]any)

	body := new(AddFriendBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when adding your friend."

		return c.JSON(http.StatusBadRequest, resp)
	}

	notif, err := s.db.RelateFriends(body.InitiatorId, body.InitiatorUsername, body.ReceiverUsername)
	if err != nil {
		log.Println("error when relating users:", err)
		resp["name"] = "unexpected"
		resp["message"] = err.Error()

		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(notif.UserId, ":")[1]); ok {
		mess := models.WSMessage{
			Type:    "friend_request",
			Content: notif,
		}
		data, err := json.Marshal(mess)
		if err != nil {
			log.Println(err)
			return err
		}
		conn.WriteMessage(gws.OpcodeText, data)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerAcceptFriend(c echo.Context) error {
	resp := make(map[string]any)

	body := new(AcceptFriendBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when sending your message."

		return c.JSON(http.StatusBadRequest, resp)
	}

	users, err := s.db.AcceptFriend(body.RequestId, body.NotifId)
	if err != nil {
		log.Println("error when accepting friend request", err)
		resp["message"] = "An error occured when accepting friend request."

		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(users[0].ID, ":")[1]); ok {
		mess := models.WSMessage{
			Type:    "friend_accept",
			Content: users[1],
		}
		data, err := json.Marshal(mess)
		if err != nil {
			log.Println(err)
			return err
		}
		conn.WriteMessage(gws.OpcodeText, data)
	}

	resp["message"] = "success"
	resp["friend"] = users[0]

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerRefuseFriend(c echo.Context) error {
	resp := make(map[string]any)

	body := new(AcceptFriendBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when sending your message."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.RefuseFriend(body.RequestId, body.NotifId)
	if err != nil {
		log.Println("error when refusing friend request", err)
		resp["message"] = "An error occured when refusing friend request."

		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}
