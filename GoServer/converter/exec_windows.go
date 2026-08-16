//go:build windows

package converter

import (
	"context"
	"os/exec"

	"broadcast-tool/platform"
)

func createCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return platform.CommandContext(ctx, name, args...)
}
