package server

import (
	"context"
	"goback/internal/utils"
	"goback/proto/protoMess"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	"github.com/lxzan/gws"
	"google.golang.org/protobuf/proto"
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

type typingBody struct {
	DisplayName string `json:"display_name"`
	UserId      string `json:"user_id"`
	ChannelId   string `json:"channel_id"`
	Status      string `json:"status"`
}

func (s *Server) HandlerCreateChannel(c echo.Context) error {
	resp := make(map[string]any)

	body := new(createChannelBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when creating the channel."

		return c.JSON(http.StatusBadRequest, resp)
	}

	channelAndMembers, err := s.db.CreateChannel(body.ServerId, body.CategoryName, body.ChannelType, body.Name)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when creating the channel."
		return c.JSON(http.StatusNotFound, resp)
	}

	for _, member := range channelAndMembers.Members {
		if conn, ok := s.ws.sessions.Load(strings.Split(member, ":")[1]); ok {
			Sub(globalEmitter, channelAndMembers.Channel.ID, &Socket{conn})
		}
	}

	resp["message"] = "success"

	channelObj := &protoMess.Channel{
		Id:        channelAndMembers.Channel.ID,
		Name:      channelAndMembers.Channel.Name,
		Type:      channelAndMembers.Channel.Type,
		Private:   channelAndMembers.Channel.Private,
		CreatedAt: channelAndMembers.Channel.CreatedAt,
	}

	wsMess := &protoMess.WSMessage{
		Type: "create_channel",
		Content: &protoMess.WSMessage_Channel{
			Channel: &protoMess.CreateChannel{
				ServerId:     body.ServerId,
				Channel:      channelObj,
				CategoryName: body.CategoryName,
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	Pub(globalEmitter, body.ServerId, gws.OpcodeBinary, compMess)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerDeleteChannel(c echo.Context) error {
	resp := make(map[string]any)

	body := new(removeChannelBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when deleting the channel."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.RemoveChannel(body.ServerId, body.CategoryName, body.ChannelId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when deleting the channel."
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["message"] = "success"

	wsContent := make(map[string]any)
	wsContent["channel_id"] = body.ChannelId
	wsContent["category_name"] = body.CategoryName

	wsMess := &protoMess.WSMessage{
		Type: "delete_channel",
		Content: &protoMess.WSMessage_Delchannel{
			Delchannel: &protoMess.DeleteChannel{
				ServerId:     body.ServerId,
				ChannelId:    body.ChannelId,
				CategoryName: body.CategoryName,
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	Pub(globalEmitter, body.ServerId, gws.OpcodeBinary, compMess)

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerCreateCategory(c echo.Context) error {
	resp := make(map[string]any)

	body := new(categoryBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when creating the category."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.CreateCategory(body.ServerId, body.CategoryName)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["message"] = "success"

	wsMess := &protoMess.WSMessage{
		Type: "create_category",
		Content: &protoMess.WSMessage_CreateCategory{
			CreateCategory: &protoMess.CreateCategory{
				ServerId:     body.ServerId,
				CategoryName: body.CategoryName,
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	Pub(globalEmitter, body.ServerId, gws.OpcodeBinary, compMess)

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

	wsMess := &protoMess.WSMessage{
		Type: "delete_category",
		Content: &protoMess.WSMessage_DeleteCategory{
			DeleteCategory: &protoMess.DeleteCategory{
				ServerId:     body.ServerId,
				CategoryName: body.CategoryName,
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	Pub(globalEmitter, body.ServerId, gws.OpcodeBinary, compMess)

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

func (s *Server) HandlerTyping(c echo.Context) error {
	resp := make(map[string]any)

	body := new(typingBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured on typing indicator."

		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	wsMess := &protoMess.WSMessage{
		Type: "typing",
		Content: &protoMess.WSMessage_Typing{
			Typing: &protoMess.Typing{
				UserId:      body.UserId,
				DisplayName: body.DisplayName,
				ChannelId:   body.ChannelId,
				Status:      body.Status,
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	Pub(globalEmitter, "channels:"+body.ChannelId, gws.OpcodeBinary, compMess)

	return c.JSON(http.StatusOK, resp)
}
