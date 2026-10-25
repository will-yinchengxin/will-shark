package utils

import (
	"crypto/hmac"
	"crypto/md5"  // #nosec G501 -- 见 Md5Encrypt 注释：仅用于非安全场景
	"crypto/sha1" // #nosec G505 -- HMAC-SHA1 用于对接要求该算法的第三方接口，HMAC 构造下仍安全
	"encoding/base64"
	"fmt"
	"io"
)

func Base64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}

// Md5Encrypt 返回 MD5 十六进制摘要。仅用于校验和、兼容第三方签名；不可用于密码存储（用 SetPassword）
func Md5Encrypt(str string) string {
	w := md5.New() // #nosec G401 -- MD5 仅用于校验和与兼容第三方签名，不用于安全场景
	_, err := io.WriteString(w, str)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", w.Sum(nil))
}

// HmacSha1Encrypt HMAC-SHA1 十六进制签名，用于对接要求该算法的第三方接口；新接口建议使用 HMAC-SHA256
func HmacSha1Encrypt(key string, accessSecret string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(accessSecret))
	result := mac.Sum(nil)
	return fmt.Sprintf("%x", result)
}
