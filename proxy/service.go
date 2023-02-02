package services

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/utils"
	"github.com/spf13/cast"
	"log"
	"runtime/debug"
)

func init() {
	httpArgs := HTTPArgs{}
	timeOut := 5000
	local := "0.0.0.0:" + cast.ToString(utils.ProxyPort)
	parent := ""
	auth := []string{}
	localType := "tcp"
	httpArgs.HTTPTimeout = &timeOut
	httpArgs.Timeout = &timeOut
	httpArgs.Args.Local = &local
	httpArgs.Args.Parent = &parent
	httpArgs.AuthFile = &parent
	httpArgs.Auth = &auth
	httpArgs.LocalType = &localType
	Register("http", NewHTTP(), httpArgs)
}

type Service interface {
	Start(args interface{}) (err error)
	Clean()
}
type ServiceItem struct {
	S    Service
	Args interface{}
	Name string
}

var servicesMap = map[string]*ServiceItem{}

func Register(name string, s Service, args interface{}) {
	servicesMap[name] = &ServiceItem{
		S:    s,
		Args: args,
		Name: name,
	}
}
func Run(name string) (service *ServiceItem, err error) {
	service, ok := servicesMap[name]
	if ok {
		go func() {
			defer func() {
				err := recover()
				if err != nil {
					log.Fatalf("%s servcie crashed, ERR: %s\ntrace:%s", name, err, string(debug.Stack()))
				}
			}()
			err := service.S.Start(service.Args)
			if err != nil {
				log.Fatalf("%s servcie fail, ERR: %s", name, err)
			}
		}()
	}
	if !ok {
		err = fmt.Errorf("service %s not found", name)
	}
	return
}
