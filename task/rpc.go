package task

import (
	"GoQuickIM/config"
	"GoQuickIM/proto"
	"GoQuickIM/tools"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/rpcxio/libkv/store"
	etcdV3 "github.com/rpcxio/rpcx-etcd/client"
	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/client"
)

var RClient = &RpcConnectClient{
	ServerInsMap: make(map[string][]Instance),
	IndexMap:     make(map[string]int),
}

type Instance struct {
	ServerType string
	ServerId   string
	Client     client.XClient
}
type RpcConnectClient struct {
	lock         sync.Mutex
	ServerInsMap map[string][]Instance // ?
	IndexMap     map[string]int
}

func (task *Task) InitConnectRpcClient() (err error) {
	etcdConfigOption := &store.Config{
		ClientTLS:         nil,
		TLS:               nil,
		ConnectionTimeout: time.Duration(config.Conf.Common.CommonEtcd.ConnectionTimeOut) * time.Second,
		Bucket:            "",
		PersistConnection: true,
		Username:          config.Conf.Common.CommonEtcd.UserName,
		Password:          config.Conf.Common.CommonEtcd.Password,
	}
	etcdConfig := config.Conf.Common.CommonEtcd
	d, e := etcdV3.NewEtcdV3Discovery(
		etcdConfig.BasePath,
		etcdConfig.ServerPathConnect,
		strings.Split(config.Conf.Common.CommonEtcd.Host, ","),
		true,
		etcdConfigOption,
	)
	if e != nil {
		logrus.Fatalf("init task rpc etcd discovery client fail:%s", e.Error())
	}
	if len(d.GetServices()) <= 0 {
		logrus.Panicf("no etcd server find!")
	}
	for _, connectConf := range d.GetServices() {

		//RpcConnectClients
		serverType := getParamByKey(connectConf.Value, "serverType")
		serverId := getParamByKey(connectConf.Value, "serverId")
		logrus.Infof("serverType is:%s,serverId is:%s", serverType, serverId)
		if serverType == "" || serverId == "" {
			continue
		}
		//pear to pear
		d, e := client.NewPeer2PeerDiscovery(connectConf.Key, "")
		if e != nil {

			continue
		}
		c := client.NewXClient(etcdConfig.ServerPathConnect, client.Failtry, client.RandomSelect, d, client.DefaultOption)
		ins := Instance{
			ServerType: serverType,
			ServerId:   serverId,
			Client:     c,
		}
		if _, ok := RClient.ServerInsMap[serverId]; !ok {
			RClient.ServerInsMap[serverId] = []Instance{ins}
		} else {
			RClient.ServerInsMap[serverId] = append(RClient.ServerInsMap[serverId], ins)
		}
	}
	//watch connect server change && update RpcConnectClientList
	go task.watchServicesChange(d)
	return
}
func (task *Task) watchServicesChange(d client.ServiceDiscovery) {
	etcdConfig := config.Conf.Common.CommonEtcd
	//watch
	for kvChan := range d.WatchService() {
		if len(kvChan) <= 0 {
			logrus.Errorf("connect services change, connect alarm, no abailable ip")
		}
		logrus.Infof("connect services change trigger...")
		insMap := make(map[string][]Instance)
		for _, kv := range kvChan {
			serverType := getParamByKey(kv.Value, "serverType")
			serverId := getParamByKey(kv.Value, "serverId")
			logrus.Infof("serverType is:%s,serverId is:%s", serverType, serverId)
			if serverType == "" || serverId == "" {
				continue
			}
			d, e := client.NewPeer2PeerDiscovery(kv.Key, "")
			if e != nil {
				logrus.Errorf("init task client.NewPeer2PeerDiscovery watch client fail:%s", e.Error())
				continue
			}
			c := client.NewXClient(etcdConfig.ServerPathConnect, client.Failtry, client.RandomSelect, d, client.DefaultOption)
			ins := Instance{
				ServerType: serverType,
				ServerId:   serverId,
				Client:     c,
			}
			if _, ok := insMap[serverId]; !ok {
				insMap[serverId] = []Instance{ins}
			} else {
				insMap[serverId] = append(insMap[serverId], ins)
			}
		}
		RClient.lock.Lock()
		RClient.ServerInsMap = insMap
		RClient.lock.Unlock()
	}
}
func getParamByKey(s, key string) string {
	params := strings.Split(s, "&")
	for _, p := range params {
		kv := strings.Split(p, "=")
		if len(kv) == 2 && kv[0] == key {
			return kv[1]
		}
	}
	return ""
}
func (task *Task) broadcastRoomCountToConnect(roomId, count int) {
	// ?
	msg := &proto.RedisRoomCountMsg{
		Count: count,
		Op:    config.OpRoomCountSend,
	}
	var (
		body []byte
		err  error
	)
	if body, err = json.Marshal(msg); err != nil {
		logrus.Warnf("broadcastRoomCountToConnect  json.Marshal err :%s", err.Error())
		return
	}
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpRoomCountSend,
			SeqId:     tools.GetSnowflakeId(),
			Body:      body,
		},
	}
	reply := &proto.SuccessReply{}
	rpcList := RClient.GetAllConnectTypeRpcClient()
	for _, rpc := range rpcList {
		logrus.Infof("broadcastRoomCountToConnect rpc  %v", rpc)
		rpc.Call(context.Background(), "PushRoomCount", pushRoomMsgReq, reply)
		logrus.Infof("reply %s", reply.Msg)
	}

}
func (task *Task) broadcastRoomToConnect(roomId int, msg []byte) {
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpRoomSend,
			SeqId:     tools.GetSnowflakeId(),
			Body:      msg,
		},
	}
	reply := &proto.SuccessReply{}
	rpcList := RClient.GetAllConnectTypeRpcClient()
	for _, rpc := range rpcList {
		logrus.Infof("broadcastRoomToConnect rpc  %v", rpc)
		// ?
		rpc.Call(context.Background(), "PushRoomMsg", pushRoomMsgReq, reply)
		logrus.Infof("reply %s", reply.Msg)
	}
}
func (rc *RpcConnectClient) GetAllConnectTypeRpcClient() (rpcClientList []client.XClient) {
	for serverId := range rc.ServerInsMap {
		c, err := rc.GetRpcClientByServerId(serverId)
		if err != nil {
			logrus.Infof("GetAllConnectTypeRpcClient err:%s", err.Error())
			continue
		}
		rpcClientList = append(rpcClientList, c)
	}
	return
}
func (rc *RpcConnectClient) GetRpcClientByServerId(serverId string) (c client.XClient, err error) {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	if _, ok := rc.ServerInsMap[serverId]; !ok || len(rc.ServerInsMap[serverId]) <= 0 {
		return nil, errors.New("no connect layer ip:" + serverId)
	}
	if _, ok := rc.IndexMap[serverId]; !ok {
		rc.IndexMap = map[string]int{
			serverId: 0,
		}
	}
	idx := rc.IndexMap[serverId] % len(rc.ServerInsMap[serverId])
	ins := rc.ServerInsMap[serverId][idx]
	rc.IndexMap[serverId] = (rc.IndexMap[serverId] + 1) % len(rc.ServerInsMap[serverId])
	return ins.Client, nil
}

