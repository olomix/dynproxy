package main

import (
	"fmt"
	"net"
	"io"
	"time"
	"log"
	"flag"
	"os"
	"bufio"
)

var proxyList = make(chan string, 10)
var proxyFileName string

func init() {
	flag.StringVar(&proxyFileName, "in", "-", "file to read proxies from")
	flag.Parse()
}

type Proxy struct {
	Addr string
	lastCheck time.Time
	failCounter int
}

// Read proxies from input file. One address:port per line.
func readProxiesFromFile() []Proxy{
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
		result = append(result, Proxy{Addr: reader.Text()})
	}
	return result
}

func proxyEmitter() {
	var proxies []Proxy = readProxiesFromFile()
	for {
		for i := range proxies {
			proxyList <- proxies[i].Addr
		}
	}
}

func nextProxy() string {
	return <-proxyList
}

func main() {
	go proxyEmitter()
	var addr *net.TCPAddr
	var err error
	var unresolved_addr string = "0.0.0.0:3128";
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
		var conn *net.TCPConn;
		conn, err = server.AcceptTCP();
		if err != nil {
			panic(err)
		}
		go handleConnection(conn)
	}

}

func handleConnection(conn *net.TCPConn) {
	fmt.Print(conn)
	var proxyAddr *net.TCPAddr
	var err error
	var proxy string = nextProxy()
	proxyAddr, err = net.ResolveTCPAddr("tcp", proxy)
	if err != nil {
		panic(fmt.Sprintf("can't resolve addr %v: %v", proxy, err))
	}
	var proxyConn *net.TCPConn
	log.Printf("Handle connection with %v", proxy)
	proxyConn, err = net.DialTCP("tcp", nil, proxyAddr)
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
