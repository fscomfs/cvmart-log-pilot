package Service

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"net/http"
	"net/http/httputil"
	url "net/url"
)

func RequestAndRedirect(w http.ResponseWriter, r *http.Request) (res bool) {
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
		res = false
	}
	return res
}
