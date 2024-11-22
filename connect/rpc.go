package connect

import (
	"GoQuickIM/config"
	"GoQuickIM/proto"
	"GoQuickIM/tools"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/rpcxio/libkv/store"
	etcdV3 "github.com/rpcxio/rpcx-etcd/client"
	"github.com/rpcxio/rpcx-etcd/serverplugin"
	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/server"
)

var logicRpcClient client.XClient
var once sync.Once

type RpcConnect struct {
}

type RpcConnectPush struct {
}

func (rpc *RpcConnectPush) PushSingleMsg(ctx context.Context, pushMsgReq *proto.PushMsgRequest, reply *proto.SuccessReply) (err error) {
	var (
		bucket  *Bucket
		channel *Channel
	)
	logrus.Infof("rpc PushMsg:%v", pushMsgReq)
	if pushMsgReq == nil {
		logrus.Errorf("rpc PushSingleMsg() args:(%v)", pushMsgReq)
		return
	}
	bucket = DefaultServer.Bucket(pushMsgReq.UserId)
	if channel = bucket.Channel(pushMsgReq.UserId); channel != nil {
		logrus.Infof("DefaultServer Channel err nil ,args: %v", pushMsgReq)
		err = channel.Push(&pushMsgReq.Msg)
	}
	reply.Code = config.SuccessReplyCode
	reply.Msg = config.SuccessReplyMsg
	logrus.Infof("successReply:%v", reply)
	return
}
func (rpc *RpcConnectPush) PushRoomMsg(ctx context.Context, pushRoomMsgReq *proto.PushRoomMsgRequest, successReply *proto.SuccessReply) (err error) {
	successReply.Code = config.SuccessReplyCode
	successReply.Msg = config.SuccessReplyMsg
	logrus.Infof("PushRoomMsg msg %+v", pushRoomMsgReq)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(pushRoomMsgReq)
	}
	return
}

func (rpc *RpcConnectPush) PushRoomCount(ctx context.Context, pushRoomMsgReq *proto.PushRoomMsgRequest, successReply *proto.SuccessReply) (err error) {
	successReply.Code = config.SuccessReplyCode
	successReply.Msg = config.SuccessReplyMsg
	logrus.Infof("PushRoomCount msg %v", pushRoomMsgReq)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(pushRoomMsgReq)
	}
	return
}

func (rpc *RpcConnectPush) PushRoomInfo(ctx context.Context, pushRoomMsgReq *proto.PushRoomMsgRequest, successReply *proto.SuccessReply) (err error) {
	successReply.Code = config.SuccessReplyCode
	successReply.Msg = config.SuccessReplyMsg
	logrus.Infof("connect,PushRoomInfo msg %+v", pushRoomMsgReq)
	for _, bucket := range DefaultServer.Buckets {
		bucket.BroadcastRoom(pushRoomMsgReq)
	}
	return
}
func (c *Connect) InitConnectWebsocketRpcServer() (err error) {
	var network, addr string
	connctRpcAddress := strings.Split(config.Conf.Connect.ConnectRpcAddressWebSockts.Address, ",")
	for _, bind := range connctRpcAddress {
		if network, addr, err = tools.ParseNetWork(bind); err != nil {
			logrus.Panicf("InitConnectWebSocketRpcServer ParseNetwork error:%s", err)
		}
		logrus.Infof("Connect start run at -->%s:%s", network, addr)
		go c.createConnectWebsocketsRpcServer(network, addr)
	}
	return
}
func (c *Connect) createConnectWebsocketsRpcServer(network, addr string) {
	s := server.NewServer()
	addRegistryPlugin(s, network, addr)
	s.RegisterName(config.Conf.Common.CommonEtcd.ServerPathConnect, new(RpcConnectPush), fmt.Sprintf("serverId=%s&serverType=ws", c.ServerId))
	s.RegisterOnShutdown(func(s *server.Server) {
		s.UnregisterAll()
	})
	s.Serve(network, addr)
}
func addRegistryPlugin(s *server.Server, network, addr string) {
	r := &serverplugin.EtcdV3RegisterPlugin{
		ServiceAddress: network + "@" + addr,
		EtcdServers:    []string{config.Conf.Common.CommonEtcd.Host},
		BasePath:       config.Conf.Common.CommonEtcd.BasePath,
		Metrics:        metrics.NewRegistry(),
		UpdateInterval: time.Minute,
	}
	err := r.Start()
	if err != nil {
		logrus.Fatal(err)
	}
	s.Plugins.Add(r)
}
func (rpc *RpcConnect) Connect(connReq *proto.ConncetRequest) (uid int, err error) {
	reply := &proto.ConncetResponse{}
	err = logicRpcClient.Call(context.Background(), "Connect", connReq, reply)
	if err != nil {
		logrus.Fatalf("failed to call: %v", err)
	}
	uid = reply.UserId
	logrus.Infof("connect logic userId:%d", reply.UserId)
	return
}
func (rpc *RpcConnect) DisConnect(disConnReq *proto.DisConnectRequest) (err error) {
	reply := &proto.DisConnectResponse{}
	err = logicRpcClient.Call(context.Background(), "DisConnect", disConnReq, reply)
	if err != nil {
		logrus.Fatalf("failed to call: %v", err)
	}
	return
}

func (c *Connect) InitLogicRpcClient() (err error) {
	//config etcd,not using TLS security,using Persist long connection
	etcdConfigOption := &store.Config{
		ClientTLS:         nil,
		TLS:               nil,
		ConnectionTimeout: time.Duration(config.Conf.Common.CommonEtcd.ConnectionTimeOut) * time.Second,
		Bucket:            "",
		PersistConnection: true,
		Username:          config.Conf.Common.CommonEtcd.UserName,
		Password:          config.Conf.Common.CommonEtcd.Password,
	}
	once.Do(func() {
		//register
		d, e := etcdV3.NewEtcdV3Discovery(
			config.Conf.Common.CommonEtcd.BasePath,
			config.Conf.Common.CommonEtcd.ServerPathLogic,
			[]string{config.Conf.Common.CommonEtcd.Host},
			true,
			etcdConfigOption,
		)
		if e != nil {
			logrus.Fatalf("init conncet rpc etcd discovery client fail:%s", e.Error())

		}
		logicRpcClient = client.NewXClient(config.Conf.Common.CommonEtcd.ServerPathLogic, client.Failtry, client.RandomSelect, d, client.DefaultOption)
	})
	return
}

func (c *Connect) InitConnectTcpRpcServer() (err error) {
	var network, addr string
	connectRpcAddress := strings.Split(config.Conf.Connect.ConnectRpcAddressTcp.Address, ",")
	for _, bind := range connectRpcAddress {
		if network, addr, err = tools.ParseNetWork(bind); err != nil {
			logrus.Panicf("InitConnectTcpRpcServer ParseNetwork error : %s", err)
		}
		logrus.Infof("Connect start run at-->%s:%s", network, addr)
		go c.createConnectTcpRpcServer(network, addr)
	}
	return
}

func (c *Connect) createConnectTcpRpcServer(network string, addr string) {
	s := server.NewServer()
	addRegistryPlugin(s, network, addr)
	s.RegisterName(config.Conf.Common.CommonEtcd.ServerPathConnect, new(RpcConnectPush), fmt.Sprintf("serverId=%s&serverType=tcp", c.ServerId))
	s.RegisterOnShutdown(func(s *server.Server) {
		s.UnregisterAll()
	})
	s.Serve(network, addr)
}
