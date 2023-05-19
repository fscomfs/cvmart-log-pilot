package container_log

import (
	"github.com/fscomfs/cvmart-log-pilot/config"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

type JwtAuth struct {
}

func Auth(token string) (*LogParam, error) {
	var SecretKey = config.GlobConfig.SecretKey
	tokenClaims, err := jwt.ParseWithClaims(token, &LogParam{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := tokenClaims.Claims.(*LogParam); ok && tokenClaims.Valid {
		return claims, nil
	}
	return nil, nil
}

func GeneratorToken(logParam LogParam, live int) (string, error) {
	var SecretKey = config.GlobConfig.SecretKey
	logParam.RegisteredClaims = jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second * time.Duration(live))),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, logParam)
	return token.SignedString([]byte(SecretKey))
}
