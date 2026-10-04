//go:build !windows

package launcher

import "fmt"

func executableVersion(path string) (string, error) {
	return "", fmt.Errorf("PE discovery is only available on Windows")
}
