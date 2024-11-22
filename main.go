package main

import (
	"GoQuickIM/api"
	"GoQuickIM/connect"
	"GoQuickIM/logic"
	"GoQuickIM/site"
	"GoQuickIM/task"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	//Parase Flag
	var module string
	flag.StringVar(&module, "module", "", "assign run module")
	flag.Parse()
	fmt.Printf("start run %s module\n", module)
	switch module {
	case "task":
		task.New().Run()
	case "connect_tcp":
		connect.New().RunTcp()
	case "connect_websocket":
		connect.New().Run()
	case "logic":
		logic.New().Run()
	case "api":
		api.New().Run()
	case "site":
		site.New().Run()
	default:
		fmt.Println("Exiting,module param error!")
		return
	}
	fmt.Printf("run %s module done\n", module)
	//Exit gracefully
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-quit
	fmt.Println("Server exiting")
}
