package proxy_cache

import "testing"

func TestCheckWithProxy(t *testing.T) {
	badProxy := "120.195.201.189:80"
	if !checkWithProxy(badProxy) {
		t.Fail()
	}
}
