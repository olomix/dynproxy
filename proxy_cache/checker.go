package proxy_cache

import (
	"bufio"
	"container/heap"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"net/http"
	"io/ioutil"
)

const proxyCheckPool = 1000
const proxyCheckTimeoutMin = 5 * time.Minute
const proxyCheckTimeoutMax = 24 * time.Hour

type Proxy struct {
	Addr        string
	lastCheck   time.Time
	failCounter uint
}

type ProxyCache interface {
	Stop()
	NextProxy() string
}

type CacheContext struct {
	lock          sync.RWMutex
	proxies       ProxyHeap
	checkPoolSize *int64
}

func NewProxyCache(proxyFileName string) ProxyCache {
	cache := &CacheContext{
		proxies:       readProxiesFromFile(proxyFileName),
		checkPoolSize: new(int64),
	}
	heap.Init(&cache.proxies)
	go worker(cache)
	return cache
}

func (cc *CacheContext) NextProxy() string{
	panic("not implemented error")
}

func (cc *CacheContext) Stop() {
	panic("not implemented error")
}

func worker(pc *CacheContext) {
	for {
		proxy := heap.Pop(&pc.proxies).(Proxy)

		// If nearest proxy checking in some time, wait for this time
		waitFor := recheckIn(&proxy)
		if waitFor > 0 {
			log.Printf("There is %v to check. Sleep for now.", waitFor)
			time.Sleep(waitFor)
		}

		// If worker pool is full, wait for some time
		var checkPoolSize int64 = atomic.LoadInt64(pc.checkPoolSize)
		for checkPoolSize >= proxyCheckPool {
			log.Print("Checking pool is full. Wait for one second.")
			time.Sleep(time.Second)
		}

		// Start checking gorotine
		go pc.checkProxy(proxy)
	}
}

func (pc *CacheContext) checkProxy(proxy Proxy) {
	atomic.AddInt64(pc.checkPoolSize, 1)
	defer func() {
		atomic.AddInt64(pc.checkPoolSize, -1)
		pc.lock.Lock()
		heap.Push(&pc.proxies, proxy)
		pc.lock.Unlock()
	}()

	client := &http.Client{
		Transport: &http.Transport{},
	}
	resp, err := client.Get("http://lomaka.org.ua/t.txt")
	if err != nil {
		log.Printf("Error: %v", err)
	}
	out, err := ioutil.ReadAll(resp.Body)
	proxy.lastCheck = time.Now().UTC()
	if err != nil || string(out) != "6b5f2815-5c7a-4970-99f1-8eb290564ddc" {
		log.Printf("Proxy %v check failed", proxy.Addr)
		proxy.failCounter++
	} else {
		log.Printf("Proxy %v check OK", proxy.Addr)
		proxy.failCounter = 0
	}
	defer resp.Body.Close()
}

// Read proxies from input file. One address:port per line.
func readProxiesFromFile(proxyFileName string) []Proxy {
	var file *os.File
	var err error
	file, err = os.Open(proxyFileName)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	var reader *bufio.Scanner = bufio.NewScanner(file)
	var result []Proxy = make([]Proxy, 0)
	for reader.Scan() {
		result = append(
			result,
			Proxy{
				Addr:        reader.Text(),
				failCounter: 1, // by default proxy is BAD
			},
		)
	}
	return result
}
