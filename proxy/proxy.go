package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/paivaspol/lowerboundproxy"
)

func orPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	port := flag.Int("port", 8443, "The port this proxy should listen to")
	verbose := flag.Bool("verbose", false, "Whether to verbosely log the proxy")
	flag.Parse()

	log.Printf(fmt.Sprintf("Starting proxy on %d\n", *port))

	resourceQueue := lowerboundproxy.NewResourceQueue()
	defer func() {
		resourceQueue.Cleanup()
	}()

	proxyHandler := goproxy.NewProxyHttpServer()
	proxyHandler.Verbose = *verbose
	proxyHandler.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxyHandler.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		log.Printf("[In Req] req: %v", r)
		return r, nil
	})
	proxyHandler.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		log.Printf("[In Response] req: %v", r.Request.URL)
		signalChan := make(chan bool)
		resourceQueue.QueueRequest(lowerboundproxy.High, r.Request.URL.String(), signalChan)
		// Block until we get a go from the queue.
		<-signalChan
		return r
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), proxyHandler))
}
