package server

import (
	"fmt"
	"goback/internal/auth"
	"goback/internal/database"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	lksdk "github.com/livekit/server-sdk-go/v2"
	"github.com/lxzan/event_emitter"
)

type Server struct {
	port int
	auth auth.Service
	db   database.Service
	ws   *Websocket
	rtc  *lksdk.RoomServiceClient
}

var globalEmitter = event_emitter.New[*Socket](&event_emitter.Config{
	BucketNum:  16,
	BucketSize: 128,
})

func NewServer() *http.Server {
	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		panic(err)
	}

	sessionStore := auth.NewCookieStore(auth.SessionOptions{
		CookiesKey: os.Getenv("SESSION_SECRET"),
		MaxAge:     86400 * 30,
		Secure:     false,
		HttpOnly:   true,
	})

	NewServer := &Server{
		port: port,
		auth: auth.New(sessionStore),
		db:   database.New(),
		ws:   NewWebsocket(),
		rtc:  NewRTC(),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		// TLSConfig:    tlsConfig,
	}

	return server
}
