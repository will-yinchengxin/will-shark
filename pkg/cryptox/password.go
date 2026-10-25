// Package cryptox 提供密码哈希与 HMAC 签名工具。
package cryptox

import (
	"willshark/consts"

	"golang.org/x/crypto/bcrypt"
)

func SetPassword(password string) ([]byte, error) {
	newPass, err := bcrypt.GenerateFromPassword([]byte(password), consts.PasswordDifficult)
	return newPass, err
}
