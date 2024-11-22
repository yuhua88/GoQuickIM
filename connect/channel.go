package connect

import (
	"GoQuickIM/proto"
	"github.com/gorilla/websocket"
	"net"
)

// Bucket -> Room -> Channel
type Channel struct {
	Room      *Room
	Next      *Channel
	Prev      *Channel
	broadcast chan *proto.Msg
	userId    int
	conn      *websocket.Conn
	connTcp   *net.TCPConn
}

// non-blocking sends
func (ch *Channel) Push(msg *proto.Msg) (err error) {
	select {
	case ch.broadcast <- msg:
	default:
	}
	return
}

func NewChannel(size int) (c *Channel) {
	c = new(Channel)
	c.broadcast = make(chan *proto.Msg, size)
	c.Next = nil
	c.Prev = nil
	return
}
