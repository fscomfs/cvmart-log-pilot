package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cast"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const (
	DefaultFileBeatHost    = "/run/filebeat_minio.sock"
	FileBeatUpload         = "http://localhost/uploadFile"
	API_LOG                = "/api/log"
	API_CHECK_GPU          = "/api/checkGpu"
	API_CONTAINERGPUINFO   = "/api/containerGpuInfo"
	API_DOWNLOADLOG        = "/api/downloadLog"
	API_UPLOADLOGBYTRACKNO = "/api/uploadLogByTrackNo"
	SUCCESS_CODE           = 200
	FAIL_CODE              = 999
)

type BaseResult struct {
	Code   int         `json:"code"`
	Status int         `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
}

var MinioClient *minio.Client
var FileBeatClient *http.Client
var ProxyHttpClient *http.Client
var RemoteProxyUrl *url.URL

func init() {
	if config.GlobConfig.RemoteProxyHost != "" && config.GlobConfig.EnableProxy {
		if proxyUrl, error := url.Parse(config.GlobConfig.RemoteProxyHost); error == nil {
			RemoteProxyUrl = proxyUrl
		} else {
			log.Warnf("remoteProxyHost format error", error)
		}
	}
}

func SUCCESS_RES(msg string, data interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application-json")
	res := BaseResult{
		Code: SUCCESS_CODE,
		Msg:  msg,
		Data: data,
	}
	c, _ := json.Marshal(&res)
	w.Write(c)
}
func FAIL_RES(msg string, data interface{}, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application-json")
	res := BaseResult{
		Code: SUCCESS_CODE,
		Msg:  msg,
		Data: data,
	}
	c, _ := json.Marshal(&res)
	w.Write(c)
}

func NewDockerClient(dockerHost string) (client *docker.Client) {
	if dockerHost != "" {
		if !strings.HasPrefix(dockerHost, "http://") {
			dockerHost = "http://" + dockerHost
		}
		transport := new(http.Transport)
		if hostURL, error := docker.ParseHostURL(dockerHost); error == nil {
			if RemoteProxyUrl != nil {
				transport.Proxy = http.ProxyURL(RemoteProxyUrl)
			}
			sockets.ConfigureTransport(transport, hostURL.Scheme, hostURL.Host)
			httpClient := &http.Client{
				Transport:     transport,
				CheckRedirect: docker.CheckRedirect,
			}
			c, err := docker.NewClient(dockerHost, "", httpClient, nil)
			if err != nil {
				log.Printf("ParseHostURL error %+v", error.Error())
				return nil
			}
			return c
		} else {
			log.Printf("ParseHostURL error %+v", error.Error())
			return nil
		}

	}
	return nil
}

func InitProxyHttpClient() {
	transport := new(http.Transport)
	if RemoteProxyUrl != nil {
		transport.Proxy = http.ProxyURL(RemoteProxyUrl)
	}
	sockets.ConfigureTransport(transport, "http", "")
	httpClient := &http.Client{
		Transport:     transport,
		CheckRedirect: docker.CheckRedirect,
	}
	ProxyHttpClient = httpClient
}

func InitFileBeatClient() {
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", DefaultFileBeatHost)
			},
		},
	}
	FileBeatClient = &httpc
}

func InitMinioClient() {
	if config.GlobConfig.MinioAuth.UserName != "" && config.GlobConfig.MinioAuth.PassWord != "" && config.GlobConfig.MinioUrl != "" && config.GlobConfig.Bucket != "" {
		if client, err := minio.New(config.GlobConfig.MinioUrl, &minio.Options{
			Creds:  credentials.NewStaticV4(config.GlobConfig.MinioAuth.UserName, config.GlobConfig.MinioAuth.PassWord, ""),
			Secure: false,
		}); err != nil {
			fmt.Printf("create minio client error:%+v", err)
		} else {
			MinioClient = client
		}
	}
}

func GetURLByHost(host string) string {
	var hostUrl string
	if strings.HasPrefix(host, "http") {
		hostUrl = strings.TrimSuffix(host, "/") + ":" + cast.ToString(config.GlobConfig.ServerPort)
	} else {
		hostUrl = "http://" + strings.TrimSuffix(host, "/") + ":" + cast.ToString(config.GlobConfig.ServerPort)
	}
	return hostUrl
}

var zz = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
var zzLen = 26 * 2

func LineConfound(line []byte, rsb bool) []byte {
	if rsb {
		rIndex := bytes.LastIndexByte(line, '\r')
		if rIndex > 0 {
			line = line[rIndex+1:]
		}
	}
	lens := len(line)
	if lens > config.GlobConfig.LineMaxSize {
		line = line[:config.GlobConfig.LineMaxSize]
		lens = config.GlobConfig.LineMaxSize
		for i := 0; i < lens/200; i++ {
			s := rand.Intn(zzLen)
			is := rand.Intn(lens)
			line = append(line[:is+1], append([]byte{zz[s]}, line[is+1:]...)...)
		}
	}
	return line
}
