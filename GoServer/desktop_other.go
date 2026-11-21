//go:build !windows

package main

import (
	"errors"
	"net/http"
)

func checkDesktop() error    { return errors.New("桌面模式支持 Windows x64") }
func showLaunchError(string) {}
func runDesktop(_ http.Handler, _ <-chan struct{}, _ <-chan string, _ string) error {
	return checkDesktop()
}
