package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"goback/internal/auth"
	"goback/internal/database"
)

type Server struct {
	port int
	auth auth.Service
	db   database.Service
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

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
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
