package server

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/auth"
)

func (s *Server) HandlerGenerateRTCToken(c echo.Context) error {
	resp := make(map[string]any)

	room := c.Param("room")
	identity := c.Param("identity")

	apiKey := os.Getenv("LIVEKIT_KEY")
	apiSecret := os.Getenv("LIVEKIT_SECRET")

	grant := &auth.VideoGrant{RoomJoin: true, Room: room}
	at := auth.NewAccessToken(apiKey, apiSecret)
	at.AddGrant(grant).SetIdentity(identity)

	token, err := at.ToJWT()
	if err != nil {
		log.Printf("error generating token: %v", err)
		return c.JSON(http.StatusInternalServerError, fmt.Errorf("could not generate token"))
	}

	resp["token"] = token

	return c.JSON(http.StatusOK, resp)
}
