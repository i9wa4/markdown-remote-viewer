package assets

import (
	"io/fs"
	"strings"
	"testing"
)

func TestStaticAssetsEmbedded(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "static/style.css", want: "font-family"},
		{path: "static/markdown-viewer.js", want: "mermaid.min.js"},
		{path: "static/vendor/mermaid.min.js", want: "mermaid"},
		{path: "static/vendor/mermaid.LICENSE", want: "The MIT License"},
	} {
		data, err := fs.ReadFile(FS, tc.path)
		if err != nil {
			t.Fatalf("read embedded asset %s: %v", tc.path, err)
		}
		if !strings.Contains(string(data), tc.want) {
			t.Fatalf("embedded asset %s missing %q", tc.path, tc.want)
		}
	}
}
