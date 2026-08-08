//go:build windows

package converter

import (
	"context"
	"os/exec"

	"broadcast-tool/runner"
)

func createCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return runner.CommandContext(ctx, name, args...)
}
