package cryptox

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- HMAC-SHA1 用于对接要求该算法的第三方接口，HMAC 构造下仍安全
	"fmt"
)

// HmacSha1Encrypt HMAC-SHA1 十六进制签名，用于对接要求该算法的第三方接口；新接口建议使用 HMAC-SHA256
func HmacSha1Encrypt(key string, accessSecret string) string {
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(accessSecret))
	result := mac.Sum(nil)
	return fmt.Sprintf("%x", result)
}
