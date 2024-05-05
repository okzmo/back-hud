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
	CORSConfig := middleware.CORSConfig{
		Skipper:          middleware.DefaultSkipper,
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowCredentials: true,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderSetCookie, echo.HeaderCookie, echo.HeaderContentType, echo.HeaderAccept},
	}
	e.Use(middleware.CORSWithConfig(CORSConfig))

	e.GET("/", s.HelloWorldHandler)
	// e.GET("/health", s.healthHandler)

	// Auth
	auth := e.Group("/auth")
	auth.POST("/signup", s.HandlerSignUp)
	auth.POST("/signin", s.HandlerSignIn)
	auth.GET("/verify", s.HandlerVerify)

	// auth.GET("/:provider", s.ProviderLoginHandler)
	// auth.GET("/:provider/callback", s.AuthCallbackHandler)
	// auth.GET("/logout/:provider", s.LogoutHandler)

	api := e.Group("/api/v1")
	api.GET("/friends/:userId", s.HandlerFriends)
	api.GET("/channels/:channelId/users", s.HandlerUsersIdFromChannel)

	return e
}
