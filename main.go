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
 * * fetcher interface
 * * inject logger with desired LEVEL
 */
func main() {
	// Fetching command line arguments
	var startingURL string
	flag.StringVar(&startingURL, "url", "", "The starting URL")
	var maxProcs int
	flag.IntVar(&maxProcs, "maxprocs", 4, "The maximum number of CPUs that can be executing simultaneously")
	var maxDepth int
	flag.IntVar(&maxDepth, "maxDepth", 3, "Maximum depth when following links")
	var concurrencyCap int
	flag.IntVar(&concurrencyCap, "cap", 20, "Limits the amount of go routines that run at the same time")
	flag.Parse()

	// Validating command line arguments
	parsedURL, err := url.Parse(startingURL)
	if err != nil {
		log.Fatalf("Could not parse URL %q: %v", startingURL, err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		log.Fatalf("URL must have an http or https scheme")
	}
	if concurrencyCap < 1 {
		log.Fatalf("Concurrency cap must be greater or equal to 1, got %d instead", concurrencyCap)
	}
	if maxDepth < 1 {
		log.Fatalf("Max depth must be greater or equal to 1, got %d instead", maxDepth)
	}
	if maxProcs < 1 {
		log.Fatalf("maxprocs should be greater or equal to 1, got %d instead", maxProcs)
	}

	// Starting program
	runtime.GOMAXPROCS(maxProcs)

	sm := sitemap.New(parsedURL.String(), maxDepth, concurrencyCap)
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
