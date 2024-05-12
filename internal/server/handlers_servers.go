package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

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
