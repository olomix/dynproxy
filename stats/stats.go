package stats

import "sync/atomic"

type GoRoutineStats struct {
	clientProxyNum uint64
	proxyClientNum uint64
	checkProxyNum  uint64
}

func (grs *GoRoutineStats) IncClientProxy() {
	atomic.AddUint64(&(grs.clientProxyNum), 1)
}

func (grs *GoRoutineStats) DecClientProxy() {
	atomic.AddUint64(&(grs.clientProxyNum), ^uint64(0))
}

func (grs *GoRoutineStats) GetClientProxy() uint64 {
	return atomic.LoadUint64(&(grs.clientProxyNum))
}

func (grs *GoRoutineStats) IncProxyClient() {
	atomic.AddUint64(&(grs.proxyClientNum), 1)
}

func (grs *GoRoutineStats) DecProxyClient() {
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
