package proxy_cache

import (
	"bufio"
	"container/heap"
	"fmt"
	"github.com/olomix/dynproxy/log"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const proxyCheckPool = 100
const proxyCheckTimeoutMin = 5 * time.Minute
const proxyCheckTimeoutMax = 24 * time.Hour

type Proxy struct {
	Addr        string
	lastCheck   time.Time
	failCounter uint
}

type ProxyCache interface {
	Stop()
	NextProxy() (string, error)
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

func (cc *CacheContext) NextProxy() (string, error) {
	return cc.goodProxyList.next()
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
			log.Debug("There is %v to check. Sleep for now.", waitFor)
			time.Sleep(waitFor)
		}

		// If worker pool is full, wait for some time
		var checkPoolSize int64 = atomic.LoadInt64(pc.checkPoolSize)
		for checkPoolSize >= proxyCheckPool {
			log.Debug("Checking pool is full. Wait for one second.")
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

	pc.lock.RLock()
	var proxyAddr string = proxy.Addr
	pc.lock.RUnlock()

	// long operation, put locking after it
	var checkResult bool = checkWithProxy(proxyAddr)

	pc.lock.Lock()
	if checkResult {
		log.Debugf("Proxy %v check OK", proxy.Addr)
		if proxy.failCounter != 0 {
			pc.goodProxyList.append(proxy.Addr)
			proxy.failCounter = 0
		}
	} else {
		log.Debugf("Proxy %v check failed", proxy.Addr)
		if proxy.failCounter == 0 {
			pc.goodProxyList.remove(proxy.Addr)
		}
		proxy.failCounter++
	}
	proxy.lastCheck = time.Now().UTC()
	pc.lock.Unlock()
}

func checkWithProxy(addr string) (result bool) {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(fmt.Sprintf("http://%s", addr))
			},
		},
	}
	var req *http.Request
	var resp *http.Response
	var err error

	req, err = http.NewRequest("GET", "http://lomaka.org.ua/t.txt", nil)
	if err != nil {
		log.Errorf("Can't create request: %v", err)
		return false
	}

	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Proxy-Connection", "Keep-Alive")
	resp, err = client.Do(req)
	if err != nil {
		log.Debugf("Can't connect to %v: %v", addr, err)
		return false
	}

	defer resp.Body.Close()

	out, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Debugf("Can't read from proxy %v: %v", addr, err)
		return false
	}
	return string(out) == "6b5f2815-5c7a-4970-99f1-8eb290564ddc\n"
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
