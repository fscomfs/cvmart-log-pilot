package server

import (
	"github.com/golang-jwt/jwt/v4"
	"os"
)

var SecretKey = os.Getenv("JWT_SEC")

type LogClaims struct {
	Host         string `json:"host"`
	Port         string `json:"port"`
	ContainerId  string `json:"containerId"`
	Operator     string `json:"operator"`
	Tail         string `json:"tail"`
	PodLabel     string `json:"podLabel"`
	MinioObjName string `json:"minioObjName"`
	jwt.RegisteredClaims
}

func Auth(token string) (*LogClaims, error) {
	tokenClaims, err := jwt.ParseWithClaims(token, &LogClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tokenClaims.Claims.(*LogClaims); ok && tokenClaims.Valid {
		return claims, nil
	}
	return nil, nil
}

func GeneratorToken(claims *LogClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(SecretKey))
}
