package server

import (
	"fmt"
	"goback/internal/models"
	"goback/internal/utils"
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/labstack/echo/v4"
)

func (s *Server) HelloWorldHandler(c echo.Context) error {
	resp := map[string]string{
		"message": "Hello World",
	}

	return c.JSON(http.StatusOK, resp)
}

// func (s *Server) healthHandler(c echo.Context) error {
// 	return c.JSON(http.StatusOK, s.db.Health())
// }

// AUTH
type UserBodySignup struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

func (s *Server) HandlerSignUp(c echo.Context) error {
	resp := make(map[string]any)

	body := new(UserBodySignup)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when creating your account."

		return c.JSON(http.StatusBadRequest, resp)
	}

	if !utils.EmailValid(body.Email) {
		log.Println("invalid email")
		resp["name"] = "email"
		resp["message"] = "The format of the email is invalid."

		return c.JSON(http.StatusConflict, resp)
	}

	_, err := s.db.GetUser("", "", body.Email)
	if err == nil {
		resp["name"] = "email"
		resp["message"] = "This email is unavailable."

		return c.JSON(http.StatusConflict, resp)
	}

	_, err = s.db.GetUser("", body.Username, "")
	if err == nil {
		resp["name"] = "username"
		resp["message"] = "This username is unavailable."

		return c.JSON(http.StatusConflict, resp)
	}

	hashedPassword, err := argon2id.CreateHash(body.Password, argon2id.DefaultParams)
	if err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when creating your account."

		return c.JSON(http.StatusBadRequest, resp)
	}

	userCreated := models.User{
		Email:       body.Email,
		Password:    hashedPassword,
		Username:    body.Username,
		DisplayName: body.DisplayName,
		Avatar:      "avatar",
		Banner:      "banner",
		Status:      "Online",
		AboutMe:     "",
	}

	userId, err := s.db.CreateUser(userCreated)
	if err != nil {
		log.Println("error when creating the user", err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when creating your account."

		return c.JSON(http.StatusBadRequest, resp)
	}

	sessionCreated := models.Session{
		IpAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		UserId:    userId,
	}

	sess, err := s.db.CreateSession(sessionCreated)
	if err != nil {
		log.Println("error when creating a session after creating account:", err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when trying to connect to your account."

		return c.JSON(http.StatusBadRequest, resp)
	}

	sessionExpire, error := time.Parse(time.RFC3339, sess.ExpiresdAt)
	if error != nil {
		log.Println("error when creating the user session", err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured on sign in."

		return error
	}

	session := new(http.Cookie)
	session.Name = "session"
	session.Path = "/"
	session.Value = sess.ID
	session.Expires = sessionExpire
	session.HttpOnly = true
	session.Secure = false
	c.SetCookie(session)

	resp["message"] = "success"
	resp["user"] = map[string]string{
		"username":    userCreated.Username,
		"displayName": userCreated.DisplayName,
		"avatar":      userCreated.Avatar,
		"banner":      userCreated.Banner,
		"status":      userCreated.Status,
	}

	return c.JSON(http.StatusOK, resp)
}

type UserBodySignIn struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) HandlerSignIn(c echo.Context) error {
	resp := make(map[string]any)

	body := new(UserBodySignIn)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when creating your account."

		return c.JSON(http.StatusBadRequest, resp)
	}

	user, err := s.db.GetUser("", "", body.Email)
	if err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "Please check your login information and try again."

		return c.JSON(http.StatusBadRequest, resp)
	}

	match, err := argon2id.ComparePasswordAndHash(body.Password, user.Password)
	if err != nil || !match {
		log.Println(err, match)
		resp["name"] = "unexpected"
		resp["message"] = "Please check your login information and try again."
		fmt.Println(resp)

		return c.JSON(http.StatusBadRequest, resp)
	}

	sessionCreated := models.Session{
		IpAddress: c.RealIP(),
		UserAgent: c.Request().UserAgent(),
		UserId:    user.ID,
	}

	sess, err := s.db.CreateSession(sessionCreated)
	if err != nil {
		log.Println("error when creating the user session", err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured on sign in."

		return c.JSON(http.StatusBadRequest, resp)
	}

	sessionExpire, error := time.Parse(time.RFC3339, sess.ExpiresdAt)
	if error != nil {
		log.Println("error when creating the user session", err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured on sign in."

		return error
	}

	session := new(http.Cookie)
	session.Name = "session"
	session.Value = sess.ID
	session.Path = "/"
	session.Expires = sessionExpire
	session.HttpOnly = true
	session.Secure = false
	c.SetCookie(session)

	resp["message"] = "success"
	resp["user"] = map[string]string{
		"username":    user.Username,
		"displayName": user.DisplayName,
		"avatar":      user.Avatar,
		"banner":      user.Banner,
		"status":      user.Status,
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerVerify(c echo.Context) error {
	resp := make(map[string]any)

	sessionCookie, err := c.Cookie("session")
	if err != nil {
		resp["message"] = "No session cookie available."

		return c.JSON(http.StatusNotFound, resp)
	}

	session, err := s.db.GetSession(sessionCookie.Value)
	if err != nil {
		log.Println(err)
		resp["message"] = "No session related to given id."

		return c.JSON(http.StatusNotFound, resp)
	}

	user, err := s.db.GetUser(session.UserId, "", "")
	if err != nil {
		log.Println(err)
		resp["message"] = "No user match the given id from session."

		return c.JSON(http.StatusNotFound, resp)
	}

	resp["message"] = "success"
	resp["user"] = map[string]string{
		"username":    user.Username,
		"displayName": user.DisplayName,
		"avatar":      user.Avatar,
		"banner":      user.Banner,
		"status":      user.Status,
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerFriends(c echo.Context) error {
	resp := make(map[string]any)

	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	friends, err := s.db.GetFriends(userId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["friends"] = friends

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerUsersIdFromChannel(c echo.Context) error {
	resp := make(map[string]any)

	channelId := fmt.Sprintf("channels:%s", c.Param("channelId"))

	users, err := s.db.GetUsersFromChannel(channelId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["users"] = users

	return c.JSON(http.StatusOK, resp)
}
