package proxy_cache

import (
	"bufio"
	"container/heap"
	"encoding/gob"
	"fmt"
	"github.com/olomix/dynproxy/log"
	"github.com/olomix/dynproxy/stats"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const proxyCheckPool = 100
const proxyCheckTimeoutMin = 5 * time.Minute
const proxyCheckTimeoutMax = 24 * time.Hour
const autoSaveTimeout = 10 * time.Second
const autoSaveFilename = ".dynproxy.save"

type ProxyCache interface {
	Stop()
	NextProxy() (string, error)
}

type CacheContext struct {
	lock          sync.RWMutex
	proxies       ProxyHeap
	checkPoolSize *int64
	goodProxyList GoodProxyList
	saveLock      sync.Mutex
	grs           *stats.GoRoutineStats
}

func NewProxyCache(proxyFileName string, grs *stats.GoRoutineStats) ProxyCache {
	cache := &CacheContext{
		proxies:       readProxiesFromFile(proxyFileName),
		checkPoolSize: new(int64),
		goodProxyList: NewGoodProxyList(),
		grs:           grs,
	}
	heap.Init(&cache.proxies)
	for i := range cache.proxies {
		if cache.proxies[i].failCounter == 0 {
			cache.goodProxyList.append(cache.proxies[i].Addr)
		}
	}
	log.Debugf(
		"%d proxies in good state",
		len(cache.goodProxyList.proxies))
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
	var lastSaved time.Time

	for {
		lastSaved = saveProxyList(lastSaved, pc)
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
			pc.lock.Lock()
			heap.Push(&pc.proxies, proxy)
			pc.lock.Unlock()

			timeToSleep := autoSaveTimeout
			log.Debugf(
				"There is %v to check. Sleep for %v now.",
				waitFor, timeToSleep)
			time.Sleep(timeToSleep)

			continue
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

func saveProxyList(lastSaved time.Time, pc *CacheContext) time.Time {
	if time.Now().Add(-autoSaveTimeout).Before(lastSaved) {
		return lastSaved
	}

	pc.lock.RLock()
	pc.saveLock.Lock()

	defer func() {
		pc.lock.RUnlock()
		pc.saveLock.Unlock()
	}()

	var backup string = fmt.Sprintf("%s.old", autoSaveFilename)
	//	var isBackedUp bool
	var err error
	if _, err = os.Stat(autoSaveFilename); err == nil {
		//		isBackedUp = true
		os.Rename(autoSaveFilename, backup)
	}

	var f *os.File
	f, err = os.Create(autoSaveFilename)
	if err != nil {
		log.Errorf("Can't create file to dump proxies: ", err)
	}
	defer f.Close()

	encoder := gob.NewEncoder(f)
	err = encoder.Encode(pc.proxies)
	if err != nil {
		log.Errorf("Can't dump proxies cache: %v", err)
	} else {
		log.Debug("Proxies cache dump")
	}

	return time.Now()
}

func (pc *CacheContext) checkProxy(proxy Proxy) {
	pc.grs.IncCheckProxy()
	defer pc.grs.DecCheckProxy()

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
		Timeout: time.Duration(60 * time.Second),
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
		log.Debugf("Proxy request failed %v: %v", addr, err)
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

	proxyList := LoadCache()

	var reader *bufio.Scanner = bufio.NewScanner(file)
	var result []Proxy = make([]Proxy, 0)

	newProxies := 0
	cachedProxies := 0

	for reader.Scan() {

		addr := reader.Text()

		i := -1
		if proxyList != nil {
			i = sort.Search(
				len(proxyList),
				func(i int) bool { return proxyList[i].Addr >= addr })
			if i == len(proxyList) || proxyList[i].Addr != addr {
				i = -1
			}
		}

		if i == -1 {
			result = append(
				result,
				Proxy{
					Addr:        reader.Text(),
					failCounter: 1, // by default proxy is BAD
				},
			)
			newProxies++
		} else {
			result = append(result, proxyList[i])
			cachedProxies++
		}

	}

	log.Debugf(
		"New proxies %d, cached proxies %d, total %d",
		newProxies, cachedProxies, len(result))

	return result
}

// Return sorted []Proxy. Use to quck search.
func LoadCache() ProxyList {
	_, err := os.Stat(autoSaveFilename)
	if os.IsNotExist(err) {
		return nil
	} else if err != nil {
		// TODO implement error handling
		panic(err)
	}

	f, err := os.Open(autoSaveFilename)
	if err != nil {
		// TODO implement error handling
		panic(err)
	}

	proxyHeap := make(ProxyHeap, 0)

	decoder := gob.NewDecoder(f)
	err = decoder.Decode(&proxyHeap)
	if err != nil {
		// TODO implement error handling
		panic(err)
	}

	proxyList := ProxyList(proxyHeap)
	sort.Sort(proxyList)
	return proxyList
}
