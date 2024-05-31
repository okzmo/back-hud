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

type JoinServerBody struct {
	UserId   string `json:"user_id"`
	InviteId string `json:"invite_id"`
}

type CreateServerBody struct {
	UserId string `json:"user_id"`
	Name   string `json:"name"`
}

type GeneralServerBody struct {
	UserId   string `json:"user_id"`
	ServerId string `json:"server_id"`
}

// Servers
func (s *Server) HandlerUsersIdFromChannel(c echo.Context) error {
	resp := make(map[string]any)

	channelId := fmt.Sprintf("channels:%s", c.Param("channelId"))

	users, err := s.db.GetUsersFromChannel(channelId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["users"] = users

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerUserServers(c echo.Context) error {
	resp := make(map[string]any)

	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	servers, err := s.db.GetUserServers(userId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["servers"] = servers

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerServerInformations(c echo.Context) error {
	resp := make(map[string]any)

	userId := fmt.Sprintf("users:%s", c.Param("userId"))
	serverId := fmt.Sprintf("servers:%s", c.Param("serverId"))

	server, err := s.db.GetServer(userId, serverId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["server"] = server

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerJoinServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(JoinServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	server, err := s.db.JoinServer(body.UserId, body.InviteId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(body.UserId, ":")[1]); ok {
		for _, channel := range server.ServerChannels {
			Sub(globalEmitter, channel, &Socket{conn})
		}
		Sub(globalEmitter, server.Server.ID, &Socket{conn})
	}

	resp["server"] = server.Server

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerCreateServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(CreateServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	server, err := s.db.CreateServer(body.UserId, body.Name)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(body.UserId, ":")[1]); ok {
		for _, channel := range server.ServerChannels {
			Sub(globalEmitter, channel, &Socket{conn})
		}
		Sub(globalEmitter, server.Server.ID, &Socket{conn})
	}

	resp["server"] = server.Server

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerDeleteServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(GeneralServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when deleting the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.DeleteServer(body.UserId, body.ServerId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["success"] = true

	wsMess := models.WSMessage{
		Type:    "delete_server",
		Content: body.ServerId,
	}
	data, err := json.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}
	Pub(globalEmitter, body.ServerId, gws.OpcodeText, data)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerLeaveServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(GeneralServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when leaving the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.LeaveServer(body.UserId, body.ServerId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["success"] = true

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerCreateInvitation(c echo.Context) error {
	resp := make(map[string]any)

	body := new(GeneralServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when leaving the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	invitationId, err := s.db.CreateInvitation(body.UserId, body.ServerId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["id"] = invitationId

	return c.JSON(http.StatusOK, resp)
}
