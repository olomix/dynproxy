package proxy_cache

import (
	"testing"
	"reflect"
)

func TestGoodProxyList(t *testing.T) {
	gpl := NewGoodProxyList()
	gpl.append("one")

	if !reflect.DeepEqual(gpl.proxies, []string {"one"}) {
		t.Fatal("proxies = %v", gpl.proxies)
	}

	gpl.append("two")
	if !reflect.DeepEqual(gpl.proxies, []string {"one", "two"}) {
		t.Fatal("proxies = %v", gpl.proxies)
	}

	gpl.append("three")
	if !reflect.DeepEqual(gpl.proxies, []string {"one", "two", "three"}) {
		t.Fatal("proxies = %v", gpl.proxies)
	}

	n, err := gpl.next()
	if err != nil {
		t.Fatal(err)
	}
	if n != "one" {
		t.Fatal(n)
	}

	gpl.remove("two")
	n, err = gpl.next()
	if err != nil {
		t.Fatal(err)
	}
	if n != "three" {
		t.Fatal(n)
	}
}
