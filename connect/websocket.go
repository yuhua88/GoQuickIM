package connect

import (
	"GoQuickIM/config"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

func (c *Connect) InitWebsocket() error {
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		c.ServeWs(DefaultServer, w, r)
	})
	err := http.ListenAndServe(config.Conf.Connect.ConnectWebsocket.Bind, nil)
	return err
}

func (c *Connect) ServeWs(server *Server, w http.ResponseWriter, r *http.Request) {
	var upGrader = websocket.Upgrader{
		ReadBufferSize:  server.Options.ReadBufferSize,
		WriteBufferSize: server.Options.WriteBufferSize,
	}
	//cross origin domain support
	upGrader.CheckOrigin = func(r *http.Request) bool { return true }

	conn, err := upGrader.Upgrade(w, r, nil)

	if err != nil {
		logrus.Errorf("serverWs err:%s", err.Error())
		return
	}
	var ch *Channel = NewChannel(server.Options.BroadcastSize)
	//default broadcast size eq 512

	ch.conn = conn
	//send data to websocket conn
	go server.writePump(ch, c)
	//get data from websocket conn
	go server.readPump(ch, c)

}
