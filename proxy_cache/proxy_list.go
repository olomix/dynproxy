package proxy_cache

// Proxy list sorted by address. Use to quick find in slice

type ProxyList []Proxy

func (h ProxyList) Len() int           { return len(h) }
func (h ProxyList) Less(i, j int) bool { return h[i].Addr < h[j].Addr }
func (h ProxyList) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
