package site

import (
	"GoQuickIM/config"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/sirupsen/logrus"
)

type Site struct {
}

func New() *Site {
	return &Site{}
}

func notFound(w http.ResponseWriter) {
	data, _ := os.ReadFile("./site/index.html")
	_, _ = w.Write(data)
}

func server(fs http.FileSystem) http.Handler {
	//Create a FileServer
	fileServer := http.FileServer(fs)
	//return a Handler func
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := path.Clean("./site" + r.URL.Path)
		//check path
		_, err := os.Stat(filePath)
		if err != nil {
			notFound(w)
			return
		}
		//handle request
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Site) Run() {
	siteConfig := config.Conf.Site
	port := siteConfig.SiteBase.ListenPort
	addr := fmt.Sprintf(":%d", port)
	logrus.Fatal(http.ListenAndServe(addr, server(http.Dir("./site"))))
}
