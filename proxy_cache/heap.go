package proxy_cache

import (
	"hash/fnv"
	"time"
	"log"
)

type ProxyHeap []Proxy

func (h ProxyHeap) Len() int           { return len(h) }
func (h ProxyHeap) Less(i, j int) bool { return isLess(&h[i], &h[j]) }
func (h ProxyHeap) Swap(i, j int)      {
	log.Printf("Swap %v & %v", i, j)
	h[i], h[j] = h[j], h[i]
}

func (h *ProxyHeap) Push(x interface{}) {
	// Push and Pop use pointer receivers because they modify the slice's
	// length, not just its contents.
	*h = append(*h, x.(Proxy))
}

func (h *ProxyHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

func isLess(left, right *Proxy) bool {
	leftCheckIn := recheckIn(left)
	rightCheckIn := recheckIn(right)
	if leftCheckIn == rightCheckIn {
		return stringHash(left.Addr) < stringHash(right.Addr)
	} else {
		return leftCheckIn < rightCheckIn
	}
}

// Return duration in which we need to recheck proxy
func recheckIn(proxy *Proxy) time.Duration {
	now := time.Now().UTC()
	checkInMax := proxy.lastCheck.Add(proxyCheckTimeoutMax)

	// Catch integer overflow
	failCounter := proxy.failCounter
	if failCounter > 30 {
		failCounter = 30
	}

	checkIn := proxy.lastCheck.Add(proxyCheckTimeoutMin * (1 << failCounter))
	switch {
	case checkIn.Before(now):
		return time.Duration(0)
	case checkIn.After(checkInMax):
		return checkInMax.Sub(now)
	default:
		return checkIn.Sub(now)
	}
}

func stringHash(in string) uint64 {
	f := fnv.New64a()
	f.Write([]byte(in))
	return f.Sum64()
}
