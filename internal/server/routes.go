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
		AllowOrigins:     []string{"https://localhost:5173", "http://localhost:5173", "http://localhost:4173", "https://hudori.app"},
		AllowMethods:     []string{http.MethodGet, http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete},
		AllowCredentials: true,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderSetCookie, echo.HeaderCookie, echo.HeaderContentType, echo.HeaderAccept},
	}
	e.Use(middleware.CORSWithConfig(CORSConfig))

	// Auth
	auth := e.Group("/auth")
	auth.POST("/signup", s.HandlerSignUp)
	auth.POST("/signin", s.HandlerSignIn)
	auth.GET("/verify", s.HandlerVerify)

	// auth.GET("/:provider", s.ProviderLoginHandler)
	// auth.GET("/:provider/callback", s.AuthCallbackHandler)
	// auth.GET("/logout/:provider", s.LogoutHandler)

	e.GET("/ws/:userId", s.HandlerWebsocket)
	api := e.Group("/api/v1", s.SessionAuthMiddleware)

	api.GET("/friends/:userId", s.HandlerFriends)
	api.POST("/friends/add", s.HandlerAddFriend)
	api.POST("/friends/accept", s.HandlerAcceptFriend)
	api.POST("/friends/refuse", s.HandlerRefuseFriend)
	api.POST("/friends/delete", s.HandlerRemoveFriend)

	api.GET("/servers/:userId", s.HandlerUserServers)
	api.GET("/server/:userId/:serverId", s.HandlerServerInformations)
	api.POST("/server/join", s.HandlerJoinServer)
	api.POST("/server/create", s.HandlerCreateServer)
	api.POST("/server/delete", s.HandlerDeleteServer)
	api.POST("/server/leave", s.HandlerLeaveServer)

	api.GET("/messages/:channelId/private/:userId", s.HandlerPrivateMessages)
	api.GET("/messages/:channelId", s.HandlerChannelMessages)
	api.POST("/messages/create", s.HandlerSendMessage)

	api.GET("/channels/:channelId/users", s.HandlerUsersIdFromChannel)
	api.POST("/channels/create", s.HandlerCreateChannel)
	api.POST("/channels/delete", s.HandlerDeleteChannel)

	api.POST("/category/create", s.HandlerCreateCategory)
	api.POST("/category/delete", s.HandlerDeleteCategory)

	api.GET("/notifications/:userId", s.HandlerNotifications)

	api.POST("/invites/create", s.HandlerCreateInvitation)

	api.GET("/rtc/:room/:identity", s.HandlerGenerateRTCToken)

	return e
}
