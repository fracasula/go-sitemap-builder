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
	semaphore := make(chan struct{}, concurrencyCap)

	siteMap := newSitemap()
	var errorsSlice []error
	errorsChannel := make(chan error)

	errGroup := sync.WaitGroup{}
	waitGroup := sync.WaitGroup{}
	waitGroup.Add(1)

	go func() {
		for {
			select {
			case err := <-errorsChannel:
				errorsSlice = append(errorsSlice, err)
				errGroup.Done()
			case t := <-tasks:
				// the semaphore is to avoid having too many goroutines running at the same time
				semaphore <- struct{}{}

				go func(t task) {
					defer waitGroup.Done()

					newTasks, errs := runTask(t, siteMap, f, maxDepth)
					if len(errs) > 0 {
						errGroup.Add(len(errs))

						// avoid locking by queueing the errors
						go func(errs []error) {
							for _, err := range errs {
								errorsChannel <- err
							}
						}(errs)
					}
					if len(newTasks) > 0 {
						waitGroup.Add(len(newTasks))

						// we can't use the tasks channel as a semaphore because a single task could
						// generate an arbitrary number of new tasks so we can't lock here just by
						// pushing into tasks and a bigger buffer is a bad idea. let's create another
						// go routine to avoid locking and let the semaphore handle the concurrency bound
						go func(newTasks []task) {
							for _, newTask := range newTasks {
								tasks <- newTask
							}
						}(newTasks)
					}

					<-semaphore
				}(t)
			}
		}
	}()

	errGroup.Wait()
	waitGroup.Wait()

	return siteMap, errorsSlice
}

func runTask(
	t task,
	siteMap *SiteMap,
	f fetcher.HTTPFetcher,
	maxDepth int,
) ([]task, []error) {
	parsedURL, reader, err := f.Fetch(t.url, []string{"text/html"})
	if err != nil {
		// @TODO we could make Fetch return custom errors so that we could handle things
		// like unreachable network (exit from the program), 504 timeouts (try again) and so on.
		// Retries could be fairly simple, as easy as:
		// * add a maxRetries to the "runTask" function
		// * add the number of retries to the "task" struct
		// * check whether the retries are less than maxRetries, if yes return the task again in the slice
		return nil, []error{fmt.Errorf("could not fetch %q: %v", t.url, err)}
	}

	hrefs, err := parser.FindHrefs(reader)
	if err != nil {
		return nil, []error{fmt.Errorf("could not parse %q: %v", parsedURL, err)}
	}

	var tasks []task
	var errors []error
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
			errors = append(errors, fmt.Errorf("could not parse new URL %q: %v", newURL, err))
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

		// add path to sitemap
		siteMap.addLink(path, newPath)

		// keep creating tasks if max depth hasn't been reached and page hasn't been visited yet
		if t.depth <= maxDepth && !siteMap.has(newPath) {
			tasks = append(tasks, task{url: newURL, depth: t.depth + 1})
		}
	}

	return tasks, errors
}
