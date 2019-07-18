package sitemap

import (
	"fmt"
	"net/http"
	"net/url"
	"sitemap-builder/parser"
	"sitemap-builder/set"
	"sort"
	"strings"
	"sync"
)

type task struct {
	url   string
	depth int
}

type SiteMap struct {
	siteMap        map[string][]string
	mapLock        sync.Mutex
	baseURL        string
	visitedPaths   *set.Set
	maxDepth       int
	concurrencyCap int
}

func New(URL string, maxDepth, concurrencyCap int) *SiteMap {
	return &SiteMap{
		siteMap:        make(map[string][]string),
		baseURL:        URL,
		visitedPaths:   set.New(),
		maxDepth:       maxDepth,
		concurrencyCap: concurrencyCap,
	}
}

func (s *SiteMap) Build() []error {
	tasks := make(chan task, s.concurrencyCap)
	tasks <- task{url: s.baseURL, depth: 1}

	tasksWaitGroup := sync.WaitGroup{}
	tasksWaitGroup.Add(len(tasks))
	go func() {
		tasksWaitGroup.Wait()
		close(tasks)
	}()

	var errsSlice []error
	errsCh := make(chan error)

	for {
		select {
		case err := <-errsCh:
			errsSlice = append(errsSlice, err)
		case t, open := <-tasks:
			if !open {
				return errsSlice
			}

			go func(t task) {
				defer tasksWaitGroup.Done()
				if err := s.runTask(t, tasks, &tasksWaitGroup); err != nil {
					errsCh <- err
				}
			}(t)
		}
	}
}

func (s *SiteMap) runTask(
	t task,
	tasks chan<- task,
	wg *sync.WaitGroup,
) error {
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
		if string(newURL[0]) == "/" {
			newURL = parsedURL.Scheme + "://" + parsedURL.Host + newURL
		}

		newParsedURL, err := url.Parse(newURL)
		if err != nil {
			return fmt.Errorf("could not parse new URL %q: %v", newURL, err)
		}

		if parsedURL.Path == newParsedURL.Path {
			continue
		}

		if newParsedURL.Host != parsedURL.Host {
			continue
		}

		s.addLinkToMap(parsedURL.Path, newParsedURL.Path)

		if t.depth <= s.maxDepth && !s.visitedPaths.Has(newParsedURL.Path) {
			s.visitedPaths.Add(newParsedURL.Path)

			wg.Add(1)
			tasks <- task{url: newURL, depth: t.depth + 1}
		}
	}

	return nil
}

func (s *SiteMap) Print() {
	paths := make([]string, len(s.siteMap))

	i := 0
	for path := range s.siteMap {
		paths[i] = path
		i++
	}

	sort.Strings(paths)

	for _, path := range paths {
		fmt.Printf("> %s\n", path)

		sort.Strings(s.siteMap[path])
		for _, link := range s.siteMap[path] {
			fmt.Printf("  |- %s\n", link)
		}
	}
}

func (s *SiteMap) addLinkToMap(path, link string) {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	links, ok := s.siteMap[path]
	if !ok {
		s.siteMap[path] = []string{link}
		return
	}

	for _, l := range links {
		if l == link {
			return
		}
	}

	s.siteMap[path] = append(links, link)
}
