package fetcher

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimpleHTTPFetcher_Fetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = fmt.Fprint(w, "Hello, client")
	}))
	defer ts.Close()

	f := NewHTTPFetcher()
	_, reader, err := f.Fetch(ts.URL, []string{"text/html"})
	require.Nil(t, err)

	body, err := ioutil.ReadAll(reader)
	require.Nil(t, err)
	require.Equal(t, "Hello, client", string(body))
}

func TestSimpleHTTPFetcher_Fetch_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusNotFound)
		_, _ = fmt.Fprint(w, "Hello, client")
	}))
	defer ts.Close()

	f := NewHTTPFetcher()
	_, _, err := f.Fetch(ts.URL, []string{"text/html"})
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "invalid status code")
}

func TestSimpleHTTPFetcher_Fetch_InvalidContentType(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, "Hello, client")
	}))
	defer ts.Close()

	f := NewHTTPFetcher()
	_, _, err := f.Fetch(ts.URL, []string{"text/html"})
	require.NotNil(t, err)
	require.Contains(t, err.Error(), "must be one of [text/html]")
}
