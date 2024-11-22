package connect

import (
	"GoQuickIM/proto"
	"errors"
	"sync"

	"github.com/sirupsen/logrus"
)

const NoRoom = -1

// Bucket -> Room -> Channel
type Room struct {
	Id          int
	OnlineCount int //room online user count
	rLock       sync.RWMutex
	drop        bool     //make room alive
	next        *Channel // ?
}

func NewRoom(roomId int) *Room {
	room := new(Room)
	room.Id = roomId
	room.drop = false
	room.next = nil
	room.OnlineCount = 0
	return room
}
func (r *Room) Put(ch *Channel) (err error) {
	//doubly linked list
	r.rLock.Lock()
	defer r.rLock.Unlock()
	if !r.drop {
		if r.next != nil {
			r.next.Prev = ch
		}
		ch.Next = r.next
		r.next = ch
		ch.Prev = nil
		r.OnlineCount++
	} else {
		err = errors.New("room drop")
	}
	return
}

func (r *Room) Push(msg *proto.Msg) {
	r.rLock.RLock()
	defer r.rLock.RUnlock()
	for ch := r.next; ch != nil; ch = ch.Next {
		if err := ch.Push(msg); err != nil {
			logrus.Infof("push msg err:%s", err.Error())
		}
	}
}
func (r *Room) DeleteChannel(ch *Channel) bool {
	r.rLock.RLock()
	defer r.rLock.RUnlock()
	if ch.Next != nil {
		ch.Next.Prev = ch.Prev
	}
	if ch.Prev != nil {
		ch.Prev.Next = ch.Next
	} else {
		r.next = ch.Next
	}
	r.OnlineCount--
	r.drop = false
	if r.OnlineCount <= 0 {
		r.drop = true
	}
	return r.drop
}
