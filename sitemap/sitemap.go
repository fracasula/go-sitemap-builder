package sitemap

import (
	"fmt"
	"io"
	"sort"
	"sync"
)

type SiteMap struct {
	siteMap map[string][]string
	mapLock sync.Mutex
}

func newSitemap() *SiteMap {
	return &SiteMap{
		siteMap: make(map[string][]string),
	}
}

func (s *SiteMap) has(path string) bool {
	s.mapLock.Lock()
	defer s.mapLock.Unlock()

	_, ok := s.siteMap[path]
	return ok
}

func (s *SiteMap) addLink(path, link string) {
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

// Print sorts the sitemap and prints it to a writer which works for the current use-case but it might be worth
// refactoring if you end up calling this multiple times to make sure you sort only once (unless there have been
// changes to the data in which case you'll need to sort again anyway)
func (s *SiteMap) Print(w io.Writer) error {
	paths := make([]string, len(s.siteMap))

	i := 0
	for path := range s.siteMap {
		paths[i] = path
		i++
	}

	sort.Strings(paths)

	for _, path := range paths {
		if _, err := fmt.Fprintf(w, "> %s\n", path); err != nil {
			return err
		}

		sort.Strings(s.siteMap[path])
		for _, link := range s.siteMap[path] {
			if _, err := fmt.Fprintf(w, "  |- %s\n", link); err != nil {
				return err
			}
		}
	}

	return nil
}
