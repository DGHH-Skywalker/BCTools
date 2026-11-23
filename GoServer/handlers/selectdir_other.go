//go:build !windows

package handlers

import "fmt"

func selectDirectory(title string) (string, error) {
	return "", fmt.Errorf("folder selection is only supported on Windows")
}

func selectKugouDatabase() (string, error) {
	return "", fmt.Errorf("Kugou database selection is only supported on Windows")
}
