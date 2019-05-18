package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
	grafanaString     = "grafana.example.com"
	defaultRemoteHost = "localhost:30080"
)

var (
	port       string
	service    string
	targetHost string
)

func init() {
	flag.StringVar(&service, "svc", grafanaString, "target service")
	flag.StringVar(&port, "port", "8080", "proxy port")
	flag.StringVar(&targetHost, "host", defaultRemoteHost, "targeted host-port")
	flag.Parse()
}

func main() {
	fmt.Printf("reverse-proxy running on: %s\n", getListenAddress())
	proxy := newSingleHostReverseProxy(&url.URL{
		Scheme: "http",
		Host:   targetHost,
	})
	if err := http.ListenAndServe(getListenAddress(), proxy); err != nil {
		panic(err)
	}
}

func newSingleHostReverseProxy(target *url.URL) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = service
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
		req.Header.Set("Host", service)
	}
	proxy.Director = director
	return proxy
}

// getListenAddress get the port to listen on
func getListenAddress() string {
	return ":" + port
}
