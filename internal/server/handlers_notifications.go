package server

import (
	"fmt"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

type updateNotifications struct {
	UserId   string   `json:"user_id"`
	Channels []string `json:"channels"`
}

func (s *Server) HandlerNotifications(c echo.Context) error {
	resp := make(map[string]any)

	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	notifications, err := s.db.GetNotifications(userId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["notifications"] = notifications

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerUpdateNotifications(c echo.Context) error {
	body := new(updateNotifications)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		return err
	}

	err := s.db.UpdateMessageNotifications(body.UserId, body.Channels)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}
