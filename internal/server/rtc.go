package server

import (
	"os"

	lksdk "github.com/livekit/server-sdk-go/v2"
)

func NewRTC() *lksdk.RoomServiceClient {
	livekitURL := os.Getenv("LIVEKIT_URL")
	apiKey := os.Getenv("LIVEKIT_KEY")
	apiSecret := os.Getenv("LIVEKIT_SECRET")
	roomClient := lksdk.NewRoomServiceClient(livekitURL, apiKey, apiSecret)

	return roomClient
}