func (task *Task) broadcastRoomInfoToConnect(roomId int, roomUserInfo map[string]string) {
	msg := &proto.RedisRoomInfo{
		Count:        len(roomUserInfo),
		Op:           config.OpRoomInfoSend,
		RoomUserInfo: roomUserInfo,
		RoomId:       roomId,
	}
	var body []byte
	var err error
	if body, err = json.Marshal(msg); err != nil {
		logrus.Warnf("broadcastRoomInfoToConnect  json.Marshal err :%s", err.Error())
		return
	}
	pushRoomMsgReq := &proto.PushRoomMsgRequest{
		RoomId: roomId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpRoomInfoSend,
			SeqId:     tools.GetSnowflakeId(),
			Body:      body,
		},
	}
	reply := &proto.SuccessReply{}
	rpcList := RClient.GetAllConnectTypeRpcClient()
	for _, rpc := range rpcList {
		logrus.Infof("broadcastRoomInfoToConnect rpc  %v", rpc)
		rpc.Call(context.Background(), "PushRoomInfo", pushRoomMsgReq, reply)
		logrus.Infof("broadcastRoomInfoToConnect rpc  reply %v", reply)
	}
}

func (task *Task) pushSingleToConnect(serverId string, userId int, msg []byte) {
	logrus.Infof("pushSingleToConnect Body %s", string(msg))
	pushMsgReq := &proto.PushMsgRequest{
		UserId: userId,
		Msg: proto.Msg{
			Ver:       config.MsgVersion,
			Operation: config.OpSingleSend,
			SeqId:     tools.GetSnowflakeId(),
			Body:      msg,
		},
	}
	reply := &proto.SuccessReply{}
	connectRpc, err := RClient.GetRpcClientByServerId(serverId)
	if err != nil {
		logrus.Infof("get rpc client err %v", err)
	}
	err = connectRpc.Call(context.Background(), "PushSingleMsg", pushMsgReq, reply)
	if err != nil {
		logrus.Infof("pushSingleToConnect Call err %v", err)
	}
	logrus.Infof("reply %s", reply.Msg)
}
