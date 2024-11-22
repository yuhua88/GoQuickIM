package api

import (
	"GoQuickIM/api/router"
	"GoQuickIM/api/rpc"
	"GoQuickIM/config"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Chat struct {
}

func New() *Chat {
	return &Chat{}
}

// api server
func (c *Chat) Run() {
	//init rpc client
	rpc.InitLogicRpcClient()
	//regist router
	r := router.Register()
	runMode := config.GetGinRunMode()
	logrus.Info("server start, now run mode is ", runMode)
	gin.SetMode(runMode)
	apiConfig := config.Conf.Api
	port := apiConfig.ApiBase.ListenPort
	flag.Parse()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	//goroutin listen
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("start listen: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	logrus.Infof("Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logrus.Errorf("server shutdown: %s\n", err)
	}
	logrus.Infof("Server exiting")
	os.Exit(0)
}
