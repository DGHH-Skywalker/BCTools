package updates

import (
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	"broadcast-tool/platform"

	"golang.org/x/mod/semver"
)

type CorePayload struct {
	Target    string `json:"target"`
	Digest    string `json:"digest"`
	Length    int64  `json:"length"`
	Version   string `json:"version"`
	Installer bool   `json:"installer"`
}

func (m *Manager) corePayload(ctx context.Context) (CorePayload, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.sources) == 0 || m.state.Core == nil {
		return CorePayload{}, "", errors.New("没有可安装的更新")
	}
	p := *m.state.Core
	if !digestPattern.MatchString(p.Digest) || p.Length <= 0 || p.Length > maxArtifact ||
		!semver.IsValid("v"+p.Version) || semver.Compare("v"+p.Version, "v"+m.version) <= 0 {
		return CorePayload{}, "", errors.New("无效的安装版本")
	}
	u, err := m.updater(ctx, m.sources[0], true)
	if err != nil {
		return p, "", err
	}
	info, err := u.GetTargetInfo(p.Target)
	if err != nil || info.Length != p.Length || hex.EncodeToString(info.Hashes["sha256"]) != p.Digest {
		return p, "", errors.New("安装包缺少匹配的签名元数据")
	}
	name := filepath.Join(m.dir, "files", p.Digest)
	digest, _ := hex.DecodeString(p.Digest)
	return p, name, verifyFile(name, p.Length, digest)
}

func (m *Manager) installDir() string {
	return filepath.Dir(filepath.Dir(m.dir))
}

func (m *Manager) IsInstalled() bool {
	matches, err := filepath.Glob(filepath.Join(m.installDir(), "unins*.exe"))
	return err == nil && len(matches) != 0
}

func (m *Manager) RestartIntoUpdate(ctx context.Context, desktop bool) error {
	if !m.restarting.CompareAndSwap(false, true) {
		return errors.New("应用已经在准备重启")
	}
	started := false
	defer func() {
		if !started {
			m.restarting.Store(false)
		}
	}()
	m.operation.Lock()
	defer m.operation.Unlock()
	if _, _, err := m.corePayload(ctx); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	helperDir := filepath.Join(m.dir, "helper")
	if err := os.MkdirAll(helperDir, 0700); err != nil {
		return err
	}
	helper, err := os.CreateTemp(helperDir, "bctools-update-*.exe")
	if err != nil {
		return err
	}
	defer func() {
		if !started {
			_ = os.Remove(helper.Name())
		}
	}()
	f, err := os.Open(exe)
	if err != nil {
		helper.Close()
		return err
	}
	_, err = io.Copy(helper, f)
	f.Close()
	if err := errors.Join(err, helper.Close()); err != nil {
		return err
	}
	args := []string{"--apply-staged-update"}
	if desktop {
		args = append(args, "desktop")
	}
	cmd := platform.Command(helper.Name(), args...)
	cmd.Dir = m.installDir()
	if err := cmd.Start(); err != nil {
		return err
	}
	started = true
	_ = cmd.Process.Release()
	return nil
}

func waitForOwnership(acquire func() (io.Closer, error)) (io.Closer, error) {
	deadline := time.Now().Add(35 * time.Second)
	for {
		owner, err := acquire()
		if err != nil || owner != nil {
			return owner, err
		}
		if time.Now().After(deadline) {
			return nil, errors.New("等待旧版退出超时，尚未替换文件")
		}
		time.Sleep(200 * time.Millisecond)
	}
}
