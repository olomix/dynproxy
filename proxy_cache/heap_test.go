package proxy_cache

import (
	"testing"
	"time"
)

func TestRecheckInEpoch(t *testing.T) {
	p := Proxy{
		Addr: "test",
		lastCheck: time.Time{},
		failCounter: 0,
	}
	x := recheckIn(&p)
	if x != 0 {
		t.Fatalf("x = %v", x)
	}
}

func TestRecheckInAlmost(t *testing.T) {
	p := Proxy{
		Addr: "test",
		lastCheck: time.Now().UTC().Add(-3 * time.Minute),
		failCounter: 0,
	}
	x := recheckIn(&p)
	if !(x <= (2 * time.Minute) && x > (2 * time.Minute) - time.Second) {
		t.Fatalf("x = %v", x)
	}
}

func TestRecheckInExp(t *testing.T) {
	p := Proxy{
		Addr: "test",
		lastCheck: time.Now().UTC().Add(-3 * time.Minute),
		failCounter: 3,
	}
	x := recheckIn(&p)
	if !(x <= (37 * time.Minute) && x > (37 * time.Minute) - time.Second) {
		t.Fatalf("x = %v", x)
	}
}

func TestRecheckInMax1(t *testing.T) {
	p := Proxy{
		Addr: "test",
		lastCheck: time.Now().UTC().Add(-3 * time.Minute),
		failCounter: 28,
	}
	x := recheckIn(&p)
	expectedDate := (24 * time.Hour) - (3 * time.Minute)
	if !(x <= expectedDate && x > expectedDate - time.Second) {
		t.Fatalf("x = %v", x)
	}
}

func TestRecheckInMax2(t *testing.T) {
	p := Proxy{
		Addr: "test",
		lastCheck: time.Now().UTC().Add(-3 * time.Minute),
		failCounter: 80,
	}
	x := recheckIn(&p)
	expectedDate := (24 * time.Hour) - (3 * time.Minute)
	if !(x <= expectedDate && x > expectedDate - time.Second) {
		t.Fatalf("x = %v", x)
	}
}
