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
	// e.GET("/health", s.healthHandler)

	// Auth
	auth := e.Group("/auth")
	auth.POST("/signup", s.HandlerSignUp)
	auth.POST("/signin", s.HandlerSignIn)
	auth.POST("/verify", s.HandlerVerify)

	// auth.GET("/:provider", s.ProviderLoginHandler)
	// auth.GET("/:provider/callback", s.AuthCallbackHandler)
	// auth.GET("/logout/:provider", s.LogoutHandler)

	// api := e.Group("/api/v1")

	return e
}
