package parser

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindHrefs(t *testing.T) {
	html := `<html>
		<body>
			<!-- <a title="this shouldn't count">link in comment</a> -->
			<p>A paragraph with a <a title="test" href="www.example.com">link</A></P>
	
	<a title="some cool website"
		href="example.com/hello-world">Hello</a>
	<p>paragraph</p><a href="another website">World</a></body></html>
	`

	links, err := FindHrefs(bytes.NewBufferString(html))
	require.Nil(t, err)
	require.Equal(t, []string{
		"www.example.com",
		"example.com/hello-world",
		"another website",
	}, links)
}
