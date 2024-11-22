package connect

import (
	"GoQuickIM/api/rpc"
	"GoQuickIM/config"
	"GoQuickIM/pkg/stickpackage"
	"GoQuickIM/proto"
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const maxInt = 1<<31 - 1

func (c *Connect) InitTcpServer() error {
	aTcpAddr := strings.Split(config.Conf.Connect.ConnectTcp.Bind, ",")
	cpuNum := config.Conf.Connect.ConnectBucket.CpuNum
	var (
		addr     *net.TCPAddr
		listener *net.TCPListener
		err      error
	)
	for _, inPort := range aTcpAddr {
		//address convert
		if addr, err = net.ResolveTCPAddr("tcp", inPort); err != nil {
			logrus.Errorf("server_tcp ResolveTCPAddr error:%s", err.Error())
			return err
		}
		if listener, err = net.ListenTCP("tcp", addr); err != nil {
			logrus.Errorf("net.ListenTCP(tcp, %s),error(%v)", inPort, err)
			return err
		}
		logrus.Infof("start tcp listen at:%s", inPort)

		//co work
		for i := 0; i < cpuNum; i++ {
			go c.acceptTcp(listener)
		}
	}
	return nil
}
func (c *Connect) acceptTcp(listener *net.TCPListener) {
	var (
		conn *net.TCPConn
		err  error
		r    int
	)
	connectTcpConfig := config.Conf.Connect.ConnectTcp
	for {
		//get tcp connect
		if conn, err = listener.AcceptTCP(); err != nil {
			return
		}
		//set KeepAlive:False -> always Alive ?
		if err = conn.SetKeepAlive(connectTcpConfig.KeepAlive); err != nil {
			logrus.Errorf("listener.Accept(\"%s\") error(%v)", listener.Addr().String(), err)
			return
		}
		//set receivebuff
		if err = conn.SetReadBuffer(connectTcpConfig.ReceiveBuf); err != nil {
			logrus.Errorf("conn.SetKeepAlive() error:%s", err.Error())
			return
		}
		//set sendbuf
		if err = conn.SetWriteBuffer(connectTcpConfig.SendBuf); err != nil {
			logrus.Errorf("conn.SetReadBuffer() error:%s", err.Error())
			return
		}
		//serve
		go c.ServeTcp(DefaultServer, conn, r)

		//control
		if r++; r == maxInt {
			logrus.Infof("conn.acceptTcp num is %d", r)
			r = 0
		}
	}
}

// read and write for Tcp
func (c *Connect) ServeTcp(server *Server, conn *net.TCPConn, r int) {
	var ch = NewChannel(server.Options.BroadcastSize)
	ch.connTcp = conn
	go c.writeDataToTcp(ch)
	go c.readDataFromTcp(server, ch)
}

func (c *Connect) readDataFromTcp(s *Server, ch *Channel) {
	//Disconnect and deleteChannel
	defer func() {
		logrus.Infof("start exec disConnect ...")
		if ch.Room == nil || ch.userId == 0 {
			logrus.Infof("roomId and userId eq 0")
			_ = ch.connTcp.Close()
			return
		}
		logrus.Infof("exec disConnect")
		disConnectRequest := new(proto.DisConnectRequest)
		disConnectRequest.RoomId = ch.Room.Id
		disConnectRequest.UserId = ch.userId
		s.Buckets[ch.userId].DeleteChannel(ch)
		if err := s.Operator.DisConnect(disConnectRequest); err != nil {
			logrus.Warnf("DisConnect rpc err :%s", err.Error())
		}
		if err := ch.connTcp.Close(); err != nil {
			logrus.Warnf("DisConnect close tcp conn err :%s", err.Error())
		}
	}()
	//scanner parase data packge
	scannerPackage := bufio.NewScanner(ch.connTcp)
	scannerPackage.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if !atEOF && data[0] == 'v' {
			if len(data) > stickpackage.TcpHeaderLength {
				packSumLength := int16(0)
				_ = binary.Read(bytes.NewReader(data[stickpackage.LengthStartIndex:stickpackage.LengthStopIndex]), binary.BigEndian, &packSumLength)
				if int(packSumLength) <= len(data) {
					return int(packSumLength), data[:packSumLength], nil
				}
			}
		}
		return
	})

	//?
	scanTimes := 0
	for {
		scanTimes++
		if scanTimes > 3 {
			logrus.Infof("scannedPack times is:%d", scanTimes)
			break
		}
		for scannerPackage.Scan() {
			scannedPack := new(stickpackage.StickPackage)
			err := scannedPack.Unpack(bytes.NewReader(scannerPackage.Bytes()))
			if err != nil {
				logrus.Errorf("scan tcp package err:%s", err.Error())
				break
			}
			//get one full package
			var connReq proto.ConncetRequest
			logrus.Infof("get a messge:%s", scannedPack)
			var rawTcpMsg proto.SendTcp
			if err := json.Unmarshal([]byte(scannedPack.Msg), &rawTcpMsg); err != nil {
				logrus.Errorf("tcp message struct %+v", rawTcpMsg)
				break
			}
			logrus.Infof("json unmarshal,raw tcp msg is:%+v", rawTcpMsg)
			if rawTcpMsg.AuthToken == "" {
				logrus.Errorf("tcp s.operator.Connect no authToken")
				return
			}
			if rawTcpMsg.RoomId <= 0 {
				logrus.Errorf("tcp roomId not allow lgt 0")
				return
			}
			switch rawTcpMsg.Op {
			case config.OpBuildTcpConn:
				connReq.AuthToken = rawTcpMsg.AuthToken
				connReq.RoomId = rawTcpMsg.RoomId
				connReq.ServerId = c.ServerId
				userId, err := s.Operator.Connect(&connReq)
				logrus.Infof("tcp s.operator.Connect userId is :%d", userId)
				if err != nil {
					logrus.Errorf("tcp s.operator.Connect error %s", err.Error())
					return
				}
				if userId == 0 {
					logrus.Error("tcp Invalid AuthToken ,userId empty")
					return
				}
				b := s.Bucket(userId)
				err = b.Put(userId, connReq.RoomId, ch)
				if err != nil {
					logrus.Errorf("tcp conn put room err: %s", err.Error())
					_ = ch.connTcp.Close()
					return
				}
			case config.OpRoomSend:
				req := &proto.Send{
					Msg:          rawTcpMsg.Msg,
					FromUserId:   rawTcpMsg.FromUserId,
					FromUserName: rawTcpMsg.FromUserName,
					RoomId:       rawTcpMsg.RoomId,
					Op:           config.OpRoomSend,
				}
				code, msg := rpc.RpcLogicObj.PushRoom(req)
				logrus.Infof("tcp conn push msg to room,err code is:%d,err msg is:%s", code, msg)
			}
		}
		if err := scannerPackage.Err(); err != nil {
			logrus.Errorf("tcp get a err package:%s", err.Error())
			return
		}
	}

}
func (c *Connect) writeDataToTcp(ch *Channel) {
	//ping time 54s
	ticker := time.NewTicker(DefaultServer.Options.PingPeriod)
	defer func() {
		ticker.Stop()
		_ = ch.connTcp.Close()
	}()
	//package for tcp
	pack := stickpackage.StickPackage{
		Version: stickpackage.VersionContent,
	}
	for {
		select {
		case message, ok := <-ch.broadcast:
			if !ok {
				_ = ch.connTcp.Close()
				return
			}
			pack.Msg = message.Body
			pack.Length = pack.GetPackageLength()
			//send
			logrus.Infof("send tcp msg to conn:%s", pack.String())

			if err := pack.Pack(ch.connTcp); err != nil {
				logrus.Errorf("connTcp.write message err:%s", err.Error())
				return
			}
		case <-ticker.C:
			logrus.Infof("connTcp.ping message,send")
			//send a ping msg ,if error , return
			pack.Msg = []byte("ping msg")
			pack.Length = pack.GetPackageLength()
			if err := pack.Pack(ch.connTcp); err != nil {
				//send ping msg to tcp conn
				return
			}
		}

	}
}
