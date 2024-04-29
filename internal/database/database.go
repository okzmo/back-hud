package database

import (
	"fmt"
	"goback/internal/models"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
	"github.com/surrealdb/surrealdb.go"
)

type Service interface {
	// Health() map[string]string
	CreateUser(user models.User) (string, error)
	CreateSession(session models.Session) (string, error)
	GetUser(id, username, email string) (models.User, error)
	GetSession(id string) (models.Session, error)
}

type service struct {
	db *surrealdb.DB
}

var (
	username  = os.Getenv("DB_USERNAME")
	password  = os.Getenv("DB_PASSWORD")
	namespace = os.Getenv("DB_NAMESPACE")
	database  = os.Getenv("DB_DATABASE")
)

func New() Service {
	db, err := surrealdb.New("ws://localhost:8000/rpc")
	if err != nil {
		panic(err)
	}

	if _, err := db.Signin(map[string]interface{}{
		"user": username,
		"pass": password,
	}); err != nil {
		panic(err)
	}

	if _, err := db.Use(namespace, database); err != nil {
		panic(err)
	}

	s := &service{db: db}
	return s
}

func (s *service) GetUser(id, username, email string) (models.User, error) {
	var data interface{}
	var err error

	if id != "" {
		data, err = s.db.Select(id)
	} else if email != "" {
		data, err = s.db.Query("SELECT * FROM users WHERE email = $email", map[string]interface{}{
			"email": email,
		})
	} else if username != "" {
		data, err = s.db.Query("SELECT * FROM users WHERE username = $username", map[string]interface{}{
			"username": username,
		})
	}

	if err != nil {
		return models.User{}, err
	}

	if username != "" || email != "" {
		var users []models.User
		if ok, err := surrealdb.UnmarshalRaw(data, &users); !ok {
			if err != nil {
				return models.User{}, err
			}

			return models.User{}, fmt.Errorf("no user found")
		}
		return users[0], nil
	}

	var user models.User
	if err := surrealdb.Unmarshal(data, &user); err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s *service) CreateUser(user models.User) (string, error) {
	var users []models.User

	data, err := s.db.Create("users", user)
	if err != nil {
		return "", err
	}

	err = surrealdb.Unmarshal(data, &users)
	if err != nil {
		return "", err
	}

	return users[0].ID, nil
}

func (s *service) CreateSession(session models.Session) (string, error) {
	var sess []models.Session
	data, err := s.db.Create("sessions", session)
	if err != nil {
		return "", err
	}

	err = surrealdb.Unmarshal(data, &sess)
	if err != nil {
		return "", err
	}

	return sess[0].ID, nil
}

func (s *service) GetSession(sessionId string) (models.Session, error) {
	data, err := s.db.Select(sessionId)
	if err != nil {
		return models.Session{}, err
	}

	var session models.Session
	err = surrealdb.Unmarshal(data, &session)
	if err != nil {
		return models.Session{}, err
	}

	return session, nil
}

// func (s *service) Health() map[string]string {
// 	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
// 	defer cancel()
//
// 	err := s.db.PingContext(ctx)
// 	if err != nil {
// 		log.Fatalf(fmt.Sprintf("db down: %v", err))
// 	}
//
// 	return map[string]string{
// 		"message": "It's healthy",
// 	}
// }
