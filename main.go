package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/olomix/dynproxy/log"
	"github.com/olomix/dynproxy/proxy_cache"
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
	var addr *net.TCPAddr
	var err error
	var pCache proxy_cache.ProxyCache = proxy_cache.NewProxyCache(proxyFileName)
	addr, err = net.ResolveTCPAddr("tcp", listenAddress)
	if err != nil {
		panic(fmt.Sprintf("can't resolve addr %v: %v", listenAddress, err))
	}
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
			panic(err)
		}
		go handleConnection(conn, pCache)
	}

}

func handleConnection(clientConn *net.TCPConn, pCache proxy_cache.ProxyCache) {
	var bufReader *bufio.Reader = bufio.NewReader(clientConn)
	var req *http.Request
	var err error
	if req, err = http.ReadRequest(bufReader); err != nil {
		log.Errorf("Error on reading request: %v", err)
		return
	}
	log.Debugf("Got request to %v", req.URL)

	var proxy string
	var proxies []string
	var ok bool
	if proxies, ok = req.Header[PROXY_HEADER]; ok && len(proxies) > 0 {
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

	go copyProxyToClient(clientConn, proxyConn, req, proxy)

	var l int64
	l, err = io.Copy(proxyConn, clientConn)
	if err != nil {
		log.Errorf("Error on extra copying from client to proxy: %v", err)
		return
	}
	log.Printf("Copied %d bytes from client to proxy", l)
}

func copyProxyToClient(
	clientConn, proxyConn *net.TCPConn, req *http.Request, proxy string,
) {
	var err error
	var bufReader *bufio.Reader = bufio.NewReader(proxyConn)
	var resp *http.Response
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
