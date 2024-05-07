package server

import (
	"time"

	"github.com/lxzan/event_emitter"
	"github.com/lxzan/gws"
)

const (
	PingInterval = 5 * time.Second
	PingWait     = 30 * time.Minute
)

type Handler struct {
	Emitter *event_emitter.EventEmitter[*Socket]
}

func NewHandler() *Handler {
	emitter := event_emitter.New[*Socket](&event_emitter.Config{
		BucketNum:  16,
		BucketSize: 128,
	})
	return &Handler{
		Emitter: emitter,
	}
}

func NewWebsocketUpgrader() *gws.Upgrader {
	return gws.NewUpgrader(&Handler{}, &gws.ServerOption{
		ParallelEnabled:   true,
		Recovery:          gws.Recovery,
		PermessageDeflate: gws.PermessageDeflate{Enabled: true},
	})
}

func (c *Handler) OnOpen(socket *gws.Conn) {
	_ = socket.SetDeadline(time.Now().Add(PingInterval + PingWait))
}

func (c *Handler) OnClose(socket *gws.Conn, err error) {}

func (c *Handler) OnPing(socket *gws.Conn, payload []byte) {
	_ = socket.SetDeadline(time.Now().Add(PingInterval + PingWait))
	_ = socket.WriteString("pong")
}

func (c *Handler) OnPong(socket *gws.Conn, payload []byte) {}

func (c *Handler) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()
	if b := message.Data.Bytes(); len(b) == 4 && string(b) == "ping" {
		c.OnPing(socket, nil)
		return
	}

	Pub(globalEmitter, "event", gws.OpcodeText, message.Bytes())
}

type Socket struct{ *gws.Conn }

func (c *Socket) GetSubscriberID() int64 {
	userId, _ := c.Session().Load("userId")
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
