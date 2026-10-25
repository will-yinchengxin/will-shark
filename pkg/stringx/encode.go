package stringx

import "encoding/base64"

func Base64(str string) string {
	return base64.StdEncoding.EncodeToString([]byte(str))
}
