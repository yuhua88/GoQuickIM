package connect

import (
	"GoQuickIM/proto"
	"sync"
	"sync/atomic"
)

// Bucket -> Room -> Channel
type Bucket struct {
	cLock         sync.RWMutex     //protext the channels for chs
	chs           map[int]*Channel //map sub key to a channel
	bucketOptions BucketOptions
	rooms         map[int]*Room //bucket room channels
	routines      []chan *proto.PushRoomMsgRequest
	routinesNum   uint64
	broadcast     chan []byte
}

type BucketOptions struct {
	ChannelSize    int
	RoomSize       int
	RountineAmount uint64
	RountineSize   int
}

func NewBucket(buckOptions BucketOptions) (b *Bucket) {
	b = new(Bucket)
	b.chs = make(map[int]*Channel, buckOptions.ChannelSize)
	b.bucketOptions = buckOptions
	b.routines = make([]chan *proto.PushRoomMsgRequest, buckOptions.RountineAmount)
	b.rooms = make(map[int]*Room, buckOptions.RoomSize)
	for i := uint64(0); i < buckOptions.RountineAmount; i++ {
		c := make(chan *proto.PushRoomMsgRequest, buckOptions.RountineSize)
		b.routines[i] = c
		go b.PushRoom(c)
	}
	return
}

func (b *Bucket) PushRoom(ch chan *proto.PushRoomMsgRequest) {
	for {
		var (
			arg  *proto.PushRoomMsgRequest
			room *Room
		)
		arg = <-ch
		if room = b.Room(arg.RoomId); room != nil {
			room.Push(&arg.Msg)
		}
	}
}

func (b *Bucket) Put(userId int, roomId int, ch *Channel) (err error) {
	var (
		room *Room
		ok   bool
	)
	b.cLock.Lock()
	defer b.cLock.Unlock()
	if roomId != NoRoom {
		if room, ok = b.rooms[roomId]; !ok {
			room = NewRoom(roomId)
			b.rooms[roomId] = room
		}
		ch.Room = room
	}
	ch.userId = userId
	b.chs[userId] = ch
	if room != nil {
		err = room.Put(ch)
	}
	return
}
func (b *Bucket) Room(rid int) (room *Room) {
	b.cLock.RLock()
	defer b.cLock.RUnlock()

	room = b.rooms[rid]
	return
}

func (b *Bucket) DeleteChannel(ch *Channel) {
	var room *Room
	b.cLock.RLock()
	defer b.cLock.RUnlock()
	if ch, ok := b.chs[ch.userId]; ok {
		room = ch.Room
		//delete from bucket
		delete(b.chs, room.Id)
	}
	//delete from room
	if room != nil && room.DeleteChannel(ch) {
		//if room empty,will mark room.drop true
		if room.drop {
			//delete empty room
			delete(b.rooms, room.Id)
		}
	}
}

func (b *Bucket) Channel(userId int) (ch *Channel) {
	b.cLock.Lock()
	defer b.cLock.Unlock()
	ch = b.chs[userId]
	return
}

func (b *Bucket) BroadcastRoom(pushRoomMsgReq *proto.PushRoomMsgRequest) {
	num := atomic.AddUint64(&b.routinesNum, 1) % b.bucketOptions.RountineAmount
	b.routines[num] <- pushRoomMsgReq
}
