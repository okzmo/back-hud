package server

import (
	"fmt"
	"goback/internal/utils"
	"goback/proto/protoMess"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lxzan/gws"
	"google.golang.org/protobuf/proto"
)

type addFriendBody struct {
	InitiatorId       string `json:"initiator_id"`
	InitiatorUsername string `json:"initiator_username"`
	ReceiverUsername  string `json:"receiver_username"`
}

type acceptFriendBody struct {
	RequestId string `json:"request_id"`
	NotifId   string `json:"id"`
}

type removeFriendBody struct {
	UserId   string `json:"user_id"`
	ServerId string `json:"space_id"`
	FriendId string `json:"friend_id"`
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

func (s *Server) HandlerAddFriend(c echo.Context) error {
	resp := make(map[string]any)

	body := new(addFriendBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when adding your friend."

		return c.JSON(http.StatusBadRequest, resp)
	}

	notif, err := s.db.RelateFriends(body.InitiatorId, body.InitiatorUsername, body.ReceiverUsername)
	if err != nil {
		log.Println("error when relating users:", err)
		resp["name"] = "unexpected"
		resp["message"] = err.Error()

		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(notif.UserId, ":")[1]); ok {
		mess := &protoMess.WSMessage{
			Type: "friend_request",
			Content: &protoMess.WSMessage_FriendRequest{
				FriendRequest: &protoMess.FriendRequest{
					Id:          notif.ID,
					InitiatorId: notif.InitiatorId,
					RequestId:   notif.RequestId,
					Message:     notif.Message,
					Type:        notif.Type,
					UserId:      notif.UserId,
					CreatedAt:   notif.CreatedAt,
				},
			},
		}

		data, err := proto.Marshal(mess)
		if err != nil {
			log.Println(err)
			return err
		}

		compMess := utils.CompressMess(data)
		conn.WriteMessage(gws.OpcodeBinary, compMess)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerAcceptFriend(c echo.Context) error {
	resp := make(map[string]any)

	body := new(acceptFriendBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when accepting friend request."

		return c.JSON(http.StatusBadRequest, resp)
	}

	allInfos, err := s.db.AcceptFriend(body.RequestId, body.NotifId)
	if err != nil {
		log.Println("error when accepting friend request", err)
		resp["message"] = "An error occured when accepting friend request."

		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(allInfos.Initiator.ID, ":")[1]); ok {
		for _, channel := range allInfos.ServerChannels {
			Sub(globalEmitter, channel, &Socket{conn})
		}
		Sub(globalEmitter, allInfos.Server.ID, &Socket{conn})

		mess := &protoMess.WSMessage{
			Type: "friend_accept",
			Content: &protoMess.WSMessage_FriendAccept{
				FriendAccept: &protoMess.FriendAccept{
					User: &protoMess.User{
						Id:          allInfos.Receiver.ID,
						DisplayName: allInfos.Receiver.DisplayName,
						Avatar:      allInfos.Receiver.Avatar,
						AboutMe:     allInfos.Receiver.AboutMe,
						Status:      allInfos.Receiver.Status,
					},
					Server: &protoMess.Server{
						Id:        allInfos.Server.ID,
						Name:      allInfos.Server.Name,
						Banner:    allInfos.Server.Banner,
						Type:      allInfos.Server.Type,
						CreatedAt: allInfos.Server.CreatedAt,
					},
				},
			},
		}

		if conn, ok := s.ws.sessions.Load(strings.Split(allInfos.Receiver.ID, ":")[1]); ok {
			for _, channel := range allInfos.ServerChannels {
				Sub(globalEmitter, channel, &Socket{conn})
			}
			Sub(globalEmitter, allInfos.Server.ID, &Socket{conn})
		}

		data, err := proto.Marshal(mess)
		if err != nil {
			log.Println(err)
			return err
		}

		compMess := utils.CompressMess(data)
		conn.WriteMessage(gws.OpcodeBinary, compMess)
	}

	resp["message"] = "success"
	resp["friend"] = allInfos.Initiator
	resp["server"] = allInfos.Server

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerRefuseFriend(c echo.Context) error {
	resp := make(map[string]any)

	body := new(acceptFriendBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when refusing friend request."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.RefuseFriend(body.RequestId, body.NotifId)
	if err != nil {
		log.Println("error when refusing friend request", err)
		resp["message"] = "An error occured when refusing friend request."

		return c.JSON(http.StatusBadRequest, resp)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) HandlerRemoveFriend(c echo.Context) error {
	resp := make(map[string]any)

	body := new(removeFriendBody)
	if err := c.Bind(body); err != nil {
		log.Println(err)
		resp["message"] = "An error occured when removing your friend."

		return c.JSON(http.StatusBadRequest, resp)
	}

	err := s.db.RemoveFriend(body.UserId, body.ServerId, body.FriendId)
	if err != nil {
		log.Println("error when refusing friend request", err)
		resp["message"] = "An error occured when removing your friend."

		return c.JSON(http.StatusBadRequest, resp)
	}

	if conn, ok := s.ws.sessions.Load(strings.Split(body.FriendId, ":")[1]); ok {
		mess := &protoMess.WSMessage{
			Type: "friend_remove",
			Content: &protoMess.WSMessage_UserId{
				UserId: body.UserId,
			},
		}

		data, err := proto.Marshal(mess)
		if err != nil {
			log.Println(err)
			return err
		}

		compMess := utils.CompressMess(data)
		conn.WriteMessage(gws.OpcodeBinary, compMess)
	}

	resp["message"] = "success"

	return c.JSON(http.StatusOK, resp)
}
