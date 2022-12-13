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
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * 4)),
		},
	}
	_, err := GeneratorToken(claims)
	if err != nil {
		t.Fatalf("GeneratorToken err:%+v", err)
	}
	time.Sleep(3 * time.Second)
	s, err := Auth("eyJhbGciOiJIUzI1NiJ9.eyJwb2RMYWJlbCI6ImxvZy10ZXN0IiwiZXhwIjoxNjcwOTA1MDcxLCJpYXQiOjE2NzA5MDQ5OTksIm9wZXJhdG9yIjoibG9nIn0.OCe-8rJ3yWo3FzE3eKZ2exWvFflB7_7SPb6YS7fUZ8s")
	if err != nil {
		t.Errorf("err:%+v", err)
	} else {
		t.Logf("loginInfo:%+v", *s)
	}

	log.Print(time.Now().Unix())
}
