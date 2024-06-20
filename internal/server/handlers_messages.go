package server

import (
	"encoding/json"
	"fmt"
	"goback/internal/models"
	"goback/internal/utils"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/lxzan/gws"
)

type CreateMessage struct {
	Author         models.User     `json:"author"`
	ChannelId      string          `json:"channel_id"`
	Content        json.RawMessage `json:"content"`
	PrivateMessage bool            `json:"private_message"`
	ServerId       string          `json:"server_id,omitempty"`
}

func (s *Server) HandlerPrivateMessages(c echo.Context) error {
	resp := make(map[string]any)

	channelId := c.Param("channelId")
	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	messages, err := s.db.GetPrivateMessages(userId, channelId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["messages"] = messages

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerChannelMessages(c echo.Context) error {
	resp := make(map[string]any)

	channelId := c.Param("channelId")

	messages, err := s.db.GetChannelMessages(channelId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["messages"] = messages

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerSendMessage(c echo.Context) error {
	resp := make(map[string]any)

	body := new(CreateMessage)
	bodyStr := c.FormValue("body")
	if err := json.Unmarshal([]byte(bodyStr), body); err != nil {
		log.Println("Error parsing JSON body:", err)
		resp["message"] = "An error occurred when parsing your message."
		return c.JSON(http.StatusBadRequest, resp)
	}

	message := models.Message{
		Author:    body.Author,
		ChannelId: body.ChannelId,
		Content:   body.Content,
		Edited:    false,
		Images:    make([]string, 0),
	}

	form, err := c.MultipartForm()
	if err != nil {
		log.Println("Error parsing form data:", err)
		resp["message"] = "An error occurred when parsing your files."
		return c.JSON(http.StatusBadRequest, resp)
	}

	files := form.File
	if len(files) > 0 {
		var wg sync.WaitGroup
		for _, fileHeader := range files {
			for _, file := range fileHeader {
				wg.Add(1)
				go func(message *models.Message, file *multipart.FileHeader) {
					defer wg.Done()
					if file.Size > 8*1024*1024 {
						resp["error"] = "File size exceeds 8MB limit"
					}

					src, err := file.Open()
					if err != nil {
						log.Println(err)
						resp["error"] = "File size exceeds 8MB limit"
						return
					}
					defer src.Close()

					randId, _ := utils.GenerateRandomId(10)
					imageKey := randId + "-" + file.Filename
					_, err = s.s3.PutObject(&s3.PutObjectInput{
						Bucket: aws.String("Hudori"),
						Key:    aws.String(imageKey),
						Body:   src,
					})
					if err != nil {
						resp["error"] = "File size exceeds 8MB limit"
						return
					}

					message.Images = append(message.Images, os.Getenv("B2_URL")+imageKey)
				}(&message, file)
			}
		}
		wg.Wait()
	}

	mess, err := s.db.CreateMessage(message)
	if err != nil {
		log.Println("error when creating a message", err)
		resp["message"] = "An error occured when sending your message."

		return c.JSON(http.StatusBadRequest, resp)
	}

	if body.PrivateMessage {
		id, _ := utils.GenerateRandomId(10)
		notif := models.MessageNotif{
			ID:        id,
			Type:      "new_message",
			Counter:   1,
			UserId:    "users:" + body.ChannelId,
			ChannelId: "users:" + strings.Split(body.Author.ID, ":")[1],
		}
		wsMess := models.WSMessage{
			Type:    "text_message",
			Content: mess,
			Notif:   notif,
		}
		data, err := json.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}

		if conn, ok := s.ws.sessions.Load(strings.Split(body.Author.ID, ":")[1]); ok {
			conn.WriteMessage(gws.OpcodeText, data)
		}

		connFriend, ok := s.ws.sessions.Load(body.ChannelId)
		if ok {
			connFriend.WriteMessage(gws.OpcodeText, data)
		}
	} else {
		id, _ := utils.GenerateRandomId(10)
		notif := models.MessageNotif{
			ID:        id,
			Type:      "new_message",
			UserId:    body.Author.ID,
			ChannelId: "channels:" + body.ChannelId,
			ServerId:  body.ServerId,
		}
		wsMess := models.WSMessage{
			Type:    "text_message",
			Content: mess,
			Notif:   notif,
		}
		data, err := json.Marshal(wsMess)
		if err != nil {
			log.Println(err)
			return err
		}
		Pub(globalEmitter, "channels:"+body.ChannelId, gws.OpcodeText, data)
	}

	return c.JSON(http.StatusOK, resp)
}
