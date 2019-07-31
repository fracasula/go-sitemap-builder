package sitemap

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sitemap-builder/fetcher"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type writer struct {
	content string
}

func (w *writer) Write(p []byte) (n int, err error) {
	w.content += string(p)
	return len(p), nil
}

func TestBuild(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(router))
	defer ts.Close()

	pipe := writer{}

	f := defaultFetcher()
	sm, errs := Build(ts.URL, f, 100, 4)
	require.Len(t, errs, 0)

	err := sm.Print(&pipe)
	require.Nil(t, err)

	require.Equal(t, `> /
  |- /
  |- /page-1
  |- /page-2
> /page-1
  |- /
  |- /page-1
  |- /page-1a
  |- /page-1b
> /page-1a
  |- /
  |- /page-1
  |- /page-1a
  |- /page-1b
> /page-1b
  |- /
  |- /page-1
  |- /page-1a
  |- /page-1b
> /page-2
  |- /
  |- /page-2
  |- /page-2a
> /page-2a
  |- /a-deep-page
`, pipe.content)
}

func TestBuildMaxDepth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(router))
	defer ts.Close()

	pipe := writer{}

	f := defaultFetcher()
	sm, errs := Build(ts.URL, f, 1, 4)
	require.Len(t, errs, 0)

	err := sm.Print(&pipe)
	require.Nil(t, err)

	require.Equal(t, `> /
  |- /
  |- /page-1
  |- /page-2
> /page-1
  |- /
  |- /page-1
  |- /page-1a
  |- /page-1b
> /page-2
  |- /
  |- /page-2
  |- /page-2a
`, pipe.content)
}

func router(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	switch r.RequestURI {
	case "":
		fallthrough
	case "/":
		_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
				<a href="/page-1">Page 1</a>
				<a href="/page-2">Page 2</a>
				<a href="http://external-site.com">External</a>
			</body></html>`)
	case "/page-1":
		_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
				<a href="/page-1">Page 1</a>
				<a href="/page-1a">Page 1A</a>
				<a href="/page-1b">Page 1B</a>
			</body></html>`)
	case "/page-1a":
		fallthrough
	case "/page-1b":
		_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
				<a href="/page-1">Page 1</a>
				<a href="/page-1a">Page 1A</a>
				<a href="/page-1b">Page 1B</a>
			</body></html>`)
	case "/page-2":
		_, _ = fmt.Fprint(w, `<html><body>
				<a href="/">Home</a>
				<a href="/page-2">Page 2</a>
				<a href="/page-2a">Page 2A</a>
			</body></html>`)
	case "/page-2a":
		_, _ = fmt.Fprint(w, `<html><body>
				<a href="/a-deep-page">Deep</a>
			</body></html>`)
	case "/a-deep-page":
		_, _ = fmt.Fprint(w, `<html><body>
			<p>Nothing to see here</p>
		</body></html>`)
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, `<html><body>Not Found</body></html>`)
		return
	}
}

func defaultFetcher() fetcher.HTTPFetcher {
	return fetcher.NewHTTPFetcher(10 * time.Second)
}
