package Service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/pod_file"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/gorilla/websocket"
	"io"
	coreV1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
	url2 "net/url"
	"strings"
	"time"
)

type FileInfo struct {
	CreatedAt time.Time `json:"createdAt"`
	FileName  string    `json:"fileName"`
}

type FilesReq struct {
	Token         string `json:"token"`
	PodName       string `json:"podName"`
	ContainerId   string `json:"containerId"`
	ImageName     string `json:"imageName"`
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
}

type TailParam struct {
	ContainerPath string `json:"containerPath"`
	PodLabel      string `json:"podLabel"`
	Namespace     string `json:"namespace"`
}

type TailInterParam struct {
	ContainerPath string `json:"containerPath"`
	ContainerId   string `json:"containerId"`
}

var podFileExporter *pod_file.PodFileExporter

func InitPodFileExporter(baseDir string) {
	podFileExporter = &pod_file.PodFileExporter{
		BaseDir: baseDir,
	}
}

func TailFileInterHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	containerId := values.Get("containerId")
	containerPath := values.Get("containerPath")
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("RequestHandler upgrader %+v", err)
		return
	}
	ctx, cancelFunc := context.WithCancel(r.Context())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				_, err := GetPodFileExporter().TailContainerFile(ctx, containerId, containerPath, conn)
				if err != nil {
					log.Printf("tail log error=%+v", err)
				}
				time.Sleep(time.Second * 8)
			}
		}
	}()
	readLoop(conn)
	cancelFunc()

}

func readLoop(conn *websocket.Conn) {
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
func TailFileHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()
	token := values.Get("token")
	res, err := auth.AuthJWTToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	tailParam := &TailParam{}
	err = json.Unmarshal(res, tailParam)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("RequestHandler upgrader %+v", err)
		return
	}
	ctx, cancelFunc := context.WithCancel(context.Background())
	waitingtime := 0
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				//async exec
				listOption := v1.ListOptions{
					Watch:         false,
					LabelSelector: "app=" + tailParam.PodLabel,
				}
				podList, err := utils.GetK8sClient().CoreV1().Pods(tailParam.Namespace).List(ctx, listOption)
				host := ""
				containerId := ""
				var pod *coreV1.Pod = nil
				if err == nil && len(podList.Items) > 0 {
					pod = &podList.Items[0]
					if host == "" {
						host = pod.Status.HostIP
						containerId = strings.TrimPrefix(pod.Status.ContainerStatuses[0].ContainerID, "docker://")
					}
				}
				if containerId != "" {
					if waitingtime > 0 {
						conn.WriteMessage(websocket.BinaryMessage, utils.LogMessage([]byte(fmt.Sprintf("\r                                                                "))))
					}
					_, err := doRemoteTail(ctx, conn, host, containerId, tailParam.ContainerPath)
					if err != nil {
						log.Printf("do Remote tail file error %+v", err)
					}
				} else {
					waitingtime += 5
					conn.WriteMessage(websocket.BinaryMessage, utils.LogMessage([]byte(fmt.Sprintf("\rwaiting container start %ds", waitingtime))))
				}
				time.Sleep(time.Second * 5)
			}
		}
	}()
	readLoop(conn)
	cancelFunc()
}

func doRemoteTail(ctx context.Context, conn *websocket.Conn, host string, containerId string, containerPath string) (bool, error) {
	host = host + fmt.Sprintf(":%d", config.GlobConfig.ServerPort)
	u := url2.URL{Scheme: "ws", Host: host, Path: utils.INTER_TAIL_FILE}
	d := websocket.Dialer{
		Proxy:            utils.GetProxy(host),
		HandshakeTimeout: 45 * time.Second,
	}
	remoteConn, r2, err := d.DialContext(ctx, u.String()+"?containerId="+url2.QueryEscape(containerId)+"&containerPath="+url2.QueryEscape(containerPath), nil)
	if err != nil {
		if r2 != nil && r2.StatusCode == http.StatusNoContent {
			return false, fmt.Errorf("not content")
		}
		log.Printf("doRemoteTail err:%+v", err)
		return true, err
	}
	defer remoteConn.Close()
	go func() {
		select {
		case <-ctx.Done():
			log.Printf("tail file end")
			remoteConn.Close()
			return
		}
	}()
	for {
		_, content, err := remoteConn.ReadMessage()
		if err != nil {
			return true, err
		}
		conn.WriteMessage(websocket.BinaryMessage, content)
	}
}

func GetPodFileExporter() *pod_file.PodFileExporter {
	return podFileExporter
}
func PodFilesHandler(w http.ResponseWriter, r *http.Request) {
	param := FilesReq{}
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Printf("List container files fail %+v", err)
	}
	res, err := auth.AuthJWTToken(param.Token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	hostParam := &AuthParam{}
	err = json.Unmarshal(res, hostParam)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	if hostParam.Host == "localhost" {
		files, err := GetPodFileExporter().GetPodFiles(r.Context(), param.PodName, param.ContainerId, param.ImageName, param.ContainerPath, param.HostPath)
		if err != nil {
			utils.FAIL_RES(err.Error(), nil, w)
			return
		} else {
			utils.SUCCESS_RES("", files, w)
			return
		}
	} else {
		host := hostParam.Host
		hostParam := AuthParam{
			Host:           "localhost",
			ExpirationTime: time.Now().UnixMilli() + 1000*2,
		}
		j, _ := json.Marshal(hostParam)
		t, err := auth.GeneratorJWTToken(j)
		if err != nil {
			log.Printf("PodFilesHandler marshal error %+v", err)
		}
		param.Token = t
		jsonString, err := json.Marshal(param)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			utils.FAIL_RES(err.Error(), nil, w)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_FILES
		req, err := http.NewRequest(r.Method, url, bytes.NewBuffer(jsonString))
		if err != nil {
			http.Error(w, "Error creating request", http.StatusInternalServerError)
		}
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		resp, err := utils.GetHttpClient(host).Do(req)
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		} else {
			utils.FAIL_RES(err.Error(), nil, w)
		}
	}
}

func PodFileHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, utils.API_FILE)
	token := r.URL.Query().Get("token")
	p := &pod_file.FileURLParam{}
	res, err := auth.AuthJWTToken(token)
	err = json.Unmarshal(res, p)
	if err != nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	if p.Host == "localhost" {
		GetPodFileExporter().GetPodFile(p.Path, w, r)
	} else {
		host := p.Host
		p.Host = "localhost"
		j, _ := json.Marshal(p)
		t, _ := auth.GeneratorJWTToken(j)
		url := utils.GetURLByHost(host) + utils.API_FILE + url2.QueryEscape(path) + "?token=" + url2.QueryEscape(t)
		req, err := http.NewRequestWithContext(r.Context(), r.Method, url, r.Body)
		if err != nil {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		resp, err := utils.GetHttpClient(host).Do(req)
		if err == nil {
			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			if resp.StatusCode == 301 {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(resp.StatusCode)
			}
			io.Copy(w, resp.Body)
		} else {
			http.Error(w, "", http.StatusNotFound)
			return
		}
	}
}
