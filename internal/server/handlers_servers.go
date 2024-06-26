package server

import (
	"context"
	"fmt"
	"goback/internal/models"
	"goback/internal/utils"
	"goback/proto/protoMess"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/livekit/protocol/livekit"
	"github.com/lxzan/gws"
	"google.golang.org/protobuf/proto"
)

type JoinServerBody struct {
	User     models.User `json:"user"`
	InviteId string      `json:"invite_id"`
}

type CreateServerBody struct {
	UserId string `json:"user_id"`
	Name   string `json:"name"`
}

type GeneralServerBody struct {
	UserId   string `json:"user_id"`
	ServerId string `json:"server_id"`
}

// Servers
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

func (s *Server) HandlerUserServers(c echo.Context) error {
	resp := make(map[string]any)

	userId := fmt.Sprintf("users:%s", c.Param("userId"))

	servers, err := s.db.GetUserServers(userId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	resp["servers"] = servers

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) getParticipants(channel *models.Channel) ([]models.User, error) {
	res, err := s.rtc.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
	if err != nil {
		return nil, err
	}

	var participants []models.User

	for _, r := range res.Rooms {
		if r.Name == channel.ID {
			res, err := s.rtc.ListParticipants(context.Background(), &livekit.ListParticipantsRequest{
				Room: channel.ID,
			})
			if err != nil {
				return nil, err
			}

			for _, user := range res.Participants {
				userDb, err := s.db.GetUser(user.Identity, "", "")
				if err != nil {
					return nil, err
				}

				participants = append(participants, userDb)
			}
		}
	}

	return participants, nil
}

func (s *Server) HandlerServerInformations(c echo.Context) error {
	resp := make(map[string]any)

	userId := fmt.Sprintf("users:%s", c.Param("userId"))
	serverId := fmt.Sprintf("servers:%s", c.Param("serverId"))

	server, err := s.db.GetServer(userId, serverId)
	if err != nil {
		resp["message"] = err
		return c.JSON(http.StatusNotFound, resp)
	}

	var wg sync.WaitGroup
	for _, cat := range server.Categories {
		for i := range cat.Channels {
			wg.Add(1)
			go func(channel *models.Channel) {
				defer wg.Done()
				participants, err := s.getParticipants(channel)
				if err != nil {
					log.Print("error on getting participants", err)
				}
				channel.Participants = participants
			}(&cat.Channels[i])
		}
	}

	wg.Wait()
	resp["server"] = server

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerJoinServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(JoinServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	server, err := s.db.JoinServer(body.User.ID, body.InviteId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(body.User.ID, ":")[1]); ok {
		for _, channel := range server.ServerChannels {
			Sub(globalEmitter, channel, &Socket{conn})
		}
		Sub(globalEmitter, server.Server.ID, &Socket{conn})
	}

	wsMess := &protoMess.WSMessage{
		Type: "join_server",
		Content: &protoMess.WSMessage_JoinServer{
			JoinServer: &protoMess.JoinServer{
				User: &protoMess.User{
					Id:            body.User.ID,
					Username:      body.User.Username,
					DisplayName:   body.User.DisplayName,
					UsernameColor: body.User.UsernameColor,
					Avatar:        body.User.Avatar,
				},
				ServerId: server.Server.ID,
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	Pub(globalEmitter, server.Server.ID, gws.OpcodeBinary, compMess)

	resp["server"] = server.Server

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerCreateServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(CreateServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when joining the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	server, err := s.db.CreateServer(body.UserId, body.Name)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(body.UserId, ":")[1]); ok {
		for _, channel := range server.ServerChannels {
			Sub(globalEmitter, channel, &Socket{conn})
		}
		Sub(globalEmitter, server.Server.ID, &Socket{conn})
	}

	resp["server"] = server.Server

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerDeleteServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(GeneralServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when deleting the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.DeleteServer(body.UserId, body.ServerId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["success"] = true

	wsMess := &protoMess.WSMessage{
		Type: "delete_server",
		Content: &protoMess.WSMessage_ServerId{
			ServerId: body.ServerId,
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

func (s *Server) HandlerLeaveServer(c echo.Context) error {
	resp := make(map[string]any)

	body := new(GeneralServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when leaving the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.LeaveServer(body.UserId, body.ServerId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["success"] = true

	wsMess := &protoMess.WSMessage{
		Type: "leave_server",
		Content: &protoMess.WSMessage_QuitServer{
			QuitServer: &protoMess.QuitServer{
				ServerId: body.ServerId,
				UserId:   body.UserId,
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

func (s *Server) HandlerCreateInvitation(c echo.Context) error {
	resp := make(map[string]any)

	body := new(GeneralServerBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["name"] = "unexpected"
		resp["message"] = "An error occured when leaving the server."

		return c.JSON(http.StatusBadRequest, resp)
	}

	invitationId, err := s.db.CreateInvitation(body.UserId, body.ServerId)
	if err != nil {
		resp["name"] = "unexpected"
		resp["message"] = err.Error()
		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["id"] = invitationId

	return c.JSON(http.StatusOK, resp)
}
