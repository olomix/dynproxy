package proxy_cache

import (
	"testing"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

func TestCheckWithProxy(t *testing.T) {
	badProxy := "120.195.201.189:80"
	if !checkWithProxy(badProxy) {
		t.Fail()
	}
}

func Test2(t *testing.T) {
	var req *http.Request
	var err error
	req, err = http.NewRequest("GET", "http://lomaka.org.ua/t.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	req.Header.Add("Cache-Control", "no-cache")
	req.Header.Add("Proxy-Connection", "Keep-Alive")

	var addr string = "54.193.103.13:3128"
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: func(req *http.Request) (*url.URL, error) {
				return url.Parse(fmt.Sprintf("http://%s", addr))
			},
		},
		Timeout: time.Duration(5 * time.Second),
	}

	var resp *http.Response
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	fmt.Println("OK")
}
