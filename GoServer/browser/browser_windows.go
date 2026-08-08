//go:build windows

package browser

import "broadcast-tool/runner"

// Open opens the given URL in the system default browser.
func Open(url string) error {
	return runner.Command("cmd", "/c", "start", "", url).Start()
}
