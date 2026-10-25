// Package hashx 哈希工具与一致性哈希。
package hashx

import (
	"crypto/md5" // #nosec G501 -- 见 Md5 注释：仅用于非安全场景
	"fmt"

	"github.com/spaolacci/murmur3"
)

func Hash(data []byte) uint64 {
	return murmur3.Sum64(data)
}

// Md5 MD5 摘要，仅用于校验和、分片等非安全场景
func Md5(data []byte) []byte {
	digest := md5.New() // #nosec G401 -- MD5 仅用于校验和、分片等非安全场景
	digest.Write(data)
	return digest.Sum(nil)
}

func Md5Hex(data []byte) string {
	return fmt.Sprintf("%x", Md5(data))
}
