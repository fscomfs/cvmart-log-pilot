package container_log

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/fscomfs/cvmart-log-pilot/config"
	"io"
)

type AESAuth struct {
}

func (a *AESAuth) Auth(token string) (r *LogParam, e error) {
	defer func() {
		if err := recover(); err != nil {
			r = nil
			e = fmt.Errorf("invalin token")
		}
	}()
	t, e := base64.StdEncoding.DecodeString(token)
	if e != nil {
		return nil, e
	}
	block, e := aes.NewCipher([]byte(config.GlobConfig.SecretKey))
	if e != nil {
		return nil, e
	}
	iv := t[:block.BlockSize()]
	cipherText := t[block.BlockSize():]
	model := cipher.NewCBCDecrypter(block, iv)
	plainText := make([]byte, len(cipherText))
	model.CryptBlocks(plainText, cipherText)
	plainText, _ = UnPaddingPKCS7(plainText)
	res := &LogParam{}
	json.Unmarshal(plainText, res)
	if res.isExpiration() {
		return nil, fmt.Errorf("token expried")
	} else {
		return res, nil
	}
}

func (a *AESAuth) GeneratorToken(logParam LogParam) (string, error) {
	paramJson, err := json.Marshal(logParam)
	if err == nil {
		block, _ := aes.NewCipher([]byte(config.GlobConfig.SecretKey))
		iv := make([]byte, block.BlockSize())
		if _, e := io.ReadFull(rand.Reader, iv); e != nil {

		}
		model := cipher.NewCBCEncrypter(block, iv)
		paramJson = PaddingPKCS7(paramJson, block.BlockSize())
		chiperText := make([]byte, len(paramJson))
		model.CryptBlocks(chiperText, paramJson)
		entrypted := append(iv, chiperText...)
		encroded := base64.StdEncoding.EncodeToString(entrypted)
		return encroded, nil

	}
	return "", nil
}

func PaddingPKCS7(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	p := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, p...)
}

func UnPaddingPKCS7(data []byte) ([]byte, error) {
	dataLen := len(data)
	padding := int(data[dataLen-1])
	if padding > dataLen {
		return data, nil
	}
	return data[:dataLen-padding], nil
}
