package syncx

import (
	"errors"
	"testing"
	"willshark/pkg/netx"
	"willshark/pkg/randx"
)

func TestLimit(t *testing.T) {
	l := NewLimit(1)
	if !l.TryBorrow() || l.TryBorrow() {
		t.Fatal("borrow")
	}
	if l.Return() != nil || !errors.Is(l.Return(), ErrLimitReturn) {
		t.Fatal("return")
	}
}

func TestNetworkAndPoolHelpers(t *testing.T) {
	ips, err := netx.GetIps()
	if err != nil {
		t.Fatal(err)
	}
	for _, ip := range ips {
		if !netx.ValidateIP(ip) {
			t.Fatalf("invalid ip %s", ip)
		}
	}
	if ip := netx.InternalIP(); ip != "" && !netx.ValidateIP(ip) {
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
	if randx.GetRandFloat64() < 0 || randx.GetRandFloat32() < 0 || randx.GetRandInt32() < 0 {
		t.Fatal("rand ranges")
	}
}
