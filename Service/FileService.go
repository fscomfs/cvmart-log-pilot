package Service

import (
	"bytes"
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/pod_file"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"io"
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

var podFileExporter *pod_file.PodFileExporter

func InitPodFileExporter(baseDir string) {
	podFileExporter = &pod_file.PodFileExporter{
		BaseDir: baseDir,
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
		req, err := http.NewRequest(r.Method, url, r.Body)
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
