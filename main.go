package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"sitemap-builder/fetcher"
	"sitemap-builder/sitemap"
)

func main() {
	// Fetching command line arguments
	var startingURL string
	flag.StringVar(&startingURL, "url", "", "The starting URL")
	var maxProcs int
	flag.IntVar(&maxProcs, "maxprocs", 4, "The maximum number of CPUs that can be executing simultaneously")
	var maxDepth int
	flag.IntVar(&maxDepth, "maxDepth", 3, "Maximum depth when following links")
	var concurrencyCap int
	flag.IntVar(&concurrencyCap, "cap", 40, "Limits the amount of go routines that run at the same time")
	flag.Parse()

	// Validating command line arguments
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

	f := fetcher.NewHTTPFetcher()
	sm, errs := sitemap.Build(startingURL, f, maxDepth, concurrencyCap)
	if err := sm.Print(os.Stdout); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		if _, printErr := fmt.Fprintln(os.Stderr, "Errors while building sitemap:"); printErr != nil {
			os.Exit(1)
		}
		for _, err := range errs {
			if _, printErr := fmt.Fprintf(os.Stderr, "* %v\n", err); printErr != nil {
				os.Exit(1)
			}
		}
	}
}
