package cli

import (
	"fmt"
	"os/exec"
	"runtime"
)

type BrowserOpener func(string) error

func defaultBrowserOpener(url string) error {
	name, args := browserCommand(url)
	if name == "" {
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	return exec.Command(name, args...).Run()
}

func browserCommand(url string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "open", []string{url}
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		return "xdg-open", []string{url}
	}
}
