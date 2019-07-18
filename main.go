package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"runtime"
	"sitemap-builder/sitemap"
)

/**
 * @TODO:
 * * add maxDepth
 * * fetcher interface
 * * inject logger with desired LEVEL
 */
func main() {
	// Fetching command line arguments
	var startingURL string
	flag.StringVar(&startingURL, "url", "", "The starting Home")
	var maxProcs int
	flag.IntVar(&maxProcs, "maxprocs", 4, "Used to set GOMAXPROCS")
	var concurrencyCap int
	flag.IntVar(&concurrencyCap, "cap", 20, "Limits the amount of go routines we run at the same time")
	flag.Parse()

	// Validating command line arguments
	parsedURL, err := url.Parse(startingURL)
	if err != nil {
		log.Fatalf("Could not parse Home %q: %v", startingURL, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		log.Fatalf("Home must have an http or https scheme")
	}
	if concurrencyCap < 1 {
		log.Fatalf("Concurrency cap must be greater or equal to 1, got %d instead", concurrencyCap)
	}
	if maxProcs < 1 {
		log.Fatalf("maxprocs should be greater or equal to 1, got %d instead", maxProcs)
	}

	// Starting program
	runtime.GOMAXPROCS(maxProcs)

	sm := sitemap.New(parsedURL.String(), concurrencyCap)
	errs := sm.Build()
	sm.Print()

	if len(errs) > 0 {
		l := log.New(os.Stderr, "", log.LstdFlags)
		l.Println("Errors while building sitemap:")
		for _, err := range errs {
			l.Printf("* %v\n", err)
		}
	}
}
