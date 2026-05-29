package assets

import (
	"io/fs"
	"strings"
	"testing"
)

func TestStaticAssetsEmbedded(t *testing.T) {
	data, err := fs.ReadFile(FS, "static/style.css")
	if err != nil {
		t.Fatalf("read embedded stylesheet: %v", err)
	}
	if !strings.Contains(string(data), "font-family") {
		t.Fatal("embedded stylesheet does not look like CSS")
	}
}
