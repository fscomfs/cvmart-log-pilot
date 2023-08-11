package Service

import (
	"bytes"
	"encoding/json"
	"github.com/avast/retry-go/v4"
	"github.com/fscomfs/cvmart-log-pilot/quota"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

type QuotaParam struct {
	AuthParam   AuthParam `json:"authParam"`
	Token       string    `json:"token"`
	ImageName   string    `json:"imageName"`
	TargetPath  string    `json:"targetPath"`
	Quota       uint64    `json:"quota"`
	CallBackUrl string    `json:"callBackUrl"`
	Async       int       `json:"async"`
}

type AuthParam struct {
	Host           string `json:"host"`
	ExpirationTime int64  `json:"expirationTime"`
}

func (p *AuthParam) isExpiration() bool {
	if p.ExpirationTime > time.Now().UnixMilli() {
		return false
	}
	return true
}

func GetImageDiskQuotaInfoHandler(w http.ResponseWriter, r *http.Request) {
	param := QuotaParam{}
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Printf("Get Image Quota Info param fail %+v", err)
	}
	res, err := auth.AuthJWTToken(param.Token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	hostParam := &AuthParam{}
	e := json.Unmarshal(res, hostParam)
	if e != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(e.Error()))
		return
	}
	if hostParam.Host == "localhost" {
		quotaControl, err := utils.GetQuotaControl()
		if err != nil {
			utils.FAIL_RES(err.Error(), param, w)
			return
		}
		c := utils.GetLocalDockerClient()
		if c == nil {
			utils.FAIL_RES("client not init", param, w)
			return
		}
		if quotaInfo, err := quotaControl.GetImageDiskQuotaInfo(param.ImageName, c); err == nil {
			utils.SUCCESS_RES("success", quotaInfo, w)
		} else {
			utils.FAIL_RES(err.Error(), nil, w)
		}
	} else {
		host := param.AuthParam.Host
		param.AuthParam = AuthParam{
			Host:           "localhost",
			ExpirationTime: time.Now().UnixMilli() + 1000*2,
		}
		j, _ := json.Marshal(param.AuthParam)
		t, e := auth.GeneratorJWTToken(j)
		param.Token = t
		jsonString, err := json.Marshal(param)
		if err != nil {
			log.Printf("Get Image Quota Info  marshal error %+v", err)
		}
		if e != nil {
			w.WriteHeader(http.StatusUnauthorized)
			utils.FAIL_RES(err.Error(), nil, w)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_GETIMAGEQUOTAINFO
		requestError := retry.Do(func() error {
			resp, err2 := utils.GetHttpClient(host).Post(url, "application/json", bytes.NewBuffer(jsonString))
			if err2 == nil {
				r, _ := ioutil.ReadAll(resp.Body)
				if param.Async == 0 {
					io.Copy(w, bytes.NewReader(r))
				}
				go callback(param.CallBackUrl, 1, r)
			}
			return err2
		},
			retry.Attempts(3),
			retry.Delay(20*time.Second),
		)
		if requestError != nil {
			var resBody []byte
			w.WriteHeader(http.StatusBadRequest)
			resBody = utils.FAIL_RES(requestError.Error(), nil, w)
			go callback(param.CallBackUrl, 0, resBody)
			log.Printf("request GetImageDiskQuotaInfoHandler proxy error %+v", err)
		}
	}
}

