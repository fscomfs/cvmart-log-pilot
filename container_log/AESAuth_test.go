package container_log

import (
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"testing"
	"time"
)

func TestGeneratorToken2(t *testing.T) {

	a := AESAuth{}
	claims := LogParam{
		Host:           "localhost",
		Operator:       OPERATOR_LOG,
		Tail:           "50000",
		ContainerId:    "a2e31b25d36e",
		ExpirationTime: time.Now().UnixMilli() + int64(100*1000),
	}
	config.ParseFromFile("/etc/cvmart/daemon-config.json")
	auth, err := a.GeneratorToken(claims)
	if err != nil {

	}
	fmt.Printf("token:%+v\n", auth)

	p, _ := a.Auth(auth)

	fmt.Printf("auth:%+v\n", *p)

}
