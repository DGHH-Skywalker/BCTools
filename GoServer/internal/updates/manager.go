package updates

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"broadcast-tool/internal/plugins"

	"github.com/theupdateframework/go-tuf/v2/metadata"
	"golang.org/x/mod/semver"
)

const apiVersion = 1
const maxArtifact = 256 << 20
const maxBundle = 384 << 20

var itemID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,59}$`)
var digestPattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Item struct {
	ID              string `json:"id"`
	Kind            string `json:"kind"`
	Version         string `json:"version"`
	Target          string `json:"target"`
	InstallerTarget string `json:"installerTarget,omitempty"`
	Platform        string `json:"platform,omitempty"`
	Arch            string `json:"arch,omitempty"`
	APIVersion      int    `json:"apiVersion"`
	Storage         int    `json:"storageVersion,omitempty"`
	Description     string `json:"description,omitempty"`
}

type Catalog struct {
	Schema int    `json:"schema"`
	Items  []Item `json:"items"`
}

type Resource struct {
	Announcement string            `json:"announcement,omitempty"`
	Templates    map[string]string `json:"templates,omitempty"`
	Config       map[string]string `json:"config,omitempty"`
}

type activeVersion struct {
	Digest  string `json:"digest"`
	Version string `json:"version"`
}

type state struct {
	Source       string                   `json:"source,omitempty"`
	Web          activeVersion            `json:"web"`
	PreviousWeb  activeVersion            `json:"previousWeb"`
	WebPendingAt time.Time                `json:"webPendingAt,omitempty"`
	Resources    map[string]activeVersion `json:"resources"`
	Core         *CorePayload             `json:"core,omitempty"`
}

type Status struct {
	Configured bool   `json:"configured"`
	Available  []Item `json:"available"`
	Source     string `json:"source"`
	Web        string `json:"web"`
	CoreReady  bool   `json:"coreReady"`
}

type Manager struct {
	dir, version string
	source       SourceConfig
	sources      []source
	client       *http.Client
	plugins      *plugins.Registry
	mu           sync.RWMutex
	operation    sync.Mutex
	restarting   atomic.Bool
	state        state
	catalog      Catalog
}

func New(dataDir, version string, registry *plugins.Registry) (*Manager, error) {
	var config SourceConfig
	if err := json.Unmarshal(sourceConfiguration, &config); err != nil {
		return nil, err
	}
	sources, err := candidateSources(config)
	if err != nil && !errors.Is(err, ErrUnconfigured) {
		return nil, err
	}
	m := &Manager{
		dir: filepath.Join(dataDir, "updates"), version: version, source: config, sources: sources,
		plugins: registry, state: state{Resources: make(map[string]activeVersion)},
		client: &http.Client{Timeout: 95 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 || req.URL.Scheme != "https" || req.URL.User != nil {
				return errors.New("更新源重定向无效")
			}
			if u := req.URL.Hostname(); u == "localhost" || net.ParseIP(u) != nil || !strings.Contains(u, ".") {
				return errors.New("更新源地址无效")
			}
			return nil
		}},
	}
	b, err := os.ReadFile(filepath.Join(m.dir, "state.json"))
	if err == nil {
		if err := json.Unmarshal(b, &m.state); err != nil {
			return nil, fmt.Errorf("parse updates state: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if m.state.Resources == nil {
		m.state.Resources = make(map[string]activeVersion)
	}
	return m, nil
}

func (m *Manager) Status() Status {
	revision, _ := m.Web()
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Status{
		Configured: len(m.sources) > 0, Available: append([]Item{}, m.catalog.Items...), Source: m.state.Source,
		Web: revision, CoreReady: m.state.Core != nil && semver.Compare("v"+m.state.Core.Version, "v"+m.version) > 0,
	}
}

func (m *Manager) Check(ctx context.Context) error {
	m.operation.Lock()
	defer m.operation.Unlock()
	return m.refresh(ctx)
}

func (m *Manager) refresh(ctx context.Context) error {
	if len(m.sources) == 0 {
		return ErrUnconfigured
	}
	var errs []error
	for _, src := range m.preferredSources() {
		updater, err := m.updater(ctx, src, false)
		if err == nil {
			err = updater.Refresh()
		}
		var info *metadata.TargetFiles
		if err == nil {
			info, err = updater.GetTargetInfo("catalog.json")
		}
		if err == nil && info.Length <= 128<<10 {
			_, b, downloadErr := updater.DownloadTarget(info, "", "")
			var result Catalog
			if downloadErr == nil {
				downloadErr = json.Unmarshal(b, &result)
			}
			if downloadErr == nil && result.Schema == 1 {
				m.mu.Lock()
				m.catalog = Catalog{Schema: 1}
				for _, item := range result.Items {
					if m.compatible(item) {
						m.catalog.Items = append(m.catalog.Items, item)
					}
				}
				next := m.state
				next.Source = src.name
				err = m.saveLocked(next)
				m.mu.Unlock()
				return err
			}
			err = downloadErr
		}
		if err == nil {
			err = errors.New("invalid update catalog")
		}
		errs = append(errs, fmt.Errorf("%s: %w", src.name, err))
		if ctx.Err() != nil {
			break
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) compatible(item Item) bool {
	if !itemID.MatchString(item.ID) || strings.Count(item.Version, ".") < 2 || !semver.IsValid("v"+item.Version) ||
		item.APIVersion != apiVersion || !fs.ValidPath(item.Target) || item.Target == "." || strings.ContainsAny(item.Target, "\\:%?#") {
		return false
	}
	if (item.Platform != "" && item.Platform != runtime.GOOS) || (item.Arch != "" && item.Arch != runtime.GOARCH) {
		return false
	}
	switch item.Kind {
	case "core":
		return runtime.GOOS == "windows" && item.Storage == 2 && semver.Compare("v"+item.Version, "v"+m.version) > 0
	case "resource", "web", "plugin":
		return true
	default:
		return false
	}
}

func (m *Manager) Apply(ctx context.Context, id string) error {
	m.operation.Lock()
	defer m.operation.Unlock()
	m.mu.RLock()
	i := slices.IndexFunc(m.catalog.Items, func(item Item) bool { return item.ID == id })
	if i < 0 {
		m.mu.RUnlock()
		return errors.New("找不到已验证的更新")
	}
	item := m.catalog.Items[i]
	srcName := m.state.Source
	m.mu.RUnlock()
	if !m.compatible(item) {
		return errors.New("更新版本不兼容")
	}
	idx := slices.IndexFunc(m.sources, func(s source) bool { return s.name == srcName })
	if idx < 0 {
		return ErrUnconfigured
	}
	u, err := m.updater(ctx, m.sources[idx], false)
	if err != nil {
		return err
	}
	installer := item.Kind == "core" && m.IsInstalled()
	target := item.Target
	if installer {
		target = item.InstallerTarget
		if target == "" || !fs.ValidPath(target) || strings.ContainsAny(target, `\:%?#`) {
			return errors.New("已安装版本需要发布对应的 Inno Setup 更新包")
		}
	}
	info, err := u.GetTargetInfo(target)
	if err != nil {
		return err
	}
	if info.Length <= 0 || info.Length > maxArtifact || len(info.Hashes["sha256"]) != sha256.Size {
		return errors.New("更新文件大小或摘要无效")
	}
	if err := os.MkdirAll(filepath.Join(m.dir, "files"), 0700); err != nil {
		return err
	}
	digest := hex.EncodeToString(info.Hashes["sha256"])
	file := filepath.Join(m.dir, "files", digest)
	if err := m.download(ctx, target, info, file); err != nil {
		return err
	}
	active := activeVersion{Digest: digest, Version: item.Version}
	switch item.Kind {
	case "plugin":
		if info.Length > plugins.MaxPackageSize {
			return errors.New("插件包过大")
		}
		b, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		if m.plugins == nil {
			return errors.New("插件服务不可用")
		}
		_, err = m.plugins.Install(ctx, b)
		return err
	case "resource":
		if _, err := readResource(file); err != nil {
			return err
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if v, ok := m.state.Resources[id]; ok && semver.Compare("v"+item.Version, "v"+v.Version) < 0 {
			return errors.New("资源版本低于已安装版本")
		}
		next := m.state
		next.Resources = make(map[string]activeVersion)
		for k, v := range m.state.Resources {
			next.Resources[k] = v
		}
		next.Resources[id] = active
		return m.saveLocked(next)
	case "web":
		if err := m.prepareWeb(file, digest); err != nil {
			return err
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if m.state.Web.Digest == digest {
			return nil
		}
		if m.state.Web.Version != "" && semver.Compare("v"+item.Version, "v"+m.state.Web.Version) < 0 {
			return errors.New("网页版本低于已安装版本")
		}
		next := m.state
		next.Web, next.PreviousWeb = active, next.Web
		next.WebPendingAt = time.Now().UTC()
		return m.saveLocked(next)
	case "core":
		m.mu.Lock()
		defer m.mu.Unlock()
		next := m.state
		next.Core = &CorePayload{
			Target: target, Digest: digest, Length: info.Length, Version: item.Version, Installer: installer,
		}
		return m.saveLocked(next)
	}
	return errors.New("未知的更新类型")
}

func (m *Manager) saveLocked(next state) error {
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(m.dir, 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(m.dir, ".state-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(b)
	if err == nil {
		err = file.Sync()
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return err
	}
	if err := os.Rename(file.Name(), filepath.Join(m.dir, "state.json")); err != nil {
		return err
	}
	m.state = next
	return nil
}

func verifyFile(name string, length int64, expected []byte) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	digest := sha256.New()
	n, err := io.Copy(digest, io.LimitReader(f, length+1))
	if err != nil {
		return err
	}
	if n != length || !bytes.Equal(digest.Sum(nil), expected) {
		return errors.New("更新文件未通过完整性检查")
	}
	return nil
}

func (m *Manager) download(ctx context.Context, name string, info *metadata.TargetFiles, target string) error {
	expected := info.Hashes["sha256"]
	if verifyFile(target, info.Length, expected) == nil {
		return nil
	}
	strictName := path.Join(path.Dir(name), hex.EncodeToString(expected)+"."+path.Base(name))
	var errs []error
	for _, src := range m.preferredSources() {
		for _, candidate := range []string{strictName, name} {
			if err := m.downloadFile(ctx, src.target+candidate, info.Length, expected, target); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", src.name, err))
			} else {
				return nil
			}
		}
		if ctx.Err() != nil {
			break
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) downloadFile(ctx context.Context, raw string, length int64, expected []byte, target string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	res, err := m.client.Do(request)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 || res.ContentLength >= 0 && res.ContentLength != length {
		return fmt.Errorf("无效的下载响应 HTTP %d", res.StatusCode)
	}
	temp, err := os.CreateTemp(filepath.Dir(target), ".download-*")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	digest := sha256.New()
	n, err := io.Copy(io.MultiWriter(temp, digest), io.LimitReader(res.Body, length+1))
	if err == nil && (n != length || !bytes.Equal(digest.Sum(nil), expected)) {
		err = errors.New("更新文件未通过摘要校验")
	}
	if err == nil {
		err = temp.Sync()
	}
	if err := errors.Join(err, temp.Close()); err != nil {
		return err
	}
	return os.Rename(temp.Name(), target)
}

func readResource(file string) (Resource, error) {
	var result Resource
	content, err := os.Open(file)
	if err != nil {
		return result, err
	}
	defer content.Close()
	b, err := io.ReadAll(io.LimitReader(content, (1<<20)+1))
	if err != nil || len(b) > 1<<20 {
		return result, errors.New("无法读取更新资源")
	}
	return result, json.Unmarshal(b, &result)
}

func (m *Manager) Resources() map[string]Resource {
	m.mu.RLock()
	defer m.mu.RUnlock()
	resources := make(map[string]Resource)
	for id, ver := range m.state.Resources {
		if digestPattern.MatchString(ver.Digest) {
			if content, err := readResource(filepath.Join(m.dir, "files", ver.Digest)); err == nil {
				resources[id] = content
			}
		}
	}
	return resources
}

func (m *Manager) prepareWeb(file, digest string) error {
	name := filepath.Join(m.dir, "web", digest)
	if _, err := os.Stat(name); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	temp, err := os.MkdirTemp(filepath.Dir(name), ".web-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	z, err := zip.OpenReader(file)
	if err != nil {
		return err
	}
	defer z.Close()
	var size uint64
	for _, f := range z.File {
		rel := strings.TrimSuffix(f.Name, "/")
		if !fs.ValidPath(rel) || !filepath.IsLocal(rel) || strings.ContainsAny(rel, "\\:") ||
			(!strings.HasPrefix(rel, "main/") && !strings.HasPrefix(rel, "um-react/") && rel != "main" && rel != "um-react") {
			return errors.New("更新包路径无效")
		}
		if f.Mode().IsDir() {
			continue
		}
		if !f.Mode().IsRegular() || f.UncompressedSize64 > maxBundle-size {
			return errors.New("网页更新包文件类型或大小无效")
		}
		size += f.UncompressedSize64
		dst := filepath.Join(temp, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			return err
		}
		if err := unpackFile(f, dst); err != nil {
			return err
		}
	}
	for _, filename := range []string{"main/index.html", "um-react/index.html"} {
		b, err := os.ReadFile(filepath.Join(temp, filename))
		if err != nil || len(b) == 0 {
			return fmt.Errorf("更新包缺少 %s", filename)
		}
	}
	return os.Rename(temp, name)
}

func unpackFile(z *zip.File, dst string) error {
	rc, err := z.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(rc, int64(z.UncompressedSize64)+1))
	if err == nil && n != int64(z.UncompressedSize64) {
		err = errors.New("invalid uncompressed size")
	}
	return errors.Join(err, f.Close())
}

func (m *Manager) Web() (string, fs.FS) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.state.WebPendingAt.IsZero() && time.Since(m.state.WebPendingAt) > 2*time.Minute {
		_ = m.rollbackLocked()
	}
	for _, version := range []activeVersion{m.state.Web, m.state.PreviousWeb} {
		if !digestPattern.MatchString(version.Digest) {
			continue
		}
		dir := os.DirFS(filepath.Join(m.dir, "web", version.Digest))
		if _, err := fs.Stat(dir, "main/index.html"); err == nil {
			return version.Digest, dir
		}
	}
	return "", nil
}

func (m *Manager) WebVersion(digest string) fs.FS {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !digestPattern.MatchString(digest) || (m.state.Web.Digest != digest && m.state.PreviousWeb.Digest != digest) {
		return nil
	}
	return os.DirFS(filepath.Join(m.dir, "web", digest))
}

func (m *Manager) ConfirmWeb(digest string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if digest == "" || digest != m.state.Web.Digest || m.state.WebPendingAt.IsZero() {
		return nil
	}
	next := m.state
	next.WebPendingAt = time.Time{}
	return m.saveLocked(next)
}

func (m *Manager) RollbackWeb() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rollbackLocked()
}

func (m *Manager) rollbackLocked() error {
	next := m.state
	next.Web, next.PreviousWeb = next.PreviousWeb, activeVersion{}
	next.WebPendingAt = time.Time{}
	return m.saveLocked(next)
}
