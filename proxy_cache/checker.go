package proxy_cache

import (
	"bufio"
	"container/heap"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
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
	goodProxyList GoodProxyList
}

func NewProxyCache(proxyFileName string) ProxyCache {
	cache := &CacheContext{
		proxies:       readProxiesFromFile(proxyFileName),
		checkPoolSize: new(int64),
		goodProxyList: NewGoodProxyList(),
	}
	heap.Init(&cache.proxies)
	go worker(cache)
	return cache
}

func (cc *CacheContext) NextProxy() string {
	panic("not implemented error")
}

func (cc *CacheContext) Stop() {
	panic("not implemented error")
}

func worker(pc *CacheContext) {
	for {
		pc.lock.RLock()
		l := pc.proxies.Len()
		pc.lock.RUnlock()
		if l == 0 {
			continue
		}

		pc.lock.Lock()
		proxy := heap.Pop(&pc.proxies).(Proxy)
		pc.lock.Unlock()

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
			checkPoolSize = atomic.LoadInt64(pc.checkPoolSize)
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
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(fmt.Sprintf("http://%s", proxy.Addr))
			},
		},
	}
	resp, err := client.Get("http://lomaka.org.ua/t.txt")
	proxy.lastCheck = time.Now().UTC()
	if err == nil {
		defer resp.Body.Close()
		out, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Error: %v", err)
		}
		if err == nil && string(out) == "6b5f2815-5c7a-4970-99f1-8eb290564ddc\n" {
			log.Printf("Proxy %v check OK", proxy.Addr)
			proxy.failCounter = 0
			return
		}

	}
	log.Printf("Proxy %v check failed", proxy.Addr)
	proxy.failCounter++
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
