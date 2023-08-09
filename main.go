package main

import (
	"flag"
	log "github.com/Sirupsen/logrus"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/pilot"
	proxy "github.com/fscomfs/cvmart-log-pilot/proxy"
	"github.com/fscomfs/cvmart-log-pilot/server"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	template := flag.String("template", "/config/filebeat/filebeat.tpl", "Template filepath for fluentd or filebeat.")
	base := flag.String("base", "", "Directory which mount host root.")
	configFilePath := flag.String("config", "/etc/cvmart/daemon-config.json", "config info")
	level := flag.String("log-level", "INFO", "Log level")
	flag.Parse()
	config.ParseFromFile(*configFilePath)
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
	configInit(baseDir)
	go server.Handler()
	if config.GlobConfig.EnableProxy {
		proxy.InitProxy()
		go proxy.Run("http")
	}
	log.Printf("ListenAndServe %+v", config.GlobConfig.ServerPort)
	log.Fatal(pilot.Run(string(b), baseDir))

}

func configInit(baseDir string) {
	utils.InitConfig()
	utils.InitProxyHttpClient()
	utils.InitMinioClient()
	utils.InitFileBeatClient()
	utils.InitK8sClient()
	utils.InitRetryHttpClient()
	utils.InitQuotaController(baseDir)
}
