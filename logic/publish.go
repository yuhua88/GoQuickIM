package logic

import (
	"GoQuickIM/config"
	"GoQuickIM/proto"
	"GoQuickIM/tools"
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/rcrowley/go-metrics"
	"github.com/rpcxio/rpcx-etcd/serverplugin"
	"github.com/sirupsen/logrus"
	"github.com/smallnest/rpcx/server"
)

var RedisClient *redis.Client
var RedisSessClient *redis.Client

func (logic *Logic) InitPublishRedisClient() (err error) {
	redisOpt := tools.RedisOption{
		Address:  config.Conf.Common.CommonRedis.RedisAddress,
		Password: config.Conf.Common.CommonRedis.RedisPassword,
		Db:       config.Conf.Common.CommonRedis.Db,
	}
	RedisClient = tools.GetRedisInstance(redisOpt)
	//pong, err := RedisClient.Ping().Result(); ?
	if pong, err := RedisClient.Ping(context.Background()).Result(); err != nil {
		logrus.Infof("RedisCall Ping Result pong:%s, err:%s", pong, err)
	}
	RedisSessClient = RedisClient //ref
	return err

}

func (logic *Logic) InitRpcServer() (err error) {
	var network, addr string
	//a host multi port case
	rpcAddressList := strings.Split(config.Conf.Logic.LogicBase.RpcAddress, ",")
	for _, bind := range rpcAddressList {
		if network, addr, err = tools.ParseNetWork(bind); err != nil {
			logrus.Panicf("InitLogicRpc ParseNetWork error: %s", err.Error())
		}
		logrus.Infof("logic start run at-->%s:%s", network, addr)
		//concurrent Server
		go logic.createRpcServer(network, addr)
	}
	return
}

func (logic *Logic) createRpcServer(network, addr string) {
	s := server.NewServer()
	logic.addRegistryPlugin(s, network, addr)
	//serverId must be unique
	//rpcx RegisterName(name,interface,Id)
	err := s.RegisterName(config.Conf.Common.CommonEtcd.ServerPathLogic, new(RpcLogic), logic.ServerId)
	if err != nil {
		logrus.Errorf("register error:%s", err.Error())
	}
	//shutdown and unregister
	s.RegisterOnShutdown(func(s *server.Server) {
		s.UnregisterAll()
	})
	s.Serve(network, addr)
}

// regist to Etcd
func (logic *Logic) addRegistryPlugin(s *server.Server, network, addr string) {
	r := &serverplugin.EtcdV3RegisterPlugin{
		ServiceAddress: network + "@" + addr,
		EtcdServers:    []string{config.Conf.Common.CommonEtcd.Host},
		BasePath:       config.Conf.Common.CommonEtcd.BasePath,
		//Watch Server Health status
		Metrics:        metrics.NewRegistry(),
		UpdateInterval: time.Minute,
	}
	err := r.Start()
	if err != nil {
		logrus.Fatal(err)
	}
	s.Plugins.Add(r)
}

// "gochat_"+"id"
func (logic *Logic) getUserKey(userId string) string {
	var key bytes.Buffer
	key.WriteString(config.RedisPrefix)
	key.WriteString(userId)
	return key.String()
}

// send msg for one user
func (logic *Logic) RedisPublishChannel(serverId string, toUserId int, msg []byte) (err error) {
	redisMsg := proto.RedisMsg{
		Op:       config.OpSingleSend,
		ServerId: serverId,
		UserId:   toUserId,
		Msg:      msg,
	}
	redisMsgStr, err := json.Marshal(redisMsg)
	if err != nil {
		logrus.Errorf("logic,RedisPublishChannel Marshal err:%s", err.Error())
		return err
	}
	redisChannel := config.QueueName //one and only queue
	//Left Push to Channel
	if err := RedisClient.LPush(context.Background(), redisChannel, redisMsgStr).Err(); err != nil {
		logrus.Errorf("logic,lpush err:%s", err.Error())
		return err
	}
	return
}

// send msg for Room
// just repeat Singal
func (logic *Logic) RedisPublishRoomInfo(roomId int, count int, RoomUserInfo map[string]string, msg []byte) (err error) {
	redisMsg := proto.RedisMsg{
		Op:           config.OpRoomSend,
		RoomId:       roomId,
		Count:        count,
		Msg:          msg,
		RoomUserInfo: RoomUserInfo,
	}
	redisMsgStr, err := json.Marshal(redisMsg)
	if err != nil {
		logrus.Errorf("logic,RedisPublishRoomInfo redisMsg error:%s", err.Error())
		return
	}
	redisChannel := config.QueueName
	if err := RedisClient.LPush(context.Background(), redisChannel, redisMsgStr).Err(); err != nil {
		logrus.Errorf("logic,lpush err:%s", err.Error())
		return err
	}
	return
}

// send count msg
func (logic *Logic) RedisPushRoomCount(roomId int, count int) (err error) {
	var redisMsg = &proto.RedisMsg{
		Op:     config.OpRoomCountSend,
		RoomId: roomId,
		Count:  count,
	}
	redisMsgByte, err := json.Marshal(redisMsg)
	if err != nil {
		logrus.Errorf("logic,RedisPushRoomCount redisMsg error : %s", err.Error())
		return
	}
	redisChannel := config.QueueName
	err = RedisClient.LPush(context.Background(), redisChannel, redisMsgByte).Err()
	if err != nil {
		logrus.Errorf("logic,RedisPushRoomCount redisMsg error : %s", err.Error())
		return
	}
	return
}

// send roomInfo msg
func (logic *Logic) RedisPushRoomInfo(roomId int, count int, roomUserInfo map[string]string) (err error) {
	var redisMsg = &proto.RedisMsg{
		Op:           config.OpRoomInfoSend,
		RoomId:       roomId,
		Count:        count,
		RoomUserInfo: roomUserInfo,
	}
	redisMsgByte, err := json.Marshal(redisMsg)
	if err != nil {
		logrus.Errorf("logic,RedisPushRoomInfo redisMsg error : %s", err.Error())
		return
	}
	redisChannel := config.QueueName
	err = RedisClient.LPush(context.Background(), redisChannel, redisMsgByte).Err()
	if err != nil {
		logrus.Errorf("logic,RedisPushRoomInfo redisMsg error : %s", err.Error())
		return
	}
	return
}

// "gochat_room"+"roomId"
func (logic *Logic) getRoomUserKey(authKey string) string {
	var key bytes.Buffer
	key.WriteString(config.RedisRoomPrefix)
	key.WriteString(authKey)
	return key.String()
}

// "gochat_room_online_count_" + "roomId"
func (logic *Logic) getRoomOnlineCountKey(authKey string) string {
	var key bytes.Buffer
	key.WriteString(config.RedisRoomOnlinePrefix)
	key.WriteString(authKey)
	return key.String()
}
