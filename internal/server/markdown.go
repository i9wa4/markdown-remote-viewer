package server

import (
	"bytes"
	"html/template"
	"regexp"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var (
	markdownRenderer = goldmark.New(
		goldmark.WithExtensions(extension.GFM),
	)
	markdownSanitizer = newMarkdownSanitizer()
	markdownTemplate  = template.Must(template.New("markdown").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <link rel="stylesheet" href="/assets/style.css">
</head>
<body>
  <main>
    <article class="markdown-body">
{{.Body}}
    </article>
  </main>
</body>
</html>
`))
)

func newMarkdownSanitizer() *bluemonday.Policy {
	policy := bluemonday.UGCPolicy()
	policy.AllowAttrs("class").Matching(regexp.MustCompile(`^language-[A-Za-z0-9_-]+$`)).OnElements("code")
	policy.AllowElements("input")
	policy.AllowAttrs("type").Matching(regexp.MustCompile(`^checkbox$`)).OnElements("input")
	policy.AllowAttrs("checked", "disabled").OnElements("input")
	return policy
}

func renderMarkdownPage(title string, source []byte) ([]byte, error) {
	var rendered bytes.Buffer
	if err := markdownRenderer.Convert(source, &rendered); err != nil {
		return nil, err
	}

	var page bytes.Buffer
	err := markdownTemplate.Execute(&page, struct {
		Title string
		Body  template.HTML
	}{
		Title: title,
		Body:  template.HTML(markdownSanitizer.SanitizeBytes(rendered.Bytes())),
	})
	if err != nil {
		return nil, err
	}
	return page.Bytes(), nil
}
