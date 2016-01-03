package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"github.com/olomix/dynproxy/proxy_cache"
	"github.com/golang/glog"
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
		glog.Errorf("Can't get next proxy: %v", err)
		conn.Close()
		return
	}
	proxyAddr, err = net.ResolveTCPAddr("tcp", proxy)
	if err != nil {
		glog.Errorf("can't resolve addr %v: %v", proxy, err)
		conn.Close()
		return
	}
	var proxyConn *net.TCPConn
	glog.Infof("Handle connection with %v", proxy)
	proxyConn, err = net.DialTCP("tcp", nil, proxyAddr)
	if err != nil {
		glog.Errorf("can't deal to proxy: %v", err)
		conn.Close()
		return
	}
	var clientCh chan bool = make(chan bool)
	var proxyCh chan bool = make(chan bool)
	go copyClientToProxy(conn, proxyConn, clientCh)
	go copyProxyToClient(conn, proxyConn, proxyCh)
	for i := 0; i < 2; i++ {
		select {
		case <- clientCh:
			glog.Info("Client connection done")
		case <- proxyCh:
			glog.Info("Proxy connection done")
		}
	}
}

func copyClientToProxy(clientConn, proxyConn *net.TCPConn, ch chan bool) {
	var l int64
	var err error
	l, err = io.Copy(proxyConn, clientConn)
	if err != nil {
		glog.Errorf("Error on copying from client to proxy: %v", err)
	}
	glog.Infof("Copied %d bytes from client to proxy", l)
	ch <- false
}

func copyProxyToClient(clientConn, proxyConn *net.TCPConn, ch chan bool) {
	var l int64
	var err error
	l, err = io.Copy(clientConn, proxyConn)
	if err != nil {
		glog.Errorf("Error on copying from proxy to cient: %v", err)
	}
	glog.Infof("Copied %d bytes from proxy to client", l)
	if err = clientConn.Close(); err != nil {
		glog.Errorf("Can't close client connection: %v", err)
	}
	ch <- false
}
