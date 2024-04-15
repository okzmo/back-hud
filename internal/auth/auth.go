package auth

import (
	"fmt"
	"log"
	"os"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/google"
)

type Service interface {
	StoreUserSession(c echo.Context, user goth.User) error
	GetUserSession(c echo.Context) (goth.User, error)
	RemoveUserSession(c echo.Context)
}

type service struct{}

func New(store sessions.Store) Service {
	gothic.Store = store

	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_KEY"), os.Getenv("GOOGLE_SECRET"), "http://localhost:8080/auth/google/callback"),
		apple.New(os.Getenv("APPLE_KEY"), os.Getenv("APPLE_SECRET"), "http://localhost:8080/auth/apple/callback", nil, apple.ScopeName, apple.ScopeEmail),
	)

	return &service{}
}

func (s *service) GetUserSession(c echo.Context) (goth.User, error) {
	session, err := gothic.Store.Get(c.Request(), "session")
	if err != nil {
		return goth.User{}, err
	}

	u := session.Values["user"]
	if u == nil {
		return goth.User{}, fmt.Errorf("user is not authenticated! %v", u)
	}

	return u.(goth.User), nil
}

func (s *service) StoreUserSession(c echo.Context, user goth.User) error {
	session, _ := gothic.Store.Get(c.Request(), "session")

	session.Values["user"] = user

	err := session.Save(c.Request(), c.Response().Writer)
	if err != nil {
		c.Error(err)
		return err
	}

	return nil
}

func (s *service) RemoveUserSession(c echo.Context) {
	session, err := gothic.Store.Get(c.Request(), "session")
	if err != nil {
		log.Println(err)
		c.Error(err)
		return
	}

	session.Values["user"] = goth.User{}
	session.Options.MaxAge = -1
	session.Save(c.Request(), c.Response().Writer)
}
