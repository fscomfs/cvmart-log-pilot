package Service

import (
	"bytes"
	"encoding/json"
	"github.com/fscomfs/cvmart-log-pilot/pod_file"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"io"
	"log"
	"net/http"
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
		utils.GetPodFileExporter().GetPodFiles(r.Context(), param.PodName, param.ContainerId, param.ImageName, param.ContainerPath, param.HostPath, w)
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
			io.Copy(w, resp.Body)
		} else {
			utils.FAIL_RES(err.Error(), nil, w)
		}
	}
}

func PodFileHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	paths := strings.Split(path, "/")
	if len(paths) > 4 {
		paramStr := paths[len(paths)-2]
		p := &pod_file.FileURLParam{}
		err := json.Unmarshal([]byte(paramStr), p)
		if err != nil {
			http.Error(w, "", http.StatusNotFound)
			return
		}
		if p.Host == "localhost" {
			utils.GetPodFileExporter().GetPodFile(p.Path, w, r)
		} else {
			host := p.Host
			p.Host = "localhost"
			j, _ := json.Marshal(p)
			t, _ := auth.GeneratorJWTToken(j)
			url := utils.GetURLByHost(host) + utils.API_FILE + t + paths[len(paths)-1]
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
				io.Copy(w, resp.Body)
			} else {
				http.Error(w, "", http.StatusNotFound)
				return
			}
		}

	}
}
