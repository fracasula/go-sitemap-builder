package parser

import (
	"fmt"
	"io"

	"github.com/PuerkitoBio/goquery"
)

func FindHrefs(r io.Reader) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("could not parse document: %v", err)
	}

	selector := doc.Find("a")
	if selector.Length() == 0 {
		return nil, nil
	}

	var urls []string
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if ok {
			urls = append(urls, href)
		}
	})

	return urls, nil
}
