package Service

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"net/http"
	"net/http/httputil"
	url "net/url"
)

func RequestAndRedirect(w http.ResponseWriter, r *http.Request) (res bool, err error) {
	target := r.Header.Get("Target_host")
	if target != "" {
		res = true
		url := &url.URL{}
		url.Scheme = "http"
		host := fmt.Sprintf("%s:%d", target, config.GlobConfig.ServerPort)
		url.Host = host
		proxy := httputil.NewSingleHostReverseProxy(url)
		r.Header.Del("Target_host")
		proxy.ServeHTTP(w, r)
	} else {
		timestamp := r.URL.Query().Get("timestamp")
		pass := false
		if timestamp == "" {
			pass = false
		} else {
			hash := md5.New()
			hash.Write([]byte(timestamp + "_" + config.GlobConfig.SecretKey))
			hashBytes := hash.Sum(nil)
			hashString := hex.EncodeToString(hashBytes)
			if hashString != r.URL.Query().Get("sign") {
				pass = false
			} else {
				pass = true
			}
		}
		if !pass {
			utils.FAIL_RES("auth fail", nil, w)
			err = fmt.Errorf("auth fail")
		}
		res = false
	}
	return res, err
}