func GetNodeSpaceInfoHandler(w http.ResponseWriter, r *http.Request) {
	param := QuotaParam{}
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Printf("get Node Space Info quota param fail %+v", err)
	}
	res, err := auth.AuthJWTToken(param.Token)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	hostParam := &AuthParam{}
	e := json.Unmarshal(res, hostParam)
	if e != nil {
		w.Write([]byte(e.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if hostParam.Host == "localhost" {
		quotaControl, err := utils.GetQuotaControl()
		if err != nil {
			utils.FAIL_RES(err.Error(), param, w)
			return
		}
		if quotaInfo, err := quotaControl.GetNodeSpaceInfo(param.TargetPath); err == nil {
			utils.SUCCESS_RES("success", quotaInfo, w)
		} else {
			utils.FAIL_RES(err.Error(), nil, w)
		}
	} else {
		host := param.AuthParam.Host
		param.AuthParam = AuthParam{
			Host: "localhost",
		}
		j, _ := json.Marshal(param.AuthParam)
		t, e := auth.GeneratorJWTToken(j)
		param.Token = t
		jsonString, err := json.Marshal(param)
		if err != nil {
			log.Printf("uploadLogByTrackNo marshal error %+v", err)
		}
		if e != nil {
			utils.FAIL_RES(err.Error(), nil, w)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_GETNODESPACEINFO
		requestError := retry.Do(func() error {
			resp, err2 := utils.GetHttpClient(host).Post(url, "application/json", bytes.NewBuffer(jsonString))
			if err2 == nil {
				r, _ := ioutil.ReadAll(resp.Body)
				if param.Async == 0 {
					io.Copy(w, bytes.NewReader(r))
				}
				go callback(param.CallBackUrl, 1, r)
			}
			return err2
		},
			retry.Attempts(3),
			retry.Delay(20*time.Second),
		)
		if requestError != nil {
			var resBody []byte
			resBody = utils.FAIL_RES(requestError.Error(), nil, w)
			w.WriteHeader(http.StatusBadRequest)
			go callback(param.CallBackUrl, 0, resBody)
			log.Printf("request SetDirQuotaHandler proxy error %+v", err)
		}
	}
}

func ReleaseDirHandler(w http.ResponseWriter, r *http.Request) {
	param := QuotaParam{}
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Printf("Set Dir quota param fail %+v", err)
	}
	res, err := auth.AuthJWTToken(param.Token)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	hostParam := &AuthParam{}
	e := json.Unmarshal(res, hostParam)
	if e != nil {
		w.Write([]byte(e.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if hostParam.Host == "localhost" {
		quotaControl, err := utils.GetQuotaControl()
		if err != nil {
			utils.FAIL_RES(err.Error(), param, w)
			return
		}
		if err := quotaControl.ReleaseDir(param.TargetPath); err == nil {
			utils.SUCCESS_RES("success", param, w)
		} else {
			utils.FAIL_RES(err.Error(), nil, w)
		}
	} else {
		host := param.AuthParam.Host
		param.AuthParam = AuthParam{
			Host: "localhost",
		}
		j, _ := json.Marshal(param.AuthParam)
		t, _ := auth.GeneratorJWTToken(j)
		param.Token = t
		if err != nil {
			utils.FAIL_RES(err.Error(), nil, w)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		doReleaseFunc := func(param QuotaParam) {
			jsonString, err := json.Marshal(param)
			if err != nil {
				log.Printf("ReleaseDir marshal error %+v", err)
			}
			url := utils.GetURLByHost(host) + utils.API_RELEASEDIR
			requestError := retry.Do(func() error {
				resp, err2 := utils.GetHttpClient(host).Post(url, "application/json", bytes.NewBuffer(jsonString))
				if err2 == nil {
					r, _ := ioutil.ReadAll(resp.Body)
					if param.Async == 0 {
						io.Copy(w, bytes.NewReader(r))
					}
					go callback(param.CallBackUrl, 1, r)
				}
				return err2
			},
				retry.Attempts(3),
				retry.Delay(20*time.Second),
			)
			if requestError != nil {
				var resBody []byte
				resBody = utils.FAIL_RES(requestError.Error(), nil, w)
				if param.Async == 0 {
					w.WriteHeader(http.StatusBadRequest)
				}
				go callback(param.CallBackUrl, 0, resBody)
				log.Printf("request ReleaseDir proxy error %+v", err)
			}
		}

		if param.Async > 0 {
			go doReleaseFunc(param)
			utils.SUCCESS_RES("request success", param, w)
		} else {
			doReleaseFunc(param)
		}
	}
}

func SetDirQuotaHandler(w http.ResponseWriter, r *http.Request) {
	param := QuotaParam{}
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Printf("Set Dir quota param fail %+v", err)
	}
	res, err := auth.AuthJWTToken(param.Token)
	if err != nil {
		log.Printf("Set Dir quota auth  fail %+v", err)
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	hostParam := &AuthParam{}
	e := json.Unmarshal(res, hostParam)
	if e != nil {
		log.Printf("Set Dir quota auth  fail %+v", err)
		w.Write([]byte(e.Error()))
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if hostParam.Host == "localhost" {
		quotaControl, err := utils.GetQuotaControl()
		if err != nil {
			utils.FAIL_RES(err.Error(), param, w)
			return
		}
		if err := quotaControl.SetDirQuota(param.TargetPath, quota.Quota{Size: param.Quota}); err == nil {
			utils.SUCCESS_RES("success", param, w)
		} else {
			log.Printf("Set Dir quota error %+v", err)
			utils.FAIL_RES(err.Error(), param, w)
		}
	} else {
		host := param.AuthParam.Host
		param.AuthParam = AuthParam{
			Host: "localhost",
		}
		j, _ := json.Marshal(param.AuthParam)
		t, e := auth.GeneratorJWTToken(j)
		param.Token = t
		jsonString, err := json.Marshal(param)
		if err != nil {
			log.Printf("Set quota marshal error %+v", err)
		}
		if e != nil {
			utils.FAIL_RES(err.Error(), nil, w)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_SETQUOTA
		requestError := retry.Do(func() error {
			resp, err2 := utils.GetHttpClient(host).Post(url, "application/json", bytes.NewBuffer(jsonString))
			if err2 == nil {
				r, _ := ioutil.ReadAll(resp.Body)
				if param.Async == 0 {
					io.Copy(w, bytes.NewReader(r))
				}
				go callback(param.CallBackUrl, 1, r)
			}
			return err2
		},
			retry.Attempts(3),
			retry.Delay(20*time.Second),
		)
		if requestError != nil {
			var resBody []byte
			resBody = utils.FAIL_RES(requestError.Error(), nil, w)
			w.WriteHeader(http.StatusBadRequest)
			go callback(param.CallBackUrl, 0, resBody)
			log.Printf("request SetDirQuotaHandler proxy error %+v", err)
		}
	}

}

func GetDirQuotaInfoHandler(w http.ResponseWriter, r *http.Request) {
	param := QuotaParam{}
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		log.Printf("Set Dir quota param fail %+v", err)
	}
	res, err := auth.AuthJWTToken(param.Token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(err.Error()))
		return
	}
	hostParam := &AuthParam{}
	e := json.Unmarshal(res, hostParam)
	if e != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(e.Error()))
		return
	}
	if hostParam.Host == "localhost" {
		quotaControl, err := utils.GetQuotaControl()
		if err != nil {
			utils.FAIL_RES(err.Error(), param, w)
			return
		}
		if quotaInfo, err := quotaControl.GetDirQuota(param.TargetPath); err == nil {
			utils.SUCCESS_RES("success", quotaInfo, w)
		} else {
			utils.FAIL_RES(err.Error(), nil, w)
		}
	} else {
		host := param.AuthParam.Host
		param.AuthParam = AuthParam{
			Host:           "localhost",
			ExpirationTime: time.Now().UnixMilli() + 1000*2,
		}
		j, _ := json.Marshal(param.AuthParam)
		t, e := auth.GeneratorJWTToken(j)
		param.Token = t
		jsonString, err := json.Marshal(param)
		if err != nil {
			log.Printf("Get dir quota Info marshal error %+v", err)
		}
		if e != nil {
			w.WriteHeader(http.StatusUnauthorized)
			utils.FAIL_RES(err.Error(), nil, w)
			return
		}
		url := utils.GetURLByHost(host) + utils.API_GETDIRQUOTAINFO
		requestError := retry.Do(func() error {
			resp, err2 := utils.GetHttpClient(host).Post(url, "application/json", bytes.NewBuffer(jsonString))
			if err2 == nil {
				r, _ := ioutil.ReadAll(resp.Body)
				if param.Async == 0 {
					io.Copy(w, bytes.NewReader(r))
				}
				go callback(param.CallBackUrl, 1, r)
			}
			return err2
		},
			retry.Attempts(3),
			retry.Delay(20*time.Second),
		)
		if requestError != nil {
			var resBody []byte
			w.WriteHeader(http.StatusBadRequest)
			resBody = utils.FAIL_RES(requestError.Error(), nil, w)
			go callback(param.CallBackUrl, 0, resBody)
			log.Printf("request SetDirQuotaHandler proxy error %+v", err)
		}
	}
}

func callback(url string, status int, res []byte) {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("callback request error %+v", err)
		}
	}()
	if url != "" {
		log.Printf("do callback url:%+v,--status:%+v", url, status)
		utils.GetRetryHttpClient().Post(url, "application/json", res)
	}
}
