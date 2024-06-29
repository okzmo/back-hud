package server

import (
	"goback/internal/utils"
	"goback/proto/protoMess"
	"log"
	"math/rand"

	"github.com/labstack/echo/v4"
	"github.com/lxzan/gws"
	"google.golang.org/protobuf/proto"
)

// WEBSOCKET
func (s *Server) HandlerWebsocket(c echo.Context) error {
	upgrader := NewWebsocketUpgrader(s.ws)

	so, err := upgrader.Upgrade(c.Response(), c.Request())
	if err != nil {
		return err
	}

	socket := &Socket{so}
	userIdMain := c.Param("userId")
	socket.Conn.Session().Store("userIdMain", userIdMain)
	socket.Conn.Session().Store("userIdEmitter", rand.Int63())

	err = s.db.UpdateUserStatus(userIdMain, "online")
	if err != nil {
		log.Println(err)
	}

	go func() {
		socket.ReadLoop()
	}()

	servers, err := s.db.GetUserServers("users:" + userIdMain)
	if err != nil {
		log.Println(err)
	}
	channels, err := s.db.GetSubscribedChannels(userIdMain)
	if err != nil {
		log.Println(err)
	}

	wsMess := &protoMess.WSMessage{
		Type: "change_status",
		Content: &protoMess.WSMessage_ChangeStatus{
			ChangeStatus: &protoMess.ChangeStatus{
				UserId: "users:" + userIdMain,
				Status: "online",
			},
		},
	}

	data, err := proto.Marshal(wsMess)
	if err != nil {
		log.Println(err)
		return err
	}

	compMess := utils.CompressMess(data)
	for _, server := range servers {
		Sub(globalEmitter, server.ID, socket)
		Pub(globalEmitter, server.ID, gws.OpcodeBinary, compMess)
	}
	for _, channel := range channels {
		Sub(globalEmitter, channel.ID, socket)
	}

	return nil
}
