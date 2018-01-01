package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/paivaspol/lowerboundproxy"
)

func main() {
	port := flag.Int("port", 8443, "The port this proxy should listen to")
	importantURLFile := flag.String("important-urls", "./important", "The filye containing the important URLs delimited by newline")
	verbose := flag.Bool("verbose", false, "Whether to verbosely log the proxy")
	flag.Parse()

	log.Printf(fmt.Sprintf("Starting proxy on %d\n", *port))

	resourceQueue := lowerboundproxy.NewResourceQueue()
	defer func() {
		resourceQueue.Cleanup()
	}()

	importantURLs, err := getImportantURLs(*importantURLFile)
	if err != nil {
		log.Fatalf("failed to get important URLs: %v", err)
	}

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
		priority := lowerboundproxy.Low
		if _, ok := importantURLs[r.Request.URL.String()]; ok {
			priority = lowerboundproxy.High
		}
		resourceQueue.QueueRequest(priority, r.Request.URL.String(), signalChan)

		// Block until we get a go from the queue.
		<-signalChan
		return r
	})
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), proxyHandler))
}

// getImportantURLs reads the URLs from the given file. The file is assumed to
// contain URLs separated by new line characters.
func getImportantURLs(importantURLFile string) (map[string]bool, err) {
	file, err := os.Open(importantURLFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	importantURLs = make(map[string]bool)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		importantURL[scanner.Text()] = true
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return importantURLs, nil
}
