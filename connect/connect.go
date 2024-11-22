package connect

import (
	"GoQuickIM/config"
	"fmt"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var DefaultServer *Server

type Connect struct {
	ServerId string
}

func New() *Connect {
	return new(Connect)
}
func (c *Connect) RunTcp() {
	//get config
	connectConfig := config.Conf.Connect

	//set tha maxinum number of Cpus
	runtime.GOMAXPROCS(connectConfig.ConnectBucket.CpuNum)

	//init logic layer rpc client
	if err := c.InitLogicRpcClient(); err != nil {
		logrus.Panicf("InitLogicRpcClient err:%s", err.Error())
	}
	//init connect layer
	Buckets := make([]*Bucket, connectConfig.ConnectBucket.CpuNum)
	for i := 0; i < connectConfig.ConnectBucket.CpuNum; i++ {
		Buckets[i] = NewBucket(BucketOptions{
			ChannelSize:    connectConfig.ConnectBucket.Channel,
			RoomSize:       connectConfig.ConnectBucket.Room,
			RountineAmount: connectConfig.ConnectBucket.RoutineAmount,
			RountineSize:   connectConfig.ConnectBucket.RoutineSize,
		})
	}
	operator := new(DefaultOperator)
	DefaultServer = NewServer(Buckets, operator, ServerOptions{
		WriteWait:       10 * time.Second,
		PongWait:        60 * time.Second,
		PingPeriod:      54 * time.Second,
		MaxMessageSize:  512,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		BroadcastSize:   512,
	})
	c.ServerId = fmt.Sprintf("%s-%s", "tcp", uuid.New().String())

	//init tcp for connect layer rpc server
	if err := c.InitConnectTcpRpcServer(); err != nil {
		logrus.Panicf("InitConnectWebsocketRpcServer Fatal error: %s \n", err.Error())
	}
	if err := c.InitTcpServer(); err != nil {
		logrus.Panicf("Connect layerInitTcpServer() error:%s\n ", err.Error())
	}

}

func (c *Connect) Run() {
	//get Connect layer config
	connectConfig := config.Conf.Connect

	//set the maximum number of CPUs that cen be executing
	runtime.GOMAXPROCS(connectConfig.ConnectBucket.CpuNum)

	//init logic layer rpc client, call logic layer rpc server
	if err := c.InitLogicRpcClient(); err != nil {
		logrus.Panicf("InitLogicRpcClient err:%s", err.Error())
	}
	//init Conncet layer rpc server,logic client wiil call this
	Buckets := make([]*Bucket, connectConfig.ConnectBucket.CpuNum)
	for i := 0; i < connectConfig.ConnectBucket.CpuNum; i++ {
		Buckets[i] = NewBucket(BucketOptions{
			ChannelSize:    connectConfig.ConnectBucket.Channel,
			RoomSize:       connectConfig.ConnectBucket.Room,
			RountineAmount: connectConfig.ConnectBucket.RoutineAmount,
			RountineSize:   connectConfig.ConnectBucket.RoutineSize,
		})
	}
	operator := new(DefaultOperator)
	DefaultServer = NewServer(Buckets, operator, ServerOptions{
		WriteWait:       10 * time.Second,
		PongWait:        60 * time.Second,
		PingPeriod:      54 * time.Second,
		MaxMessageSize:  512,
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		BroadcastSize:   512,
	})
	c.ServerId = fmt.Sprintf("%s-%s", "ws", uuid.New().String())
	//init Connect layer rpc server,task layer wiill call this
	if err := c.InitConnectWebsocketRpcServer(); err != nil {
		logrus.Panicf("InitConnectWebsocketRpcServer Fatal error: %s \n", err.Error())
	}

	//start Connect layer server handler persistent connection
	if err := c.InitWebsocket(); err != nil {
		logrus.Panicf("Connect layer InitWebsocket() error:  %s \n", err.Error())
	}
}
