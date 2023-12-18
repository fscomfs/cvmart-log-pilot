package Service

import (
	"context"
	"encoding/json"
	"github.com/avast/retry-go/v4"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/quota"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"log"
	"net/http"
	"path"
	"strings"
)

type ListModelParam struct {
	HostPath string `json:"hostPath"`
	FileName string `json:"fileName"`
}

type SaveModeParam struct {
	HostPath      string     `json:"hostPath"`
	FileName      string     `json:"fileName"`
	MinioEndpoint string     `json:"minioEndpoint"`
	ObjectBucket  string     `json:"objectBucket"`
	AccessKey     string     `json:"accessKey"`
	SecretKey     string     `json:"secretKey"`
	SaveItem      []SaveItem `json:"saveItem"`
	CallBack      string     `json:"callBack"`
	Async         bool       `json:"async"`
}

type SaveItem struct {
	FileName   string `json:"fileName"`
	ObjectName string `json:"objectName"`
}

type SaveModelResp struct {
	UnSaved      []string `json:"unSaved"`
	Saved        []string `json:"saved"`
	NotFound     []string `json:"notFound"`
	ErrorMessage string   `json:"errorMessage"`
}
type ListModelResp struct {
	Files []utils.FileItem `json:"files"`
}

func ListModelFile(w http.ResponseWriter, r *http.Request) {
	if RequestAndRedirect(w, r) {
		return
	}
	var param ListModelParam
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		utils.FAIL_RES("request param error", nil, w)
		return
	}
	relPath, err := quota.FindRealPath(config.BaseDir, param.HostPath)
	relFile := path.Join(relPath, param.FileName)

	tarFile := utils.TarFile{Path: relFile}
	list, err := tarFile.ListFiles()
	if err != nil {
		utils.FAIL_RES(err.Error(), nil, w)
		return
	}
	res := ListModelResp{
		Files: list,
	}
	utils.SUCCESS_RES("", res, w)
}

func SaveModelFile(w http.ResponseWriter, r *http.Request) {
	if RequestAndRedirect(w, r) {
		return
	}

	var param SaveModeParam
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		utils.FAIL_RES("request param error", nil, w)
		return
	}
	log.Printf("[SaveModelFile] param:%v", param)
	relPath, err := quota.FindRealPath(config.BaseDir, param.HostPath)
	relFile := path.Join(relPath, param.FileName)

	tarFile := utils.TarFile{Path: relFile}
	res := SaveModelResp{}
	secure := strings.HasPrefix(param.MinioEndpoint, "https")
	if strings.HasPrefix(param.MinioEndpoint, "http") {
		param.MinioEndpoint = strings.TrimPrefix(param.MinioEndpoint, "https://")
		param.MinioEndpoint = strings.TrimPrefix(param.MinioEndpoint, "http://")
	}
	client, err := minio.New(param.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(param.AccessKey, param.SecretKey, ""),
		Secure: secure,
	})
	if err != nil {
		utils.FAIL_RES(err.Error(), nil, w)
		return
	}
	doSave := func(cal func(err2 error)) {
		needSaveFileNames := make([]string, len(param.SaveItem))
		for i := range param.SaveItem {
			needSaveFileNames[i] = param.SaveItem[i].FileName
		}
		saved := make([]string, 0)
		unSaved := make([]string, 0)
		notFound := make([]string, 0)
		errorMessage := ""
		if ok, err := client.BucketExists(context.Background(), param.ObjectBucket); err == nil && ok {

		} else {
			client.MakeBucket(context.Background(), param.ObjectBucket, minio.MakeBucketOptions{})
		}
		err = tarFile.ExtractFileTo(needSaveFileNames, func(fileName string, reader io.Reader) {
			objName := ""
			for _, item := range param.SaveItem {
				if item.FileName == fileName {
					objName = item.ObjectName
				}
			}
			err = retry.Do(func() error {
				_, err := client.PutObject(context.Background(), param.ObjectBucket, objName, reader, 0, minio.PutObjectOptions{})
				if err != nil {
					log.Printf("[SaveModelFile] put object error:%v", err)
				}
				return err
			}, retry.Attempts(5), retry.MaxDelay(10))
			if err != nil {
				unSaved = append(unSaved, fileName)
				errorMessage += err.Error() + "\n"
			} else {
				saved = append(saved, fileName)
			}
		})
		if err != nil {
			cal(err)
			return
		}
		res.Saved = saved
		res.UnSaved = unSaved
		res.ErrorMessage = errorMessage
		for _, item := range param.SaveItem {
			exits := false
			for _, s := range saved {
				if s == item.FileName {
					exits = true
				}
			}
			for _, s := range unSaved {
				if s == item.FileName {
					exits = true
				}
			}
			if !exits {
				notFound = append(notFound, item.FileName)
			}
		}
		cal(nil)
	}
	if param.Async {
		go doSave(func(err2 error) {
			if err2 != nil {
				res.ErrorMessage += err2.Error() + "\n"
				utils.GetRetryHttpClient().Post(param.CallBack, "application/json", utils.FAIL_RES("", res, nil))
			}
			if param.CallBack != "" {
				utils.GetRetryHttpClient().Post(param.CallBack, "application/json", utils.SUCCESS_RES("", res, nil))
			}

		})
	} else {
		doSave(func(err2 error) {
			if err != nil {
				utils.FAIL_RES(err.Error(), nil, w)
			} else {
				utils.SUCCESS_RES("", res, w)
			}

		})
	}

}
