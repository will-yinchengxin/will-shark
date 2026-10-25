// Package randx 提供随机数与随机字符串工具。
package randx

import (
	crand "crypto/rand"
	"encoding/binary"
	"math/rand"
	randv2 "math/rand/v2"
)

// 以下随机数仅用于非安全场景（抽样、抖动、打散等）；生成令牌、密钥请使用 crypto/rand

func GetRandFloat64() float64 {
	return randv2.Float64() // #nosec G404 -- 非安全用途的随机数，见文件注释
}

func GetRandFloat32() float32 {
	return randv2.Float32() // #nosec G404 -- 非安全用途的随机数，见文件注释
}

func GetRandInt32() int32 {
	return randv2.Int32() // #nosec G404 -- 非安全用途的随机数，见文件注释
}

func GetRandInt64() int64 {
	return randv2.Int64() // #nosec G404 -- 非安全用途的随机数，见文件注释
}

// NewRand 返回独立的伪随机数生成器（非并发安全），种子来自 crypto/rand
func NewRand() *rand.Rand {
	var seed [8]byte
	_, _ = crand.Read(seed[:])
	return rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(seed[:]) >> 1))) // #nosec G404 -- 非安全用途的伪随机数生成器，种子来自 crypto/rand
}
