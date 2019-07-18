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

type SiteMap struct {
	Map            map[string][]string
	mapLock        sync.Mutex
	baseURL        string
	visitedPaths   *set.Set
	concurrencyCap int
}

func New(URL string, concurrencyCap int) *SiteMap {
	return &SiteMap{
		Map:            make(map[string][]string),
		baseURL:        URL,
		visitedPaths:   set.New(),
		concurrencyCap: concurrencyCap,
	}
}

func (s *SiteMap) Build() []error {
	tasks := make(chan string, s.concurrencyCap)
	tasks <- s.baseURL

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

			go func(t string) {
				defer tasksWaitGroup.Done()
				if err := s.runTask(t, tasks, &tasksWaitGroup); err != nil {
					errsCh <- err
				}
			}(t)
		}
	}
}

func (s *SiteMap) runTask(
	t string,
	tasks chan<- string,
	wg *sync.WaitGroup,
) error {
	parsedURL, err := url.Parse(t)
	if err != nil {
		return fmt.Errorf("could not parse Home %q: %v", t, err)
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

		if !s.visitedPaths.Has(newParsedURL.Path) {
			s.visitedPaths.Add(newParsedURL.Path)

			wg.Add(1)
			tasks <- newURL
		}
	}

	return nil
}

func (s *SiteMap) Print() {
	for path, links := range s.Map {
		fmt.Printf("> %s\n", path)

		sort.Strings(links)
		for _, link := range links {
			fmt.Printf("  |- %s\n", link)
		}
	}
}

func (s *SiteMap) addLinkToMap(path, link string) {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	links, ok := s.Map[path]
	if !ok {
		s.Map[path] = []string{link}
		return
	}

	for _, l := range links {
		if l == link {
			return
		}
	}

	s.Map[path] = append(links, link)
}
