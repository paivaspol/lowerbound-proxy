package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/elazarl/goproxy"
	"github.com/paivaspol/lowerboundproxy"
)

func main() {
	port := flag.Int("port", 8443, "The port this proxy should listen to")
	prefetchURLFile := flag.String("prefetch-urls", "./prefetch_urls", "The file containing the prefetch URLs delimited by newline")
	verbose := flag.Bool("verbose", false, "Whether to verbosely log the proxy")
	passthrough := flag.Bool("passthrough", false, "Whether to run this proxy as a passthrough proxy")
	requestOrder := flag.String("request-order", "./request_order", "The file containing the order of the requests.")
	flag.Parse()

	log.Printf(fmt.Sprintf("Starting proxy on %d\n", *port))

	var prefetchURLs map[string]bool
	var err error
	var resourceQueue *lowerboundproxy.ResourceQueue
	if !*passthrough {
		resourceQueue, err = lowerboundproxy.NewResourceQueue(*requestOrder)
		if err != nil {
			log.Fatalf("failed to get important URLs: %v", err)
		}

		defer func() {
			resourceQueue.Cleanup()
		}()

		prefetchURLs, err = getPrefetchURLs(*prefetchURLFile)
		if err != nil {
			log.Fatalf("failed to get important URLs: %v", err)
		}
	}

	proxyHandler := goproxy.NewProxyHttpServer()
	proxyHandler.Verbose = *verbose
	proxyHandler.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxyHandler.OnRequest().DoFunc(func(r *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		log.Printf("[In Req] req: %v", r.URL.String())
		return r, nil
	})
	proxyHandler.OnResponse().DoFunc(func(r *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		// This is a passthrough proxy. Just return the response.
		if *passthrough {
			return r
		}
		log.Printf("[Response] req: %v respCode: %v", r.Request.URL, r.Status)
		signalChan := make(chan bool)
		priority := lowerboundproxy.Low

		// Prefetch URLs are less important than ones that are not prefetched.
		if _, ok := prefetchURLs[r.Request.URL.String()]; !ok {
			priority = lowerboundproxy.High
		}
		resourceQueue.QueueRequest(priority, r.Request.URL.String(), signalChan)

		// Block until we get a go from the queue.
		<-signalChan
		log.Printf("[Response] Completed req: %v", r.Request.URL)
		return r
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", *port), proxyHandler))
}

// getPrefetchURLs reads the URLs from the given file. The file is assumed to
// contain URLs separated by new line characters.
func getPrefetchURLs(prefetchURLFile string) (map[string]bool, error) {
	file, err := os.Open(prefetchURLFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	prefetchURLs := make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		prefetchURLs[scanner.Text()] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return prefetchURLs, nil
}
