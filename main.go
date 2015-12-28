package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"github.com/olomix/dynproxy/proxy_cache"
)

var proxyFileName string

func init() {
	flag.StringVar(&proxyFileName, "in", "-", "file to read proxies from")
	flag.Parse()
}

func main() {
	flag.Parse()
	var addr *net.TCPAddr
	var err error
	var unresolved_addr string = "0.0.0.0:3128"
	var pCache proxy_cache.ProxyCache = proxy_cache.NewProxyCache(proxyFileName)
	addr, err = net.ResolveTCPAddr("tcp", unresolved_addr)
	if err != nil {
		panic(fmt.Sprintf("can't resolve addr %v: %v", unresolved_addr, err))
	}
	var server *net.TCPListener
	server, err = net.ListenTCP("tcp", addr)
	if err != nil {
		panic(err)
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
		log.Printf("Can't get next proxy: %v", err)
		conn.Close()
		return
	}
	proxyAddr, err = net.ResolveTCPAddr("tcp", proxy)
	if err != nil {
		log.Printf("can't resolve addr %v: %v", proxy, err)
		conn.Close()
		return
	}
	var proxyConn *net.TCPConn
	log.Printf("Handle connection with %v", proxy)
	proxyConn, err = net.DialTCP("tcp", nil, proxyAddr)
	if err != nil {
		log.Printf("can't deal to proxy: %v", err)
		conn.Close()
		return
	}
	go copyClientToProxy(conn, proxyConn)
	go copyProxyToClient(conn, proxyConn)
}

func copyClientToProxy(clientConn, proxyConn *net.TCPConn) {
	var l int64
	var err error
	l, err = io.Copy(proxyConn, clientConn)
	if err != nil {
		log.Printf("Error on copying from client to proxy: %v", err)
	}
	log.Printf("Copied %d bytes from client to proxy", l)
}

func copyProxyToClient(clientConn, proxyConn *net.TCPConn) {
	var l int64
	var err error
	l, err = io.Copy(clientConn, proxyConn)
	if err != nil {
		log.Printf("Error on copying from proxy to cient: %v", err)
	}
	log.Printf("Copied %d bytes from proxy to client", l)
}
