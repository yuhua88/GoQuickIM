package logic

import (
	"GoQuickIM/config"
	"fmt"
	"runtime"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type Logic struct {
	ServerId string
}

func New() *Logic {
	return &Logic{}
}
func (logic *Logic) Run() {
	//read logic config: Id, CpuNum, Address
	logicConfig := config.Conf.Logic

	runtime.GOMAXPROCS(logicConfig.LogicBase.CpuNum)
	//generate a unique ServerID by uuid,(2^122),no need to check
	logic.ServerId = fmt.Sprintf("logic-%s", uuid.New().String())
	//init publish redis
	if err := logic.InitPublishRedisClient(); err != nil {
		logrus.Panicf("logic init publishRedisClient fail,err:%s", err.Error())
	}

	//init rpc server
	if err := logic.InitRpcServer(); err != nil {
		logrus.Panicf("logic init rpc server fail")
	}

}
