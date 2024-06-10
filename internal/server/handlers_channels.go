package server

import (
	"context"
	"encoding/json"
	"goback/internal/models"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	"github.com/lxzan/gws"
)

type createChannelBody struct {
	Name         string `json:"name"`
	ChannelType  string `json:"channel_type"`
	CategoryName string `json:"category_name"`
	ServerId     string `json:"server_id"`
}

type removeChannelBody struct {
	ChannelId    string `json:"channel_id"`
	CategoryName string `json:"category_name"`
	ServerId     string `json:"server_id"`
}

type categoryBody struct {
	CategoryName string `json:"category_name"`
	ServerId     string `json:"server_id"`
}

func (s *Server) HandlerCreateChannel(c echo.Context) error {
	resp := make(map[string]any)

	body := new(createChannelBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	channelAndMembers, err := s.db.CreateChannel(body.ServerId, body.CategoryName, body.ChannelType, body.Name)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	for _, member := range channelAndMembers.Members {
		if conn, ok := s.ws.sessions.Load(strings.Split(member, ":")[1]); ok {
			Sub(globalEmitter, channelAndMembers.Channel.ID, &Socket{conn})
		}
	}

	resp["message"] = "success"

	wsContent := make(map[string]any)
	wsContent["channel"] = channelAndMembers.Channel
	wsContent["category_name"] = body.CategoryName

	wsMess := models.WSMessage{
		Type:    "create_channel",
		Content: wsContent,
	}
	data, err := json.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}
	Pub(globalEmitter, body.ServerId, gws.OpcodeText, data)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerDeleteChannel(c echo.Context) error {
	resp := make(map[string]any)

	body := new(removeChannelBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.RemoveChannel(body.ServerId, body.CategoryName, body.ChannelId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["message"] = "success"

	wsContent := make(map[string]any)
	wsContent["channel_id"] = body.ChannelId
	wsContent["category_name"] = body.CategoryName

	wsMess := models.WSMessage{
		Type:    "delete_channel",
		Content: wsContent,
	}
	data, err := json.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}
	Pub(globalEmitter, body.ServerId, gws.OpcodeText, data)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerCreateCategory(c echo.Context) error {
	resp := make(map[string]any)

	body := new(categoryBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.CreateCategory(body.ServerId, body.CategoryName)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["message"] = "success"

	wsMess := models.WSMessage{
		Type:    "create_category",
		Content: body.CategoryName,
	}
	data, err := json.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	Pub(globalEmitter, body.ServerId, gws.OpcodeText, data)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerDeleteCategory(c echo.Context) error {
	resp := make(map[string]any)

	body := new(categoryBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	channels, err := s.db.RemoveCategory(body.ServerId, body.CategoryName)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["message"] = "success"

	wsMess := models.WSMessage{
		Type:    "delete_category",
		Content: body.CategoryName,
	}
	data, err := json.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}
	Pub(globalEmitter, body.ServerId, gws.OpcodeText, data)

	res, _ := s.rtc.ListRooms(context.Background(), &livekit.ListRoomsRequest{
		Names: channels,
	})
	for _, v := range res.Rooms {
		s.rtc.DeleteRoom(context.Background(), &livekit.DeleteRoomRequest{
			Room: v.Name,
		})
	}
	return c.JSON(http.StatusOK, resp)
}
