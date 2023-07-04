package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
)

var GlobConfig Config

type Config struct {
	ServerPort       int        `json:"server_port"`
	SecretKey        string     `json:"secret_key"`
	DockerServerPort int        `json:"docker_server_port"`
	DockerAuth       AuthConfig `json:"docker_auth"`
	KubeApiUrl       string     `json:"kube_api_url"`
	KubeAuth         AuthConfig `json:"kube_auth"`
	RemoteProxyHost  string     `json:"remote_proxy_host"`
	EnableProxy      bool       `json:"enable_proxy"`
	ProxyGlobal      bool       `json:"proxy_global"`
	ProxyHostPattern string     `json:"proxy_host_pattern"`
	MinioUrl         string     `json:"minio_url"`
	Bucket           string     `json:"bucket"`
	MinioAuth        AuthConfig `json:"minio_auth"`
	PilotType        string     `json:"pilot_type"`
	PilotLogPrefix   string     `json:"pilot_log_prefix"`
	LineMaxSize      int        `json:"line_max_size"`
	ProxyPort        int        `json:"proxy_port"`
}

type AuthConfig struct {
	UserName string `json:"user_name"`
	PassWord string `json:"pass_word"`
	Token    string `json:"token"`
}

func ParseFromFile(filePath string) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(fmt.Errorf("config file open fail"))
	} else {
		log.Println("config:", string(data))
	}
	json.Unmarshal(data, &GlobConfig)
	if GlobConfig.LineMaxSize == 0 {
		GlobConfig.LineMaxSize = 4 * 1024
	}
}

func init() {

}
