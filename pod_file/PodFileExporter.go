package pod_file

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/container_log"
	"github.com/fscomfs/cvmart-log-pilot/quota"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"io"
	"io/fs"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type PodFileExporter struct {
	BaseDir string
}

type FileURLParam struct {
	Host    string `json:"host"`
	Path    string `json:"path"`
	ModTime int64  `json:"modTime"`
}

type FileRes struct {
	FileName string `json:"fileName"`
	ModTime  int64  `json:"modTime"`
	URL      string `json:"url"`
}

var auth = container_log.AESAuth{}

func (p PodFileExporter) GetPodFiles(ctx context.Context, podName string, containerId string, imageName string, containerPath string, hostPath string, w http.ResponseWriter) {
	var path string
	var err error
	if hostPath != "" {
		path, err = quota.FindRealPath(p.BaseDir, hostPath)
		if err != nil {
			utils.FAIL_RES("", nil, w)
			return
		}
	} else {
		containerRootPath, err := getContainerDiffPath(ctx, p.BaseDir, podName, containerId, imageName)
		if err != nil {
			return
		}

		path = filepath.Join(containerRootPath, containerPath)
	}
	var fileInfos []fs.FileInfo
	filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			f, err := d.Info()
			if err != nil {

			}
			fileInfos = append(fileInfos, f)
		}
		return nil
	})
	var fileUrl []FileRes
	for _, info := range fileInfos {
		f := FileURLParam{
			Host:    getHostIp(),
			Path:    info.Name(),
			ModTime: info.ModTime().UnixMilli(),
		}
		jsonStr, _ := json.Marshal(f)
		t, _ := auth.GeneratorJWTToken(jsonStr)
		fileUrl = append(fileUrl, FileRes{
			FileName: info.Name(),
			ModTime:  info.ModTime().UnixMilli(),
			URL:      utils.API_FILE + t + "/" + info.Name(),
		})
	}
	utils.SUCCESS_RES("", fileUrl, w)
}

func (p PodFileExporter) GetPodFile(file string, w http.ResponseWriter, r *http.Request) {
	// Get request range.
	var rs *utils.HTTPRangeSpec
	var rangeErr error
	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		rs, rangeErr = utils.ParseRequestRangeSpec(rangeHeader)
		if rangeErr == utils.ErrInvalidRange {
			return
		}
	}
	fileInfo, err := os.Stat(file)
	if err != nil {

	}
	var start, rangeLen int64
	lastModified := fileInfo.ModTime().UTC().Format(http.TimeFormat)
	w.Header().Set("Last-Modified", lastModified)
	if rs != nil {
		// For providing ranged content
		start, rangeLen, err = rs.GetOffsetLength(fileInfo.Size())
		if err != nil {
			return
		}
		// Set content length.
		w.Header().Set("Content-Length", strconv.FormatInt(rangeLen, 10))
		if rs != nil {
			contentRange := fmt.Sprintf("bytes %d-%d/%d", start, start+rangeLen-1, fileInfo.Size())
			w.Header().Set("Content-Range", contentRange)
		}
	} else {
		start, rangeLen, err = rs.GetOffsetLength(fileInfo.Size())
	}

	fileFd, err := os.OpenFile(file, os.O_RDONLY, 0600)
	defer fileFd.Close()
	if err != nil {

	}
	_, _ = fileFd.Seek(start, 0)
	reader := &io.LimitedReader{R: io.Reader(fileFd), N: rangeLen}
	if _, err := io.Copy(w, reader); err != nil {

	}

}

func getContainerDiffPath(ctx context.Context, baseDir, podName string, containerId string, imageName string) (string, error) {
	var containerDir string
	if podName != "" {
		listOption := v1.ListOptions{
			Watch: false,
		}
		listOption.FieldSelector = "metadata.name=" + podName
		listOption.LabelSelector = "app=" + podName
		podList, err := utils.GetK8sClient().CoreV1().Pods("default").List(context.Background(), listOption)
		if err != nil || len(podList.Items) == 0 {

		} else {
			pod := podList.Items[0]
			containerId = pod.Status.ContainerStatuses[0].ContainerID
		}

	}
	var targetPath string
	if containerId == "" {
		imgInspcet, _, err := utils.GetLocalDockerClient().ImageInspectWithRaw(ctx, imageName)
		if err == nil {
			for key, val := range imgInspcet.GraphDriver.Data {
				if key == "UpperDir" {
					targetPath = val
				}
			}
		}
	} else {
		container, err := utils.GetLocalDockerClient().ContainerInspect(ctx, containerId)
		if err != nil {
			return "", err
		} else {
			for key, val := range container.GraphDriver.Data {
				if key == "UpperDir" {
					targetPath = val
				}
			}
		}
	}
	if targetPath == "" {
		return "", fmt.Errorf("path not found")
	}
	containerDir, err := quota.FindRealPath(baseDir, targetPath)
	if err != nil {
		return "", err
	}
	return containerDir, nil
}

func getHostIp() string {
	envIp := os.Getenv(utils.ENV_HOST_IP)
	if envIp != "" {
		return envIp
	}
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	return addrs[0]
}
