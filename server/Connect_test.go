package server

import (
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"os"
	"testing"
	"time"
)

func TestRegistryConnect(t *testing.T) {
	claims := &LogClaims{
		//Host:        "localhost",
		//Port:        "2375",
		//ContainerId: "285fe975d7de",
		//	MinioObjName: "uuid2/name2.log",
		PodLabel: "log-test",
		Operator: OPERATOR_LOG,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
		},
	}
	os.Setenv("JWT_SEC", "111")
	SecretKey = "111"
	token, _ := GeneratorToken(claims)
	fmt.Println(token)
	a, _ := Auth(token)
	if a != nil {
		fmt.Println(a)
	}
	//if err == nil {
	//	u := url.URL{Scheme: "ws", Host: "127.0.0.1:888", Path: "/log"}
	//	c, _, err := websocket.DefaultDialer.Dial(u.String()+"?id=123&token="+token, nil)
	//	if err != nil {
	//		fmt.Print(err)
	//		return
	//	}
	//	for {
	//		_, message, err := c.ReadMessage()
	//		if err != nil {
	//			log.Println("read:", err)
	//			return
	//		}
	//		log.Printf("recv: %s", message)
	//	}
	//	defer c.Close()
	//}

}
