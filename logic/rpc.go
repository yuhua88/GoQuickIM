package logic

import (
	"GoQuickIM/config"
	"GoQuickIM/logic/dao"
	"GoQuickIM/proto"
	"GoQuickIM/tools"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type RpcLogic struct {
}

func (rpc *RpcLogic) Register(ctx context.Context, args *proto.RegisterRequest, reply *proto.RegisterResponse) (err error) {
	reply.Code = config.FailReplyCode
	u := new(dao.User)
	uData := u.CheckHaveUserName(args.Name)
	if uData.Id > 0 {
		return errors.New("this user name already have, please login")
	}
	u.UserName = args.Name
	u.Password = args.PassWord

	//add user in db
	userId, err := u.Add()

	if err != nil {
		logrus.Infof("register err:%s", err.Error())
		return err
	}
	if userId == 0 {
		return errors.New("register userId empty")
	}
	//set token
	randToken := tools.GetRandomToken(32)
	sessionId := tools.CreateSessionId(randToken)
	userData := make(map[string]interface{})
	userData["userId"] = userId
	userData["userName"] = u.UserName
	//store in Redis
	RedisSessClient.Do(context.Background(), "MULTI")
	RedisSessClient.HMSet(context.Background(), sessionId, userData)
	RedisSessClient.Expire(context.Background(), sessionId, config.RedisBaseValidTime)
	err = RedisSessClient.Do(context.Background(), "EXEC").Err()
	if err != nil {
		logrus.Infof("register set redis token fail")
		return err
	}
	reply.Code = config.SuccessReplyCode
	reply.AuthToken = randToken
	return
}

func (rpc *RpcLogic) Login(ctx context.Context, args *proto.LoginRequest, reply *proto.RegisterResponse) (err error) {
	reply.Code = config.FailReplyCode
	u := new(dao.User)
	userName := args.Name
	passWord := args.PassWord
	data := u.CheckHaveUserName(userName)
	if data.Id == 0 || passWord != data.Password {
		return errors.New("no this user or password error")
	}
	loginSessionId := tools.GetSessionIdByUserId(data.Id)
	//set token userData
	randToken := tools.GetRandomToken(32)
	sessionId := tools.CreateSessionId(randToken)
	userData := make(map[string]interface{})
	userData["userId"] = data.Id
	userData["userName"] = data.UserName
	//clean old Session before login
	token, _ := RedisSessClient.Get(context.Background(), loginSessionId).Result()
	if token != "" {
		oldSession := tools.CreateSessionId(token)
		err := RedisSessClient.Del(context.Background(), oldSession).Err()
		if err != nil {
			return errors.New("logout user fail! token is:" + token)
		}
	}
	RedisSessClient.Do(context.Background(), "MULTI")
	RedisSessClient.HMSet(context.Background(), sessionId, userData)
	RedisSessClient.Expire(context.Background(), sessionId, config.RedisBaseValidTime)
	RedisSessClient.Set(context.Background(), loginSessionId, randToken, config.RedisBaseValidTime)
	err = RedisSessClient.Do(context.Background(), "EXEC").Err()
	if err != nil {
		logrus.Infof("register set redis token fail")
	}
	reply.Code = config.SuccessReplyCode
	reply.AuthToken = randToken
	return
}

func (rpc *RpcLogic) GetUserInfoByUserId(ctx context.Context, args *proto.GetUserInfoRequest, reply *proto.GetUserInfoResponse) (err error) {
	reply.Code = config.FailReplyCode
	userId := args.UserId
	u := new(dao.User)
	userName := u.GetUserNameByUserId(userId)
	reply.UserId = userId
	reply.UserName = userName
	reply.Code = config.SuccessReplyCode
	return
}

func (rpc *RpcLogic) CheckAuth(ctx context.Context, args *proto.CheckAuthRequest, reply *proto.CheckAuthResponse) (err error) {
	reply.Code = config.FailReplyCode
	authToken := args.AuthToken
	sessionName := tools.GetSessionName(authToken)
	var userDataMap = map[string]string{}
	userDataMap, err = RedisSessClient.HGetAll(context.Background(), sessionName).Result()
	if err != nil {
		logrus.Infof("check auth fail,authtoken is: %s", authToken)
		return err
	}
	if len(userDataMap) == 0 {
		logrus.Infof("no this user session,authToken is %s", authToken)
		return
	}
	intUserId, _ := strconv.Atoi(userDataMap["userId"])
	reply.UserId = intUserId
	userName := userDataMap["userName"]
	reply.Code = config.SuccessReplyCode
	reply.UserName = userName
	return
}

func (rpc *RpcLogic) Logout(ctx context.Context, args *proto.LogoutRequest, reply *proto.LogoutReponse) (err error) {
	reply.Code = config.FailReplyCode
	authToken := args.AuthToken
	sessionName := tools.GetSessionName(authToken)

	var userDataMap = map[string]string{}
	userDataMap, err = RedisSessClient.HGetAll(context.Background(), sessionName).Result()
	if err != nil {
		logrus.Infof("check auth fail,authToken is:%s", authToken)
		return err
	}
	if len(userDataMap) == 0 {
		logrus.Infof("no this user session,authToken is:%s", authToken)
		return
	}
	intUserId, _ := strconv.Atoi(userDataMap["userId"])
	sessionId := tools.GetSessionIdByUserId(intUserId)
	//del session
	err = RedisSessClient.Del(context.Background(), sessionId).Err()
	if err != nil {
		logrus.Infof("logout del sess map error:%s", err.Error())
		return err
	}
	//del server
	logic := new(Logic)
	serverIdKey := logic.getUserKey(fmt.Sprintf("%d", intUserId))
	err = RedisSessClient.Del(context.Background(), serverIdKey).Err()
	if err != nil {
		logrus.Infof("logout del server id error:%s", err.Error())
		return err
	}
	reply.Code = config.SuccessReplyCode
	return
}

func (rpc *RpcLogic) Push(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	sendData := args
	var bodyBytes []byte
	bodyBytes, err = json.Marshal(sendData)
	if err != nil {
		logrus.Errorf("logic,push msg fail,err:%s", err.Error())
		return
	}
	logic := new(Logic)
	//get address
	userSidKey := logic.getUserKey(fmt.Sprintf("%d", sendData.ToUserId))
	serverIdStr := RedisSessClient.Get(context.Background(), userSidKey).Val()
	//send msg
	err = logic.RedisPublishChannel(serverIdStr, sendData.ToUserId, bodyBytes)
	if err != nil {
		logrus.Errorf("logic,redis publish error:%s", err.Error())
		return
	}
	reply.Code = config.SuccessReplyCode
	return

}

func (rpc *RpcLogic) PushRoom(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	sendData := args
	roomId := sendData.RoomId
	logic := new(Logic)
	roomUserInfo := make(map[string]string)
	roomUserKey := logic.getRoomUserKey(strconv.Itoa(roomId))
	roomUserInfo, err = RedisClient.HGetAll(context.Background(), roomUserKey).Result()
	if err != nil {
		logrus.Errorf("logic,PushRoom redis hGetAll err:%s", err.Error())
		return
	}
	sendData.CreatTime = tools.GetNowDateTime()
	var bodyBytes []byte
	bodyBytes, err = json.Marshal(sendData)
	if err != nil {
		logrus.Errorf("logic,PushRoom Marshal err:%s", err.Error())
		return
	}
	err = logic.RedisPublishRoomInfo(roomId, len(roomUserInfo), roomUserInfo, bodyBytes)
	if err != nil {
		logrus.Errorf("logic,PushRoom err:%s", err.Error())
		return
	}
	reply.Code = config.SuccessReplyCode
	return
}

// get room user count
func (rpc *RpcLogic) Count(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	roomId := args.RoomId
	logic := new(Logic)
	//get count
	var count int
	roomCountKey := logic.getRoomOnlineCountKey(fmt.Sprintf("%d", roomId))
	count, _ = RedisSessClient.Get(context.Background(), roomCountKey).Int()
	//push count msg
	err = logic.RedisPushRoomCount(roomId, count)
	if err != nil {
		logrus.Errorf("logic,Count err:%s", err.Error())
		return
	}

	reply.Code = config.SuccessReplyCode
	return
}

func (rpc *RpcLogic) GetRoomInfo(ctx context.Context, args *proto.Send, reply *proto.SuccessReply) (err error) {
	reply.Code = config.FailReplyCode
	logic := new(Logic)
	roomId := args.RoomId
	roomUserInfo := make(map[string]string)
	roomUserKey := logic.getRoomUserKey(strconv.Itoa(roomId))
	roomUserInfo, _ = RedisClient.HGetAll(context.Background(), roomUserKey).Result()
	if len(roomUserInfo) == 0 {
		return errors.New("getRoomInfo no this user")
	}
	err = logic.RedisPushRoomInfo(roomId, len(roomUserInfo), roomUserInfo)
	if err != nil {
		logrus.Errorf("logic,GetRoomInfo err:%s", err.Error())
		return
	}
	reply.Code = config.SuccessReplyCode
	return
}

func (rpc *RpcLogic) Connect(ctx context.Context, args *proto.ConncetRequest, reply *proto.ConncetResponse) (err error) {
	if args == nil {
		logrus.Errorf("logic,connect args empty")
		return
	}
	logic := new(Logic)
	logrus.Infof("logic,authToken is %s", args.AuthToken)
	//getUsesInfo
	key := tools.GetSessionName(args.AuthToken)
	userInfo, err := RedisClient.HGetAll(context.Background(), key).Result()
	if err != nil {
		logrus.Infof("RedisCli HgetAll key:%s,err:%s", key, err.Error())
		return err
	}
	if len(userInfo) == 0 {
		reply.UserId = 0
		return
	}
	userIdStr := userInfo["userId"]
	userId, _ := strconv.Atoi(userIdStr)
	reply.UserId = userId

	roomUserKey := logic.getRoomUserKey(strconv.Itoa(args.RoomId))
	if userId != 0 {
		userKey := logic.getUserKey(userIdStr)
		logrus.Infof("logic redis set userKey:%s,ServerId:%s", userKey, args.ServerId)
		validTime := config.RedisBaseValidTime
		//set connect validTime
		err = RedisClient.Set(context.Background(), userKey, args.ServerId, validTime).Err()
		if err != nil {
			logrus.Warnf("logic set err:%s", err)
		}
		//check user has exist
		if RedisClient.HGet(context.Background(), roomUserKey, userIdStr).Val() == "" {
			RedisClient.HSet(context.Background(), roomUserKey, userIdStr, userInfo["userName"])
			//add room user count++
			RedisClient.Incr(context.Background(), logic.getRoomOnlineCountKey(fmt.Sprintf("%d", args.RoomId)))
		}
	}
	logrus.Infof("logic rpc userId:%d", reply.UserId)
	return
}

func (rpc *RpcLogic) DisConnect(ctx context.Context, args *proto.DisConnectRequest, reply *proto.DisConnectResponse) (err error) {
	logic := new(Logic)
	roomIdStr := fmt.Sprintf("%d", args.RoomId)
	roomUserKey := logic.getRoomUserKey(roomIdStr)
	//room user count--
	if args.RoomId > 0 {
		count, _ := RedisSessClient.Get(context.Background(), logic.getRoomOnlineCountKey(roomIdStr)).Int()
		if count > 0 {
			RedisClient.Decr(context.Background(), logic.getRoomOnlineCountKey(roomIdStr)).Result()
		}
	}
	//room login user--
	if args.UserId != 0 {
		err = RedisClient.HDel(context.Background(), fmt.Sprintf("%d", args.UserId)).Err()
		if err != nil {
			logrus.Warnf("HDel getRoomUserKey err:%s", err)
		}
	}
	//below code can optimize send a singal to queue,another process get a signal from queue,then push event to websocker
	roomUserInfo, err := RedisClient.HGetAll(context.Background(), roomUserKey).Result()
	if err != nil {
		logrus.Warnf("RedisClient HGetALl roomUserInfo key:%s,err:%s", roomUserKey, err)
	}
	if err = logic.RedisPublishRoomInfo(args.RoomId, len(roomUserInfo), roomUserInfo, nil); err != nil {
		logrus.Warnf("publish RedisPublishRoomCount err:%s", err.Error())
	}
	return
}
