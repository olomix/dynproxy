package stats

import (
	"fmt"
	"github.com/olomix/dynproxy/log"
	"sync"
	"sync/atomic"
	"time"
)

type RequestIdx struct {
	idx int
	wg  *sync.WaitGroup
}

func (i RequestIdx) String() string {
	return fmt.Sprintf("Request ID %v", i.idx)
}

type Request struct {
	URL, Client, Proxy                        string
	ClientHandlerRunning, ProxyHandlerRunning bool
	Start                                     time.Time
}

type GoRoutineStats struct {
	clientProxyNum uint64
	proxyClientNum uint64
	checkProxyNum  uint64
	lock           sync.Mutex
	requests       []Request
	requestsMask   []bool // If false, then appropriate element in requests is free
}

func New() *GoRoutineStats {
	return new(GoRoutineStats)
}

func (grs *GoRoutineStats) incClientProxy() {
	atomic.AddUint64(&(grs.clientProxyNum), 1)
}

func (grs *GoRoutineStats) decClientProxy() {
	atomic.AddUint64(&(grs.clientProxyNum), ^uint64(0))
}

func (grs *GoRoutineStats) GetClientProxy() uint64 {
	return atomic.LoadUint64(&(grs.clientProxyNum))
}

func (grs *GoRoutineStats) incProxyClient() {
	atomic.AddUint64(&(grs.proxyClientNum), 1)
}

func (grs *GoRoutineStats) decProxyClient() {
	atomic.AddUint64(&(grs.proxyClientNum), ^uint64(0))
}

func (grs *GoRoutineStats) GetProxyClient() uint64 {
	return atomic.LoadUint64(&(grs.proxyClientNum))
}

func (grs *GoRoutineStats) IncCheckProxy() {
	atomic.AddUint64(&(grs.checkProxyNum), 1)
}

func (grs *GoRoutineStats) DecCheckProxy() {
	atomic.AddUint64(&(grs.checkProxyNum), ^uint64(0))
}

func (grs *GoRoutineStats) GetCheckProxy() uint64 {
	return atomic.LoadUint64(&(grs.checkProxyNum))
}

func (grs *GoRoutineStats) allocateRequest() int {
	grs.lock.Lock()
	defer grs.lock.Unlock()

	for idx, v := range grs.requestsMask {
		if !v {
			grs.requestsMask[idx] = true
			return idx
		}
	}

	// No free elements. Reallocate larger array.
	var idx int = len(grs.requests)
	var newSize int = len(grs.requests) * 2
	if newSize == 0 {
		newSize = 100
	}
	newRequests := make([]Request, newSize)
	copy(newRequests, grs.requests)
	grs.requests = newRequests
	newRequestsMask := make([]bool, newSize)
	copy(newRequestsMask, grs.requestsMask)
	grs.requestsMask = newRequestsMask
	grs.requestsMask[idx] = true
	return idx
}

func (grs *GoRoutineStats) freeRequest(idx int) {
	grs.lock.Lock()
	grs.requestsMask[idx] = false
	grs.lock.Unlock()
}

func (grs *GoRoutineStats) NewRequest(client string) RequestIdx {
	idx := grs.allocateRequest()
	log.Debugf("New request %v", idx)
	grs.lock.Lock()
	grs.requests[idx].Client = client
	grs.requests[idx].ClientHandlerRunning = true
	grs.requests[idx].ProxyHandlerRunning = false
	grs.requests[idx].Start = time.Now()

	var ri RequestIdx = RequestIdx{idx: idx, wg: new(sync.WaitGroup)}
	ri.wg.Add(1)
	go waitForClose(grs, ri)

	grs.lock.Unlock()

	grs.incClientProxy()

	return ri
}

func (grs *GoRoutineStats) SetUrl(idx RequestIdx, url string) {
	grs.lock.Lock()
	grs.requests[idx.idx].URL = url
	grs.lock.Unlock()
}

func (grs *GoRoutineStats) SetProxy(idx RequestIdx, proxy string) {
	grs.lock.Lock()
	grs.requests[idx.idx].Proxy = proxy
	grs.lock.Unlock()
}

func (grs *GoRoutineStats) StartProxyHandler(idx RequestIdx) {
	grs.lock.Lock()
	grs.requests[idx.idx].ProxyHandlerRunning = true
	idx.wg.Add(1)
	grs.lock.Unlock()

	grs.incProxyClient()
}

func (grs *GoRoutineStats) StopProxyHandler(idx RequestIdx) {
	grs.lock.Lock()
	grs.requests[idx.idx].ProxyHandlerRunning = false
	idx.wg.Done()
	grs.lock.Unlock()

	grs.decProxyClient()
}

func (grs *GoRoutineStats) StopClientHandler(idx RequestIdx) {
	grs.lock.Lock()
	grs.requests[idx.idx].ClientHandlerRunning = false
	idx.wg.Done()
	grs.lock.Unlock()

	grs.decClientProxy()
}

type ActiveRequest struct {
	Idx                                       int
	URL, Client, Proxy                        string
	ClientHandlerRunning, ProxyHandlerRunning bool
	ActiveSeconds                             int
}

func (grs *GoRoutineStats) ActiveRequests() []ActiveRequest {
	grs.lock.Lock()
	l := 0
	for _, i := range grs.requestsMask {
		if i {
			l += 1
		}
	}
	var reqs []ActiveRequest = make([]ActiveRequest, 0, l)
	for idx, i := range grs.requestsMask {
		if !i {
			continue
		}
		reqs = append(reqs, ActiveRequest{
			Idx:                  idx,
			URL:                  grs.requests[idx].URL,
			Client:               grs.requests[idx].Client,
			Proxy:                grs.requests[idx].Proxy,
			ClientHandlerRunning: grs.requests[idx].ClientHandlerRunning,
			ProxyHandlerRunning:  grs.requests[idx].ProxyHandlerRunning,
			ActiveSeconds:        int(time.Since(grs.requests[idx].Start).Seconds()),
		})
	}
	grs.lock.Unlock()
	return reqs
}

func waitForClose(grs *GoRoutineStats, ri RequestIdx) {
	ri.wg.Wait()
	grs.freeRequest(ri.idx)
	log.Debugf("Free request %v", ri.idx)
}
