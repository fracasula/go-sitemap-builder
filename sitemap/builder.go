package sitemap

import (
	"fmt"
	"net/url"
	"sitemap-builder/fetcher"
	"sitemap-builder/parser"
	"strings"
	"sync"
)

type task struct {
	url   string
	depth int
}

func Build(URL string, f fetcher.HTTPFetcher, maxDepth, concurrencyCap int) (*SiteMap, []error) {
	tasks := make(chan task, concurrencyCap)
	tasks <- task{url: URL, depth: 1}

	waitGroup := sync.WaitGroup{}
	waitGroup.Add(len(tasks))
	go func() {
		waitGroup.Wait()
		close(tasks)
	}()

	var errsSlice []error
	errsCh := make(chan error)
	siteMap := newSitemap()

	for {
		select {
		case err := <-errsCh:
			errsSlice = append(errsSlice, err)
		case t, open := <-tasks:
			if !open {
				return siteMap, errsSlice
			}

			go func(t task) {
				if err := runTask(t, tasks, siteMap, f, &waitGroup, maxDepth); err != nil {
					errsCh <- err
				}
			}(t)
		}
	}
}

func runTask(
	t task,
	tasks chan<- task,
	siteMap *SiteMap,
	f fetcher.HTTPFetcher,
	waitGroup *sync.WaitGroup,
	maxDepth int,
) error {
	defer waitGroup.Done()

	parsedURL, reader, err := f.Fetch(t.url, []string{"text/html"})
	if err != nil {
		// @TODO we could make Fetch return custom errors so that we could handle things
		// like unreachable network (exit from the program), 504 timeouts (try again) and so on.
		// Retries could be fairly simple, as easy as:
		// * add a maxRetries to the "runTask" function
		// * add the number of retries to the "task" struct
		// * check whether the retries are less than maxRetries, if yes push the task again into the tasks chan
		return fmt.Errorf("could not fetch %q: %v", t.url, err)
	}

	hrefs, err := parser.FindHrefs(reader)
	if err != nil {
		return fmt.Errorf("could not parse %q: %v", parsedURL, err)
	}

	for _, href := range hrefs {
		if len(href) == 0 {
			continue
		}

		newURL := strings.Trim(href, " ")
		if len(newURL) > 1 && string(newURL[len(newURL)-1]) == "/" { // remove ending slash
			newURL = newURL[:len(newURL)-1]
		}
		if string(newURL[0]) == "/" { // it's a relative URL, make it absolute
			newURL = parsedURL.Scheme + "://" + parsedURL.Host + newURL
		}

		newParsedURL, err := url.Parse(newURL)
		if err != nil {
			return fmt.Errorf("could not parse new URL %q: %v", newURL, err)
		}

		if parsedURL.Path == newParsedURL.Path { // page is linking itself, skip
			continue
		}
		if newParsedURL.Host != parsedURL.Host { // different website or sub-domain, skip
			continue
		}

		// if there's no path force it to "/" to avoid duplicates
		path := parsedURL.Path
		if path == "" {
			path = "/"
		}
		newPath := newParsedURL.Path
		if newPath == "" {
			newPath = "/"
		}

		// keep creating tasks if max depth hasn't been reached and page hasn't been visited yet
		if t.depth <= maxDepth && !siteMap.has(newPath) {
			waitGroup.Add(1)
			tasks <- task{url: newURL, depth: t.depth + 1}
		}

		// add path to sitemap
		siteMap.addLink(path, newPath)
	}

	return nil
}
