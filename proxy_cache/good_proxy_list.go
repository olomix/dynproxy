package proxy_cache

import (
	"errors"
	"sync"
)

type GoodProxyList struct {
	lock    sync.RWMutex
	proxies []string
	nextIdx int
}

func NewGoodProxyList() GoodProxyList {
	return GoodProxyList{}
}

var ProxyListEmpty = errors.New("Proxy list is empty")

func (gpl *GoodProxyList) next() (string, error) {
	gpl.lock.RLock()
	defer gpl.lock.RUnlock()
	if len(gpl.proxies) == 0 {
		return "", ProxyListEmpty
	}

	if gpl.nextIdx >= len(gpl.proxies) {
		gpl.nextIdx = 0
	}
	var out string = gpl.proxies[gpl.nextIdx]
	gpl.nextIdx++
	return out, nil
}

func (gpl *GoodProxyList) append(proxyAddr string) {
	gpl.lock.Lock()
	gpl.proxies = append(gpl.proxies, proxyAddr)
	gpl.lock.Unlock()
}

func (gpl *GoodProxyList) remove(proxyAddr string) {
	gpl.lock.Lock()
	idx := len(gpl.proxies)
	for i := range gpl.proxies {
		if gpl.proxies[i] == proxyAddr {
			idx = i
			break
		}
	}
	if idx < len(gpl.proxies) {
		gpl.proxies[idx] = gpl.proxies[len(gpl.proxies)-1]
		gpl.proxies = gpl.proxies[:len(gpl.proxies)-1]
	}
	gpl.lock.Unlock()
}
