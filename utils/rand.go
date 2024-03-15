package utils

import (
	"math/rand"
	"time"
)

func GetRandFloat64() float64 {
	return NewRand().Float64()
}

func GetRandFloat32() float32 {
	return NewRand().Float32()
}

func GetRandInt32() int32 {
	return NewRand().Int31()
}

func GetRandInt64() int64 {
	return NewRand().Int63()
}

func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}
