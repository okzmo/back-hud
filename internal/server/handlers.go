package server

import (
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/markbates/goth/gothic"
)

func (s *Server) HelloWorldHandler(c echo.Context) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) healthHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, s.db.Health())
}

// AUTH
func (s *Server) ProviderLoginHandler(c echo.Context) error {
	provider := c.Param("provider")

	gothic.GetProviderName = func(req *http.Request) (string, error) {
		return provider, nil
	}

	if _, err := gothic.CompleteUserAuth(c.Response(), c.Request()); err == nil {
		c.Redirect(http.StatusTemporaryRedirect, "http://localhost:5173/chat")
	} else {
		log.Println(err)
		gothic.BeginAuthHandler(c.Response(), c.Request())
	}

	return nil
}

func (s *Server) AuthCallbackHandler(c echo.Context) error {
	provider := c.Param("provider")

	gothic.GetProviderName = func(req *http.Request) (string, error) {
		return provider, nil
	}

	user, err := gothic.CompleteUserAuth(c.Response(), c.Request())
	if err != nil {
		return err
	}

	err = s.auth.StoreUserSession(c, user)
	if err != nil {
		return err
	}

	c.Redirect(http.StatusTemporaryRedirect, "http://localhost:5173/chat")
	return nil
}

func (s *Server) LogoutHandler(c echo.Context) error {
	provider := c.Param("provider")

	gothic.GetProviderName = func(req *http.Request) (string, error) {
		return provider, nil
	}

	err := gothic.Logout(c.Response(), c.Request())
	if err != nil {
		return err
	}

	s.auth.RemoveUserSession(c)

	c.Redirect(http.StatusTemporaryRedirect, "http://localhost:5173/connect")

	return nil
}

func (s *Server) LoginHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, s.db.Health())
}
