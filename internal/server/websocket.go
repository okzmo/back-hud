package server

import (
	"fmt"
	"time"

	"github.com/lxzan/event_emitter"
	"github.com/lxzan/gws"
)

const (
	PingInterval = 5 * time.Second
	PingWait     = 30 * time.Minute
)

type Websocket struct {
	Emitter  *event_emitter.EventEmitter[*Socket]
	sessions *gws.ConcurrentMap[string, *gws.Conn]
}

func NewWebsocket() *Websocket {
	emitter := event_emitter.New[*Socket](&event_emitter.Config{
		BucketNum:  16,
		BucketSize: 128,
	})
	return &Websocket{
		Emitter:  emitter,
		sessions: gws.NewConcurrentMap[string, *gws.Conn](16),
	}
}

func NewWebsocketUpgrader(handler *Websocket) *gws.Upgrader {
	return gws.NewUpgrader(handler, &gws.ServerOption{
		ParallelEnabled:   true,
		Recovery:          gws.Recovery,
		PermessageDeflate: gws.PermessageDeflate{Enabled: true},
	})
}

func (c *Websocket) getMainUserId(socket *gws.Conn) string {
	userId, _ := socket.Session().Load("userIdMain")
	return userId.(string)
}

func (c *Websocket) OnOpen(socket *gws.Conn) {
	userId := c.getMainUserId(socket)
	if conn, ok := c.sessions.Load(userId); ok {
		conn.WriteClose(1000, []byte("connection has been replaced"))
	}
	_ = socket.SetDeadline(time.Now().Add(PingInterval + PingWait))
	c.sessions.Store(userId, socket)
}

func (c *Websocket) OnClose(socket *gws.Conn, err error) {}

func (c *Websocket) OnPing(socket *gws.Conn, payload []byte) {
	_ = socket.SetDeadline(time.Now().Add(PingInterval + PingWait))
	_ = socket.WriteString("pong")
}

func (c *Websocket) OnPong(socket *gws.Conn, payload []byte) {}

func (c *Websocket) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()
	if b := message.Data.Bytes(); len(b) == 4 && string(b) == "ping" {
		c.OnPing(socket, nil)
		return
	}

	Pub(globalEmitter, "event", gws.OpcodeText, message.Bytes())
}

// EMITTER
type Socket struct{ *gws.Conn }

func (c *Socket) GetSubscriberID() int64 {
	userId, _ := c.Session().Load("userIdEmitter")
	fmt.Println(userId)
	return userId.(int64)
}

func (c *Socket) GetMetadata() event_emitter.Metadata {
	return c.Session()
}

func Sub(em *event_emitter.EventEmitter[*Socket], topic string, socket *Socket) {
	em.Subscribe(socket, topic, func(subscriber *Socket, msg any) {
		_ = msg.(*gws.Broadcaster).Broadcast(subscriber.Conn)
	})
}

func Pub(em *event_emitter.EventEmitter[*Socket], topic string, op gws.Opcode, msg []byte) {
	broadcaster := gws.NewBroadcaster(op, msg)
	defer broadcaster.Close()
	em.Publish(topic, broadcaster)
}
