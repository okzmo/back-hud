package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (s *Server) RegisterRoutes() http.Handler {

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	e.GET("/", s.HelloWorldHandler)
	e.GET("/health", s.healthHandler)

	// Auth
	e.GET("/auth/:provider", s.ProviderLoginHandler)
	e.GET("/auth/:provider/callback", s.AuthCallbackHandler)
	e.GET("/auth/logout/:provider", s.LogoutHandler)

	return e
}
