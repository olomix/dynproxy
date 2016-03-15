package main

import (
	"bufio"
	"flag"
	"fmt"
	chttp "github.com/olomix/dynproxy/http"
	"github.com/olomix/dynproxy/log"
	"github.com/olomix/dynproxy/proxy_cache"
	"github.com/olomix/dynproxy/stats"
	"io"
	"net"
	"net/http"
	"os"
)

const PROXY_HEADER = "X-Dynproxy-Proxy"

var proxyFileName string
var listenAddress string

func init() {
	flag.StringVar(&proxyFileName, "in", "-", "file to read proxies from")
	flag.StringVar(
		&listenAddress,
		"listen", "0.0.0.0:3128", "address to listen on")
	flag.Parse()
}

func main() {
	flag.Parse()
	log.SetupLogs()

	var grs *stats.GoRoutineStats = stats.New()

	var addr *net.TCPAddr
	var err error
	var pCache proxy_cache.ProxyCache = proxy_cache.NewProxyCache(
		proxyFileName, grs)
	addr, err = net.ResolveTCPAddr("tcp", listenAddress)
	if err != nil {
		panic(fmt.Sprintf("can't resolve addr %v: %v", listenAddress, err))
	}

	chttp.ListenAndServe(grs)

	var server *net.TCPListener
	server, err = net.ListenTCP("tcp", addr)
	if err != nil {
		log.Error(err)
		os.Exit(1)
	}

	for {
		var conn *net.TCPConn
		conn, err = server.AcceptTCP()
		if err != nil {
			log.Error(err)
			panic(err)
		}
		go handleConnection(conn, pCache, grs)
	}

}

func handleConnection(
	clientConn *net.TCPConn,
	pCache proxy_cache.ProxyCache,
	grs *stats.GoRoutineStats,
) {

	grs.IncClientProxy()
	defer grs.DecClientProxy()
	requestIdx := grs.NewRequest(clientConn.RemoteAddr().String())
	defer grs.StopClientHandler(requestIdx)

	var (
		bufReader *bufio.Reader = bufio.NewReader(clientConn)
		req       *http.Request
		err       error
	)
	if req, err = http.ReadRequest(bufReader); err != nil {
		log.Errorf("Error on reading request: %v", err)
		return
	}
	log.Debugf("Got request to %v", req.URL)
	grs.SetUrl(requestIdx, req.URL.String())

	var proxy string
	if proxies, ok := req.Header[PROXY_HEADER]; ok && len(proxies) > 0 {
		proxy = proxies[0]
		req.Header.Del(PROXY_HEADER)
	} else {
		proxy, err = pCache.NextProxy()
		if err != nil {
			log.Errorf("Can't get next proxy: %v", err)
			clientConn.Close()
			return
		}
	}
	grs.SetProxy(requestIdx, proxy)
	var proxyAddr *net.TCPAddr
	proxyAddr, err = net.ResolveTCPAddr("tcp", proxy)
	if err != nil {
		log.Errorf("can't resolve addr %v: %v", proxy, err)
		clientConn.Close()
		return
	}
	var proxyConn *net.TCPConn
	log.Printf("Handle connection with %v", proxy)
	proxyConn, err = net.DialTCP("tcp", nil, proxyAddr)
	if err != nil {
		log.Errorf("can't deal to proxy: %v", err)
		clientConn.Close()
		return
	}

	err = req.Write(proxyConn)
	if err != nil {
		log.Errorf("Error on copying from client to proxy: %v", err)
		return
	}

	grs.StartProxyHandler(requestIdx)
	go copyProxyToClient(clientConn, proxyConn, req, proxy, grs, requestIdx)

	var l int64
	l, err = io.Copy(proxyConn, clientConn)
	if err != nil {
		log.Errorf("Error on extra copying from client to proxy: %v", err)
		return
	}
	log.Printf("Copied %d bytes from client to proxy", l)
}

func copyProxyToClient(
	clientConn, proxyConn *net.TCPConn,
	req *http.Request, proxy string,
	grs *stats.GoRoutineStats,
	requestIdx stats.RequestIdx,
) {
	grs.IncProxyClient()
	defer grs.DecProxyClient()
	defer grs.StopProxyHandler(requestIdx)

	var (
		err       error
		bufReader *bufio.Reader = bufio.NewReader(proxyConn)
		resp      *http.Response
	)
	resp, err = http.ReadResponse(bufReader, req)
	if err != nil {
		log.Errorf("Can't read response from proxy: %v", err)
		proxyConn.Close()
		clientConn.Close()
		return
	}
	resp.Header.Add(PROXY_HEADER, proxy)
	resp.Write(clientConn)
}
