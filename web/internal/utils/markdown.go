package utils

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/renderer/html"
)

// RenderMarkdown converts markdown to sanitized HTML.
func RenderMarkdown(input string) string {
	var buf bytes.Buffer
	md := goldmark.New(
		goldmark.WithRendererOptions(html.WithHardWraps()),
	)
	if err := md.Convert([]byte(input), &buf); err != nil {
		return ""
	}
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("href", "title").OnElements("a")
	p.AllowAttrs("src", "alt").OnElements("img")
	return p.Sanitize(buf.String())
}
