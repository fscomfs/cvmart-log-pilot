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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
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

var minioClient *minio.Client
var fileBeatClient *http.Client
var proxyHttpClient *http.Client
var remoteProxyUrl *url.URL
var k8sClient *kubernetes.Clientset

func InitConfig() {
	if config.GlobConfig.RemoteProxyHost != "" && config.GlobConfig.EnableProxy {
		if proxyUrl, error := url.Parse(config.GlobConfig.RemoteProxyHost); error == nil {
			remoteProxyUrl = proxyUrl
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
		Code: FAIL_CODE,
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

			sockets.ConfigureTransport(transport, hostURL.Scheme, hostURL.Host)
			if UseProxy(strings.TrimPrefix(dockerHost, "http://")) {
				transport.Proxy = http.ProxyURL(remoteProxyUrl)
			}
			httpClient := &http.Client{
				Transport:     transport,
				Timeout:       45 * time.Second,
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
	if remoteProxyUrl != nil {
		transport.Proxy = http.ProxyURL(remoteProxyUrl)
	}
	sockets.ConfigureTransport(transport, "http", "")
	httpClient := &http.Client{
		Transport:     transport,
		Timeout:       45 * time.Second,
		CheckRedirect: docker.CheckRedirect,
	}
	proxyHttpClient = httpClient
}

func GetHttpClient(host string) *http.Client {
	return proxyHttpClient
}

func InitFileBeatClient() {
	httpc := http.Client{
		Timeout: 45 * time.Second,
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", DefaultFileBeatHost)
			},
		},
	}
	fileBeatClient = &httpc
}
func GetFileBeatClient() *http.Client {
	return fileBeatClient
}

func InitMinioClient() {
	if config.GlobConfig.MinioAuth.UserName != "" && config.GlobConfig.MinioAuth.PassWord != "" && config.GlobConfig.MinioUrl != "" && config.GlobConfig.Bucket != "" {
		if client, err := minio.New(config.GlobConfig.MinioUrl, &minio.Options{
			Creds:  credentials.NewStaticV4(config.GlobConfig.MinioAuth.UserName, config.GlobConfig.MinioAuth.PassWord, ""),
			Secure: false,
		}); err != nil {
			fmt.Printf("create minio client error:%+v", err)
		} else {
			minioClient = client
		}
	}
}

func GetMinioClient() *minio.Client {
	return minioClient
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
func GetK8sClient() *kubernetes.Clientset {
	return k8sClient
}
func InitK8sClient() {
	c, err := rest.InClusterConfig()
	if err != nil {
		log.Info("create k8s client from config")
		c = &rest.Config{
			Host:        config.GlobConfig.KubeApiUrl,
			BearerToken: config.GlobConfig.KubeAuth.Token,
			TLSClientConfig: rest.TLSClientConfig{
				Insecure: true,
			},
		}
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		panic(err.Error())
	}
	k8sClient = clientset
}

func UseProxy(host string) bool {
	if config.GlobConfig.EnableProxy && !config.GlobConfig.ProxyGlobal && config.GlobConfig.ProxyHostPattern != "" {
		regex, err := regexp.Compile(config.GlobConfig.ProxyHostPattern)
		if err != nil {
			return true
		}
		return regex.MatchString(host)
	}
	if config.GlobConfig.RemoteProxyHost != "" && config.GlobConfig.ProxyGlobal {
		return true
	}
	return false
}

func GetProxy(host string) func(*http.Request) (*url.URL, error) {
	if UseProxy(host) {
		return http.ProxyURL(remoteProxyUrl)
	} else {
		return nil
	}

}
