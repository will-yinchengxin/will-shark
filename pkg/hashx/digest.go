package hashx

import (
	"crypto/md5" // #nosec G501 -- 见 Md5Encrypt 注释：仅用于非安全场景
	"fmt"
	"io"
)

// Md5Encrypt 返回 MD5 十六进制摘要。仅用于校验和、兼容第三方签名；密码存储使用 cryptox.SetPassword。
func Md5Encrypt(str string) string {
	w := md5.New() // #nosec G401 -- MD5 仅用于校验和与兼容第三方签名，不用于安全场景
	_, err := io.WriteString(w, str)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", w.Sum(nil))
}
