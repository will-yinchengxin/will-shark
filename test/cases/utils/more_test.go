package utils

import (
	"errors"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestEncode(t *testing.T) {
	if Base64("hi") != "aGk=" || Md5Encrypt("a") != "0cc175b9c0f1b6a831c399e269772661" {
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

func TestValidateIP(t *testing.T) {
	cases := map[string]bool{
		"10.0.0.1": true, "::1": true, "10.0.0.0/8": true, "10.0.0.1~10.0.0.9": true,
		"10.0.0.0/33": false, "a~b": false, "1.1.1.1~": false, "x": false, "": false,
	}
	for in, want := range cases {
		if got := ValidateIP(in); got != want {
			t.Errorf("ValidateIP(%q) = %v", in, got)
		}
	}
}

func TestLimit(t *testing.T) {
	l := NewLimit(1)
	if !l.TryBorrow() || l.TryBorrow() {
		t.Fatal("borrow")
	}
	if l.Return() != nil || !errors.Is(l.Return(), ErrLimitReturn) {
		t.Fatal("return")
	}
}

func TestUUIDAndRet(t *testing.T) {
	id := GetProUUID("order", "x")
	if !strings.HasPrefix(id, "order-x-") || len(id) != len("order-x-")+32 {
		t.Fatalf("id = %s", id)
	}
	if SetRetVal("k", 1)["k"] != 1 {
		t.Fatal("ret")
	}
}

func TestGetTimeSeries(t *testing.T) {
	ok, err := GetTimeSeries([][]string{
		{"2024-03-03 00:00:00", "2024-03-03 01:00:00"},
		{"2024-03-02 00:00:00", "2024-03-02 01:00:00"},
	})
	if err != nil || len(ok) != 2 || !ok[0].StartTime.Before(ok[1].StartTime) {
		t.Fatalf("sorted series: %v %v", ok, err)
	}
	for name, in := range map[string][][]string{
		"overlap":     {{"2024-03-02 00:00:00", "2024-03-02 02:00:00"}, {"2024-03-02 01:00:00", "2024-03-02 03:00:00"}},
		"too early":   {{"2023-01-01 00:00:00", "2023-01-01 01:00:00"}},
		"bad format":  {{"2024/03/02", "2024-03-02 01:00:00"}},
		"bad end fmt": {{"2024-03-02 00:00:00", "x"}},
	} {
		if _, err := GetTimeSeries(in); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}

func TestTimeFormat(t *testing.T) {
	ts := ParseTimeStrToInt("2024-03-02 10:20:30")
	if ParseTimeIntToStr(ts) != "2024-03-02 10:20:30" || ParseTimeIntToStrOnlyDay(ts) != "2024-03-02" {
		t.Fatal("round trip")
	}
	if Now() <= 0 || NowData().IsZero() {
		t.Fatal("now")
	}
	if _, err := AgoUnixMilli("3q"); err == nil {
		t.Fatal("unknown unit")
	}
	for _, u := range []string{"1h", "1d", "1w", "1m", "1y"} {
		if v, err := AgoUnixMilli(u); err != nil || v >= time.Now().UnixMilli() {
			t.Errorf("%s: %d %v", u, v, err)
		}
	}
}

func TestCopyStructFields(t *testing.T) {
	type src struct {
		Name string
		Age  int
	}
	type dst struct{ Name string }
	var d dst
	if err := CopyStructFields(&d, src{Name: "a", Age: 1}); err != nil || d.Name != "a" {
		t.Fatalf("d=%+v err=%v", d, err)
	}
}

func TestFuncName(t *testing.T) {
	if GetSelfFuncName() != "TestFuncName" {
		t.Fatal(GetSelfFuncName())
	}
	if err := WithMessage(errors.New("x"), "ctx"); !strings.Contains(err.Error(), "ctx: x") {
		t.Fatal(err)
	}
	ErrorLog(nil)
}

func TestNetworkAndPoolHelpers(t *testing.T) {
	ips, err := GetIps()
	if err != nil {
		t.Fatal(err)
	}
	for _, ip := range ips {
		if !ValidateIP(ip) {
			t.Fatalf("invalid ip %s", ip)
		}
	}
	if ip := InternalIP(); ip != "" && !ValidateIP(ip) {
		t.Fatalf("internal ip %q", ip)
	}

	l := NewLimit(1)
	l.Borrow()
	if l.TryBorrow() {
		t.Fatal("limit exceeded")
	}

	p, _ := NewPool(2)
	if p.Cap() != 2 || p.Free() != 2 || p.Running() != 0 {
		t.Fatal("pool stats")
	}
	p.Release()
	p.Reboot()
	if err := p.Submit(func() {}); err != nil {
		t.Fatalf("rebooted pool must accept tasks: %v", err)
	}
	p.Wait()
	if _, err := NewPool(-1); err != nil {
		t.Logf("negative size: %v", err)
	}
	if GetRandFloat64() < 0 || GetRandFloat32() < 0 || GetRandInt32() < 0 {
		t.Fatal("rand ranges")
	}
}

func TestErrorLogNonNil(t *testing.T) {
	ErrorLog(errors.New("logged"))
}
