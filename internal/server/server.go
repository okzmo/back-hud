package server

import (
	"crypto/tls"
	"fmt"
	"goback/internal/auth"
	"goback/internal/database"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
	s3   *s3.S3
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

	// S3 session
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(os.Getenv("B2_ID"), os.Getenv("B2_KEY"), ""),
		Endpoint:         aws.String(os.Getenv("B2_ENDPOINT")),
		Region:           aws.String(os.Getenv("B2_REGION")),
		S3ForcePathStyle: aws.Bool(true),
	}

	newSession, err := session.NewSession(s3Config)
	if err != nil {
		fmt.Printf("Failed to create a new session s3")
	}

	s3Client := s3.New(newSession)

	NewServer := &Server{
		port: port,
		auth: auth.New(sessionStore),
		db:   database.New(),
		ws:   NewWebsocket(),
		rtc:  NewRTC(),
		s3:   s3Client,
	}
	environment := os.Getenv("ENVIRONMENT")

	var tlsConfig *tls.Config
	if environment == "DEV" {
		serverTLSCert, err := tls.LoadX509KeyPair("cert/cert.pem", "cert/key.pem")
		if err != nil {
			log.Fatalf("Error loading certificate and key file: %v", err)
		}
		tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{serverTLSCert},
		}
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		TLSConfig:    tlsConfig,
	}

	return server
}
