package server

import (
	"math/rand"

	"github.com/labstack/echo/v4"
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
	socket.Session().Store("userIdMain", userIdMain)
	socket.Session().Store("userIdEmitter", rand.Int63())

	go func() {
		socket.ReadLoop()
	}()

	// Sub(globalEmitter, "event", socket)
	// Pub(globalEmitter, "event", gws.OpcodeText, []byte("New user connected"))

	return nil
}
