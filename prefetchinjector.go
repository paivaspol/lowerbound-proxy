package lowerboundproxy

import (
	"compress/gzip"
	"fmt"
	"html/template"
	"log"
	"net/http"
)

const htmlTemplateFile = "static/prefetch_template.html"

// PrefetchInjector is a HTTP handler that injects prefetch URLs into the templated
// response and redirects to the destination page.
type PrefetchInjector struct {
	htmlTemplate *template.Template
	prefetches   map[string]bool
}

// NewPrefetchInjector creates a prefetch injector object.
func NewPrefetchInjector(prefetches map[string]bool) (*PrefetchInjector, error) {
	htmlTemplate, err := Asset(htmlTemplateFile)
	if err != nil {
		return nil, err
	}
	return &PrefetchInjector{
		htmlTemplate: template.Must(template.New("").Parse(string(htmlTemplate))),
		prefetches:   prefetches,
	}, nil
}

// ServeHTTP handles the HTTP request.
func (pi *PrefetchInjector) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	dstPage := r.URL.Query().Get("dstPage")
	log.Printf("DstPage: %v", dstPage)
	prefetchesSlice := []string{}
	for url, _ := range pi.prefetches {
		prefetchesSlice = append(prefetchesSlice, url)
	}
	// Generate the JS stub.
	templateData := struct {
		DstPage    template.JS
		Prefetches []string
	}{
		DstPage:    template.JS(dstPage),
		Prefetches: prefetchesSlice,
	}

	// (2) Return with the templated response.
	rw.Header().Set("Content-Encoding", "gzip")
	writer, err := gzip.NewWriterLevel(rw, gzip.BestCompression)
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	defer writer.Close()

	rw.Header().Set("Content-Type", "text/html")
	rw.WriteHeader(http.StatusOK)
	err = pi.htmlTemplate.Execute(writer, templateData)
	if err != nil {
		fmt.Printf("template.Execute: %v\n", err)
	}
}
