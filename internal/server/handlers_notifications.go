package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

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
