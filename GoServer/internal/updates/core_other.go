//go:build !windows

package updates

import "errors"

func SignalReady(string)               {}
func RunCoreHelper(string, bool) error { return errors.New("应用更新支持 Windows") }
