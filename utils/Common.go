package utils

import (
	"context"
	"encoding/json"
	"fmt"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cast"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	ServerPort             = 5465
	ProxyPort              = 5466
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

var MinioUrl = os.Getenv("MINIO_URL")
var Bucket = os.Getenv("BUCKET")
var MinioUsername = os.Getenv("MINIO_USERNAME")
var MinioPassword = os.Getenv("MINIO_PASSWORD")
var RemoteProxyUrl *url.URL
var MinioClient *minio.Client
var FileBeatClient *http.Client
var ProxyHttpClient *http.Client

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
	if MinioUsername != "" && MinioPassword != "" && MinioUrl != "" && Bucket != "" {
		if client, err := minio.New(MinioUrl, &minio.Options{
			Creds:  credentials.NewStaticV4(MinioUsername, MinioPassword, ""),
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
		hostUrl = strings.TrimSuffix(host, "/") + ":" + cast.ToString(ServerPort)
	} else {
		hostUrl = "http://" + strings.TrimSuffix(host, "/") + ":" + cast.ToString(ServerPort)
	}
	return hostUrl
}

var zz = "zbcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
var lineSizeMax = 4 * 1024

func LineConfound(line string) string {
	t := line
	lens := len(line)
	if lens > lineSizeMax {
		s := lens / lineSizeMax
		w := rand.Intn(len(zz) - 2)
		for i := 0; i <= s; i++ {
			r := rand.Intn(lens)
			t = t[0:r] + zz[w:w+2] + t[r:]
		}
	}
	return t
}
