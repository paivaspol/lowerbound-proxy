package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 8443, "The port this proxy should listen to")
	certFile := flag.String("cert_file", "./cert.pem", "The certificate this proxy should use")
	keyFile := flag.String("key_file", "./key.pem", "The key this proxy should use")
	flag.Parse()

	log.Printf(fmt.Sprintf("Starting proxy on %d\n", *port))

	http.HandleFunc("/", handleHTTP)
	log.Fatal(http.ListenAndServeTLS(fmt.Sprintf(":%d", *port), *certFile, *keyFile, nil))
}

// handleHTTP handles all requests that goes through this proxy.
//
// In this setup, the proxy can respond to a request in two ways:
// 	(1) Serve the request from a recorded file
// 	(2) Serve passthrough the request to the wide area network.
func handleHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[PROXY] Handling request: %v\n", r.URL.String())
	w.WriteHeader(200)
	w.Write([]byte("hello!\n"))
}
