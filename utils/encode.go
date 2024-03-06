package utils

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
)

func Base64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

func Md5Encrypt(str string) string {
	w := md5.New()
	_, err := io.WriteString(w, str)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", w.Sum(nil))
}

func HmacSha1Encrypt(key string, accessSecret string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(accessSecret))
	result := mac.Sum(nil)
	return fmt.Sprintf("%x", result)
}
