// devlauncher 启动 bctools.exe 的 -dev 模式，用于开发测试。
package main

import (
	"broadcast-tool/platform"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	dir := filepath.Dir(os.Args[0])
	if exe, err := os.Executable(); err == nil {
		dir = filepath.Dir(exe)
	}
	target := filepath.Join(dir, "bctools.exe")
	cmd := platform.Command(target, "-dev")
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to start bctools.exe:", err)
		os.Exit(1)
	}
}
