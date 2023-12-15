package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	docker "github.com/docker/docker/client"
	"github.com/docker/go-connections/sockets"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/quota"
	retryhttp "github.com/hashicorp/go-retryablehttp"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cast"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

const (
	DefaultFileBeatHost    = "/run/filebeat_minio.sock"
	ENV_HOST_IP            = "ENV_HOST_IP"
	FileBeatUpload         = "http://localhost/uploadFile"
	API_LOG                = "/api/log"
	API_CHECK_GPU          = "/api/checkGpu"
	API_CONTAINERGPUINFO   = "/api/containerGpuInfo"
	API_DOWNLOADLOG        = "/api/downloadLog"
	API_UPLOADLOGBYTRACKNO = "/api/uploadLogByTrackNo"
	API_SETQUOTA           = "/api/setDirQuota"
	API_GETDIRQUOTAINFO    = "/api/getDirQuotaInfo"
	API_GETNODESPACEINFO   = "/api/getNodeSpaceInfo"
	API_RELEASEDIR         = "/api/releaseDir"
	API_GETIMAGEQUOTAINFO  = "/api/getImageQuotaInfo"
	API_FILES              = "/api/listFiles"
	API_FILE               = "/api/file/"
	API_TAIL_FILE          = "/api/tailFile"
	API_LIST_MODEL_FILE    = "/api/listModeFile"
	API_SAVE_MODEL_FILE    = "/api/saveModeFile"
	INTER_TAIL_FILE        = "/inter/tailFile"
	SUCCESS_CODE           = 200
	FAIL_CODE              = 999
)

type BaseResult struct {
	Code   int         `json:"code"`
	Status int         `json:"status"`
	Msg    string      `json:"msg"`
	Data   interface{} `json:"data"`
}

type UploadByParamRes struct {
	trackFlag  int `json:"trackFlag"`
	uploadCode int `json:"uploadCode"`
}

var minioClient *minio.Client
var fileBeatClient *http.Client
var proxyHttpClient *http.Client
var httpClient *http.Client
var remoteProxyUrl *url.URL
var k8sClient *kubernetes.Clientset
var retryHttpClient *retryhttp.Client
var quotaController *quota.Control
var localDockerClient *docker.Client

func InitConfig() {
	if config.GlobConfig.RemoteProxyHost != "" && config.GlobConfig.EnableProxy {
		if proxyUrl, error := url.Parse(config.GlobConfig.RemoteProxyHost); error == nil {
			remoteProxyUrl = proxyUrl
		} else {
			log.Warnf("remoteProxyHost format error", error)
		}
	}
}

func SUCCESS_RES(msg string, data interface{}, w http.ResponseWriter) []byte {
	res := BaseResult{
		Code: SUCCESS_CODE,
		Msg:  msg,
		Data: data,
	}
	c, _ := json.Marshal(&res)
	if w != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(c)
	}
	return c
}
func FAIL_RES(msg string, data interface{}, w http.ResponseWriter) []byte {

	res := BaseResult{
		Code: FAIL_CODE,
		Msg:  msg,
		Data: data,
	}
	c, _ := json.Marshal(&res)
	if w != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(c)
	}
	return c
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
			httpClientTemp := &http.Client{
				Transport:     transport,
				CheckRedirect: docker.CheckRedirect,
			}
			c, err := docker.NewClient(dockerHost, "", httpClientTemp, nil)
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
	httpClientTemp := &http.Client{
		Transport:     transport,
		CheckRedirect: docker.CheckRedirect,
	}
	proxyHttpClient = httpClientTemp

	transport2 := new(http.Transport)
	sockets.ConfigureTransport(transport2, "http", "")
	httpClient2 := &http.Client{
		Transport:     transport,
		CheckRedirect: docker.CheckRedirect,
	}
	httpClient = httpClient2
}

func GetHttpClient(host string) *http.Client {
	if UseProxy(host) {
		return proxyHttpClient
	}
	return httpClient
}

func InitFileBeatClient() {
	httpc := http.Client{
		Timeout: 120 * time.Second,
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

var zz = "#^()abc=defg123898TUVWXYZ*<>_"
var zzLen = len(zz)

func LineConfound(line []byte, rsb bool) []byte {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("LineConfound error:%+v", err)
		}
	}()
	if rsb {
		rIndex := bytes.LastIndexByte(line, '\r')
		if rIndex > 0 {
			line = line[rIndex+1:]
		}
	}
	lens := len(line)
	if lens > config.GlobConfig.LineMaxSize {
		for i := 0; i < lens/600; i++ {
			s := rand.Intn(zzLen - 1)
			is := rand.Intn(lens - 2)
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

func InitRetryHttpClient() {
	retryHttpClient = retryhttp.NewClient()
	httpClientTemp := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	retryHttpClient.HTTPClient = httpClientTemp
	retryHttpClient.RetryMax = 8
	retryHttpClient.RetryWaitMin = 2 * time.Second
	retryHttpClient.RetryWaitMax = 60 * time.Second
	retryHttpClient.CheckRetry = retryhttp.DefaultRetryPolicy
}

func GetRetryHttpClient() *retryhttp.Client {
	return retryHttpClient
}
func GetQuotaControl() (*quota.Control, error) {
	if quotaController == nil {
		return nil, fmt.Errorf("Quota not support")
	}
	return quotaController, nil
}

func InitQuotaController(baseDir string) {
	log.Printf("start Init quota Controller on %s", path.Join(baseDir, config.GlobConfig.HostTempDataPath))
	var err error
	quotaController, err = quota.NewControl(baseDir, config.GlobConfig.HostTempDataPath)
	if err != nil {
		log.Errorf("NewControl error = %s", err.Error())
	} else {
		log.Printf("Success Init quota Controller on %s", path.Join(baseDir, config.GlobConfig.HostTempDataPath))
	}

}

func GetLocalDockerClient() *docker.Client {
	return localDockerClient
}

func InitEnvDockerClient() {
	client, err := docker.NewEnvClient()
	if err != nil {
		log.Printf("new Env DockerClient %+v", err)
	} else {
		localDockerClient = client
	}
}

var LOG_MESSAGE = []byte{'1', '0', '0', '0'}
var LOG_STAT_MESSAGE = []byte{'0', '0', '0', '0'}

func LogMessage(message []byte) []byte {
	return append(LOG_MESSAGE, message...)
}

func LogStatMessage(message []byte) []byte {
	return append(LOG_STAT_MESSAGE, message...)
}
