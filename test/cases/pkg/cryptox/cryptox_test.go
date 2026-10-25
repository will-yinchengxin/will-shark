package cryptox

import (
	"testing"

	"willshark/pkg/hashx"
	"willshark/pkg/stringx"

	"golang.org/x/crypto/bcrypt"
)

func TestEncode(t *testing.T) {
	if stringx.Base64("hi") != "aGk=" || hashx.Md5Encrypt("a") != "0cc175b9c0f1b6a831c399e269772661" {
		t.Fatal("encode")
	}
	if got := HmacSha1Encrypt("key", "msg"); len(got) != 40 {
		t.Fatalf("hmac len %d", len(got))
	}
}

func TestSetPassword(t *testing.T) {
	h, err := SetPassword("p@ssw0rd")
	if err != nil || bcrypt.CompareHashAndPassword(h, []byte("p@ssw0rd")) != nil {
		t.Fatalf("hash err=%v", err)
	}
}
