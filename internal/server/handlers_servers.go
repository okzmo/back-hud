package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

type JoinServerBody struct {
	UserId   string `json:"user_id"`
	InviteId string `json:"invite_id"`
}

type CreateServerBody struct {
	UserId string `json:"user_id"`
	Name   string `json:"name"`
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

	serverId := fmt.Sprintf("servers:%s", c.Param("serverId"))

	server, err := s.db.GetServer(serverId)
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
	fmt.Println(server, err)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["server"] = server

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
	fmt.Println(server, err)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["server"] = server

	return c.JSON(http.StatusOK, resp)
}
