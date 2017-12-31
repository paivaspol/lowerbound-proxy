package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
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

	proxy := goproxy.NewProxyHttpServer()
	// proxy.Verbose = true
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		log.Printf("req: %v", r)
		return r, goproxy.NewResponse(r, goproxy.ContentTypeText, http.StatusOK, "Oops")
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), proxy))
}
