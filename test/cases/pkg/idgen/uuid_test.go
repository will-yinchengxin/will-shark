package idgen

import (
	"strings"
	"testing"
	"willshark/pkg/mapx"
)

func TestUUIDAndRet(t *testing.T) {
	id := GetProUUID("order", "x")
	if !strings.HasPrefix(id, "order-x-") || len(id) != len("order-x-")+32 {
		t.Fatalf("id = %s", id)
	}
	if mapx.SetRetVal("k", 1)["k"] != 1 {
		t.Fatal("ret")
	}
}
