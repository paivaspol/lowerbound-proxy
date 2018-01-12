package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/paivaspol/lowerboundproxy"
)

func main() {
	port := flag.Int("port", 8443, "The port this proxy should listen to")
	prefetchURLFile := flag.String("prefetch-urls", "./prefetch_urls", "The file containing the URLs to prefetch delimited by newline")
	flag.Parse()

	prefetchURLs, err := getPrefetchURLs(*prefetchURLFile)
	if err != nil {
		log.Fatalf("failed to get prefetch URLs: %v", err)
	}

	// Initialize the prefetch injector HTTP handle for generating page with prefetches.
	pi, err := lowerboundproxy.NewPrefetchInjector(prefetchURLs)
	if err != nil {
		log.Fatalf("failed to get important URLs: %v", err)
	}
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", *port), pi))
}

// getPrefetchURLs reads the URLs from the given file. The file is assumed to
// contain URLs separated by new line characters.
func getPrefetchURLs(prefetchURLFile string) ([]string, error) {
	file, err := os.Open(prefetchURLFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	prefetchURLs := []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		prefetchURLs = append(prefetchURLs, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return prefetchURLs, nil
}
