package sitemap

import (
	"fmt"
	"net/http"
	"net/url"
	"sitemap-builder/parser"
	"strings"
	"sync"
)

type task struct {
	url   string
	depth int
}

func Build(URL string, maxDepth, concurrencyCap int) (SiteMap, []error) {
	tasks := make(chan task, concurrencyCap)
	tasks <- task{url: URL, depth: 1}

	tasksWaitGroup := sync.WaitGroup{}
	tasksWaitGroup.Add(len(tasks))
	go func() {
		tasksWaitGroup.Wait()
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
				return *siteMap, errsSlice
			}

			go func(t task) {
				if err := runTask(siteMap, t, tasks, &tasksWaitGroup, maxDepth); err != nil {
					errsCh <- err
				}
			}(t)
		}
	}
}

func runTask(
	siteMap *SiteMap,
	t task,
	tasks chan<- task,
	wg *sync.WaitGroup,
	maxDepth int,
) error {
	defer wg.Done()

	parsedURL, err := url.Parse(t.url)
	if err != nil {
		return fmt.Errorf("could not parse URL %q: %v", t.url, err)
	}

	res, err := http.Get(parsedURL.String())
	if err != nil {
		return fmt.Errorf("could not get %q: %v", parsedURL, err)
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		return fmt.Errorf(
			"invalid status code when getting %q, got %d, expected >=200 && <300",
			parsedURL, res.StatusCode,
		)
	}

	hrefs, err := parser.FindHrefs(res.Body)
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
			wg.Add(1)
			tasks <- task{url: newURL, depth: t.depth + 1}
		}

		// add path to sitemap
		siteMap.addLink(path, newPath)
	}

	return nil
}