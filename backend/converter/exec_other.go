//go:build !windows

package converter

import (
	"context"
	"os/exec"
)

func createCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
