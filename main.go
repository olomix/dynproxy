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

func handleConnection(conn *net.TCPConn, pCache proxy_cache.ProxyCache) {
	var proxyAddr *net.TCPAddr
	var err error
	var proxy string
	proxy, err = pCache.NextProxy()
	if err != nil {
		log.Errorf("Can't get next proxy: %v", err)
		conn.Close()
		return
	}
	proxyAddr, err = net.ResolveTCPAddr("tcp", proxy)
	if err != nil {
		log.Errorf("can't resolve addr %v: %v", proxy, err)
		conn.Close()
		return
	}
	var proxyConn *net.TCPConn
	log.Printf("Handle connection with %v", proxy)
	proxyConn, err = net.DialTCP("tcp", nil, proxyAddr)
	if err != nil {
		log.Errorf("can't deal to proxy: %v", err)
		conn.Close()
		return
	}
	var clientCh chan bool = make(chan bool)
	var proxyCh chan bool = make(chan bool)
	go copyClientToProxy(conn, proxyConn, clientCh)
	go copyProxyToClient(conn, proxyConn, proxyCh)
	for i := 0; i < 2; i++ {
		select {
		case <-clientCh:
			log.Print("Client connection done")
		case <-proxyCh:
			log.Print("Proxy connection done")
		}
	}
}

func copyClientToProxy(clientConn, proxyConn *net.TCPConn, ch chan bool) {
	var bufReader *bufio.Reader = bufio.NewReader(clientConn)
	var req *http.Request
	var err error
	if req, err = http.ReadRequest(bufReader); err != nil {
		log.Errorf("Error on reading request: %v", err)
		ch <- false
		return
	}
	log.Debugf("Got request to %v", req.URL)

	err = req.Write(proxyConn)
	if err != nil {
		log.Errorf("Error on copying from client to proxy: %v", err)
		ch <- false
		return
	}

	var l int64
	l, err = io.Copy(proxyConn, clientConn)
	if err != nil {
		log.Errorf("Error on extra copying from client to proxy: %v", err)
		ch <- false
		return
	}
	log.Printf("Copied %d bytes from client to proxy", l)
	ch <- false
}

func copyProxyToClient(clientConn, proxyConn *net.TCPConn, ch chan bool) {
	var l int64
	var err error
	l, err = io.Copy(clientConn, proxyConn)
	if err != nil {
		log.Errorf("Error on copying from proxy to cient: %v", err)
	}
	log.Printf("Copied %d bytes from proxy to client", l)
	if err = clientConn.Close(); err != nil {
		log.Errorf("Can't close client connection: %v", err)
	}
	ch <- false
}
