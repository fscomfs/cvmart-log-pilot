package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/fscomfs/cvmart-log-pilot/pilot"
	proxy "github.com/fscomfs/cvmart-log-pilot/proxy"
	"github.com/fscomfs/cvmart-log-pilot/server"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
)

func main() {
	template := flag.String("template", "", "Template filepath for fluentd or filebeat.")
	base := flag.String("base", "", "Directory which mount host root.")
	level := flag.String("log-level", "INFO", "Log level")
	proxyFlag := flag.Bool("enable_proxy", true, "Enable proxy")
	remoteProxyHost := flag.String("remote_proxy_host", "", "proxy host")
	flag.Parse()
	if *remoteProxyHost != "" {
		if proxyUrl, error := url.Parse(*remoteProxyHost); error == nil {
			utils.RemoteProxyUrl = proxyUrl
		} else {
			log.Warnf("remoteProxyHost format error")
		}
	}

	baseDir, err := filepath.Abs(*base)
	if err != nil {
		panic(err)
	}

	if baseDir == "/" {
		baseDir = ""
	}

	if *template == "" {
		panic("template file can not be empty")
	}

	log.SetOutput(os.Stdout)
	logLevel, err := log.ParseLevel(*level)
	if err != nil {
		panic(err)
	}
	log.SetLevel(logLevel)

	b, err := ioutil.ReadFile(*template)
	if err != nil {
		panic(err)
	}

	configInit()

	go server.Handler()

	if *proxyFlag {
		go proxy.Run("http")
	}

	log.Fatal(pilot.Run(string(b), baseDir))
}

func configInit() {
	utils.InitProxyHttpClient()
	utils.InitMinioClient()
	utils.InitFileBeatClient()
}
