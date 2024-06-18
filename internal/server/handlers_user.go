package server

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
)

type ChangeInformations struct {
	UserId      string  `json:"user_id"`
	Username    *string `json:"username,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
}

func (s *Server) HandlerChangeEmail(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "email"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.ChangeEmail(body.UserId, *body.Email)
	if err != nil {
		log.Println(err)
		resp["name"] = "email"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChangeUsername(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "username"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	_, err := s.db.GetUser("", *body.Username, "")
	if err == nil {
		resp["name"] = "username"
		resp["message"] = "This username is already in use."
		return c.JSON(http.StatusBadRequest, resp)
	}

	err = s.db.ChangeUsername(body.UserId, *body.Username)
	if err != nil {
		log.Println(err)
		resp["name"] = "username"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChangeDisplayName(c echo.Context) error {
	resp := make(map[string]any)

	body := new(ChangeInformations)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "display_name"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.ChangeDisplayName(body.UserId, *body.DisplayName)
	if err != nil {
		log.Println(err)
		resp["name"] = "display_name"
		resp["message"] = "An error occured when changing your informations."
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}
