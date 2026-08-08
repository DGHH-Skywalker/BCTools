//go:build windows

package launcher

import (
	"fmt"
	"log"
	"strings"

	"broadcast-tool/runner"
)

// KillExistingBackend kills the process occupying the given localhost port.
func KillExistingBackend(port int) error {
	out, err := runner.Command("cmd", "/c", fmt.Sprintf("netstat -ano | findstr :%d", port)).Output()
	if err != nil {
		return fmt.Errorf("netstat failed: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		localAddr := fields[1]
		pid := fields[4]
		if strings.HasSuffix(localAddr, fmt.Sprintf(":%d", port)) {
			if err := runner.Command("taskkill", "/F", "/PID", pid).Run(); err != nil {
				log.Printf("Failed to kill PID %s on port %d: %v", pid, port, err)
			} else {
				log.Printf("Killed PID %s occupying port %d", pid, port)
			}
		}
	}
	return nil
}
