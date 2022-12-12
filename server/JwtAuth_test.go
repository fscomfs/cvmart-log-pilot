package server

import (
	"github.com/golang-jwt/jwt/v4"
	"log"
	"testing"
	"time"
)

func TestAuth(t *testing.T) {
	Auth("")
}

func TestGeneratorToken(t *testing.T) {
	claims := &LogClaims{
		Host: "localhost",
		Port: "2375",
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Unix() + 4,
		},
	}
	token, err := GeneratorToken(claims)
	if err != nil {
		t.Fatalf("GeneratorToken err:%+v", err)
	}
	time.Sleep(3 * time.Second)
	s, err := Auth(token)
	if err != nil {
		t.Errorf("err:%+v", err)
	} else {
		t.Logf("loginInfo:%+v", *s)
	}

	log.Print(time.Now().Unix())
}
