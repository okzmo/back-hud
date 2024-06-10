package server

import (
	"os"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

func NewRTC() *lksdk.RoomServiceClient {
	apiKey := os.Getenv("LIVEKIT_KEY")
	apiSecret := os.Getenv("LIVEKIT_SECRET")
	roomClient := lksdk.NewRoomServiceClient("ws://localhost:7880", apiKey, apiSecret)

	return roomClient
}
