package fetcher

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HTTPFetcher interface {
	// Fetch returns the rawurl as parsed, the body response and eventually an error
	Fetch(rawurl string, allowedContentTypes []string) (*url.URL, io.Reader, error)
}

// simpleHTTPFetcher only supports simple GETs and it doesn't support proxies
// for proxies support implement your own fetcher or change this one
type simpleHTTPFetcher struct {
	client http.Client
}

func NewHTTPFetcher(timeout time.Duration) HTTPFetcher {
	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &simpleHTTPFetcher{
		client: http.Client{
			Transport: &transport,
		},
	}
}

func (f *simpleHTTPFetcher) Fetch(rawurl string, allowedContentTypes []string) (*url.URL, io.Reader, error) {
	parsedURL, err := url.Parse(rawurl)
	if err != nil {
		return nil, nil, fmt.Errorf("could not parse URL %q: %v", rawurl, err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, nil, fmt.Errorf("URL must have an http or https scheme")
	}

	res, err := f.client.Get(parsedURL.String())
	if err != nil {
		return nil, nil, fmt.Errorf("could not get %q: %v", parsedURL, err)
	}

	if len(allowedContentTypes) > 0 {
		contentType := strings.ToLower(res.Header.Get("Content-Type")) // Header.Get is case insensitive

		found := false
		for _, ct := range allowedContentTypes {
			if strings.Contains(contentType, ct) {
				found = true
				break
			}
		}

		if !found {
			return nil, nil, fmt.Errorf(
				"%q Content-Type %q must be one of %v", parsedURL, contentType, allowedContentTypes,
			)
		}
	}

	if !(res.StatusCode >= 200 && res.StatusCode < 300) {
		return nil, nil, fmt.Errorf(
			"invalid status code when getting %q, got %d, expected >=200 && <300",
			parsedURL, res.StatusCode,
		)
	}

	return parsedURL, res.Body, nil
}
