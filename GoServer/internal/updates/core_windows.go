//go:build windows

package updates

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"broadcast-tool/launcher"
	"broadcast-tool/platform"

	"github.com/google/uuid"
	"github.com/minio/selfupdate"
	"golang.org/x/sys/windows"
)

const readyEnv = "BCTOOLS_UPDATE_READY"

func SignalReady(dataDir string) {
	id := os.Getenv(readyEnv)
	parsed, err := uuid.Parse(id)
	if err == nil && parsed.String() == id {
		_ = os.WriteFile(filepath.Join(dataDir, "updates", "helper", id+".ready"), nil, 0600)
	}
}

func RunCoreHelper(version string, desktop bool) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	helperDir := filepath.Dir(exe)
	updateDir := filepath.Dir(helperDir)
	dataDir := filepath.Dir(updateDir)
	if !strings.EqualFold(filepath.Base(helperDir), "helper") ||
		!strings.EqualFold(filepath.Base(updateDir), "updates") ||
		!strings.EqualFold(filepath.Base(dataDir), "BctoolData") {
		return errors.New("无效的更新进程目录")
	}
	m, err := New(dataDir, version, nil)
	if err != nil {
		return err
	}
	owner, err := waitForOwnership(func() (io.Closer, error) {
		return launcher.Claim(dataDir, "update", make(chan string, 1))
	})
	if err != nil {
		return err
	}
	defer func() {
		if owner != nil {
			_ = owner.Close()
		}
	}()
	p, file, err := m.corePayload(context.Background())
	if err != nil {
		return err
	}
	target := filepath.Join(m.installDir(), "bctools.exe")
	backup := filepath.Join(helperDir, "previous-"+uuid.NewString()+".exe")
	args := []string{}
	if desktop {
		args = append(args, "desktop")
	}
	releaseOwnership := func() {
		_ = owner.Close()
		owner = nil
	}
	startOld := func() error {
		old := platform.Command(target, args...)
		old.Dir = m.installDir()
		if err := old.Start(); err != nil {
			return err
		}
		return old.Process.Release()
	}
	digest, _ := hex.DecodeString(p.Digest)
	if p.Installer {
		if err := copyExecutable(target, backup); err != nil {
			return err
		}
		setup := filepath.Join(helperDir, "BCTools-Setup-"+uuid.NewString()+".exe")
		defer os.Remove(setup)
		if err := copyExecutable(file, setup); err != nil {
			return err
		}
		if err := verifyFile(setup, p.Length, digest); err != nil {
			return err
		}
		if err := platform.Command(setup, "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/DIR="+m.installDir()).Run(); err != nil {
			installErr := fmt.Errorf("安装程序未完成: %w", err)
			if restoreErr := replaceBackup(backup, target); restoreErr != nil {
				return errors.Join(installErr, fmt.Errorf("恢复旧版失败: %w", restoreErr))
			}
			releaseOwnership()
			return errors.Join(installErr, startOld())
		}
	} else {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		err = selfupdate.Apply(f, selfupdate.Options{TargetPath: target, Checksum: digest, OldSavePath: backup})
		f.Close()
		if err != nil {
			return fmt.Errorf("替换程序失败: %w", err)
		}
	}
	releaseOwnership()
	id := uuid.NewString()
	ready := filepath.Join(helperDir, id+".ready")
	defer os.Remove(ready)
	cmd := platform.Command(target, args...)
	cmd.Env = append(os.Environ(), readyEnv+"="+id)
	cmd.Dir = m.installDir()
	if err := cmd.Start(); err == nil {
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		timer := time.NewTimer(25 * time.Second)
		defer timer.Stop()
		tick := time.NewTicker(200 * time.Millisecond)
		defer tick.Stop()
		healthy := false
		pending := true
		for pending {
			select {
			case <-tick.C:
				if _, err := os.Stat(ready); err == nil {
					healthy, pending = true, false
				}
			case <-done:
				pending = false
			case <-timer.C:
				_ = launcher.SendControl(dataDir, "shutdown")
				select {
				case <-done:
				case <-time.After(20 * time.Second):
					return errors.New("新版未就绪，无法关闭进程，旧版保留在更新目录")
				}
				pending = false
			}
		}
		if healthy {
			_ = os.Remove(backup)
			return nil
		}
	}
	if err := replaceBackup(backup, target); err != nil {
		return fmt.Errorf("新版启动失败，恢复旧版失败: %w", err)
	}
	if err := startOld(); err != nil {
		return err
	}
	return errors.New("新版启动失败，已恢复旧版")
}

func copyExecutable(source, target string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	return errors.Join(err, out.Close())
}

func replaceBackup(backup, target string) error {
	from, err := windows.UTF16PtrFromString(backup)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(from, to, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
