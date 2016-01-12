package main

import (
	"fmt"
	"github.com/olomix/dynproxy/proxy_cache"
)

func main() {
	proxyList := proxy_cache.LoadCache()
	for i := range proxyList {
		fmt.Println(proxyList[i].String())
	}
}
