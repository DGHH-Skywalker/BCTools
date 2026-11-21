package plugins

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/extism/go-sdk"
	"github.com/tetratelabs/wazero"
	"golang.org/x/mod/semver"
)

const MaxPackageSize = 16 << 20

var identifier = regexp.MustCompile(`^[a-z][a-z0-9-]{0,59}$`)
var route = regexp.MustCompile(`^/(?:[a-z][a-z0-9-]*)(?:/[a-z][a-z0-9-]*)*$`)

type Link struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type Page struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type Command struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Export string `json:"export"`
}

type Layout struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      Link   `json:"action"`
	Navigation  []Link `json:"navigation"`
}

type Manifest struct {
	ID          string    `json:"id"`
	Version     string    `json:"version"`
	APIVersion  int       `json:"apiVersion"`
	Name        string    `json:"name"`
	Permissions []string  `json:"permissions"`
	Menu        []Link    `json:"menu,omitempty"`
	Pages       []Page    `json:"pages,omitempty"`
	Commands    []Command `json:"commands,omitempty"`
	Layout      *Layout   `json:"layout,omitempty"`
}

type ActivePlugin struct {
	Manifest Manifest `json:"manifest"`
	Enabled  bool     `json:"enabled"`
}

type Snapshot struct {
	Plugins        []ActivePlugin `json:"plugins"`
	LayoutProvider string         `json:"layoutProvider"`
	Layout         *Layout        `json:"layout,omitempty"`
}

type state struct {
	Versions map[string]string `json:"versions"`
	Enabled  map[string]bool   `json:"enabled"`
	Layout   string            `json:"layout"`
}

type installed struct {
	manifest Manifest
	wasm     []byte
}

type Registry struct {
	mu      sync.RWMutex
	dir     string
	state   state
	active  map[string]installed
	summary func() []byte
	onError func(error)
}

func New(dir string, summary func() []byte, onError func(error)) (*Registry, error) {
	r := &Registry{dir: dir, summary: summary, onError: onError, active: make(map[string]installed)}
	r.state = state{Versions: make(map[string]string), Enabled: make(map[string]bool)}
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if os.IsNotExist(err) {
		return r, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &r.state); err != nil || r.state.Versions == nil || r.state.Enabled == nil {
		return nil, errors.New("invalid plugin state")
	}
	for id, version := range r.state.Versions {
		if !identifier.MatchString(id) || !validVersion(version) {
			return nil, errors.New("invalid installed plugin identifier")
		}
		b, err := os.ReadFile(r.packagePath(id, version))
		var candidate installed
		if err == nil {
			candidate, err = unpack(b)
		}
		if err != nil || candidate.manifest.ID != id || candidate.manifest.Version != version {
			if onError != nil {
				onError(fmt.Errorf("cannot load plugin %s: %v", id, err))
			}
			continue
		}
		r.active[id] = candidate
	}
	return r, nil
}

func validVersion(s string) bool {
	return !strings.ContainsAny(s, "/\\:") && strings.Count(s, ".") >= 2 && semver.IsValid("v"+s)
}

func unpack(b []byte) (installed, error) {
	if len(b) > MaxPackageSize {
		return installed{}, errors.New("插件包过大")
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return installed{}, err
	}
	var p installed
	seen := map[string]bool{}
	for _, f := range z.File {
		if (f.Name != "manifest.json" && f.Name != "plugin.wasm") || !f.Mode().IsRegular() || seen[f.Name] {
			return installed{}, errors.New("无效的插件包内容")
		}
		seen[f.Name] = true
		limit := int64(MaxPackageSize)
		if f.Name == "manifest.json" {
			limit = 64 << 10
		}
		rc, err := f.Open()
		if err != nil {
			return installed{}, err
		}
		data, readErr := io.ReadAll(io.LimitReader(rc, limit+1))
		rc.Close()
		if readErr != nil || int64(len(data)) > limit {
			return installed{}, errors.New("插件包内容无法读取或过大")
		}
		if f.Name == "plugin.wasm" {
			p.wasm = data
		} else {
			decoder := json.NewDecoder(bytes.NewReader(data))
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(&p.manifest); err != nil {
				return installed{}, err
			}
			if err := decoder.Decode(new(any)); err != io.EOF {
				return installed{}, errors.New("插件清单包含额外内容")
			}
		}
	}
	return p, validate(p)
}

func validate(p installed) error {
	m := p.manifest
	if !identifier.MatchString(m.ID) || !validVersion(m.Version) || m.APIVersion != 1 || m.Name == "" || len(m.Name) > 200 {
		return errors.New("插件 ID、版本或 API 版本无效")
	}
	permissions := map[string]bool{"ui.contribute": true, "ui.layout": true, "commands.run": true, "playlist.read": true}
	for _, permission := range m.Permissions {
		if !permissions[permission] {
			return errors.New("未知的插件权限")
		}
	}
	if (len(m.Pages) > 0 || len(m.Menu) > 0) && !slices.Contains(m.Permissions, "ui.contribute") ||
		m.Layout != nil && !slices.Contains(m.Permissions, "ui.layout") ||
		len(m.Commands) > 0 && (!slices.Contains(m.Permissions, "commands.run") || len(p.wasm) == 0) {
		return errors.New("插件没有声明所需权限")
	}

	links := slices.Clone(m.Menu)
	if m.Layout != nil {
		links = append(links, m.Layout.Action)
		links = append(links, m.Layout.Navigation...)
	}
	for _, item := range links {
		if !route.MatchString(item.Path) || item.Label == "" {
			return errors.New("插件路由必须是应用内路径")
		}
	}
	ids := make(map[string]bool)
	for _, page := range m.Pages {
		if !identifier.MatchString(page.ID) || ids[page.ID] || page.Title == "" {
			return errors.New("无效或重复的插件页面")
		}
		ids[page.ID] = true
	}
	ids = make(map[string]bool)
	for _, cmd := range m.Commands {
		if !identifier.MatchString(cmd.ID) || ids[cmd.ID] || cmd.Export == "" || cmd.Label == "" {
			return errors.New("无效或重复的插件命令")
		}
		ids[cmd.ID] = true
	}
	return nil
}

func (r *Registry) instantiate(ctx context.Context, p installed) (*extism.Plugin, error) {
	manifest := extism.Manifest{
		Wasm:    []extism.Wasm{extism.WasmData{Data: p.wasm}},
		Memory:  &extism.ManifestMemory{MaxPages: 256, MaxVarBytes: 1 << 20},
		Timeout: 2000,
	}
	config := extism.PluginConfig{
		RuntimeConfig: wazero.NewRuntimeConfig().WithMemoryLimitPages(256).WithCloseOnContextDone(true),
	}
	var functions []extism.HostFunction
	if slices.Contains(p.manifest.Permissions, "playlist.read") && r.summary != nil {
		functions = append(functions, extism.NewHostFunctionWithStack(
			"playlist_summary", func(_ context.Context, current *extism.CurrentPlugin, stack []uint64) {
				address, err := current.WriteBytes(r.summary())
				if err == nil {
					stack[0] = address
				}
			}, nil, []extism.ValueType{extism.ValueTypeI64},
		))
	}
	return extism.NewPlugin(ctx, manifest, config, functions)
}

func (r *Registry) packagePath(id, version string) string {
	return filepath.Join(r.dir, id, version, "package.zip")
}

func atomicWrite(name string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(name), 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(name), ".plugin-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	if err := errors.Join(err, f.Close()); err != nil {
		return err
	}
	return os.Rename(f.Name(), name)
}

func (r *Registry) save(next state) error {
	b, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(r.dir, "state.json"), b); err != nil {
		return err
	}
	r.state = next
	return nil
}

func (r *Registry) cloneState() state {
	next := state{Versions: make(map[string]string), Enabled: make(map[string]bool), Layout: r.state.Layout}
	for k, v := range r.state.Versions {
		next.Versions[k] = v
	}
	for k, v := range r.state.Enabled {
		next.Enabled[k] = v
	}
	return next
}

func (r *Registry) Install(ctx context.Context, b []byte) (Manifest, error) {
	p, err := unpack(b)
	if err != nil {
		return Manifest{}, err
	}
	if len(p.wasm) > 0 {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		vm, err := r.instantiate(ctx, p)
		if err != nil {
			return Manifest{}, err
		}
		defer vm.Close(context.Background())
		for _, cmd := range p.manifest.Commands {
			if !vm.FunctionExists(cmd.Export) {
				return Manifest{}, fmt.Errorf("插件缺少命令 %s", cmd.ID)
			}
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	m := p.manifest
	if old := r.state.Versions[m.ID]; old != "" && semver.Compare("v"+m.Version, "v"+old) < 0 {
		return Manifest{}, errors.New("插件版本低于已安装版本")
	}
	name := r.packagePath(m.ID, m.Version)
	if old, err := os.ReadFile(name); err == nil {
		if !bytes.Equal(old, b) {
			return Manifest{}, errors.New("同一插件版本的内容已经固定")
		}
	} else if !os.IsNotExist(err) {
		return Manifest{}, err
	} else if err := atomicWrite(name, b); err != nil {
		return Manifest{}, err
	}
	next := r.cloneState()
	next.Versions[m.ID] = m.Version
	if old, ok := r.active[m.ID]; ok {
		for _, permission := range m.Permissions {
			if !slices.Contains(old.manifest.Permissions, permission) {
				if next.Enabled[m.ID] {
					return Manifest{}, errors.New("新版插件申请了额外权限，请先停用旧版再更新")
				}
			}
		}
	}
	if next.Layout == m.ID && (m.Layout == nil || !next.Enabled[m.ID]) {
		next.Layout = ""
	}
	if err := r.save(next); err != nil {
		return Manifest{}, err
	}
	r.active[m.ID] = p
	return m, nil
}

func (r *Registry) Configure(id string, enabled, layout bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.active[id]
	if !ok || layout && (!enabled || p.manifest.Layout == nil) {
		return errors.New("插件或布局不可用")
	}
	next := r.cloneState()
	next.Enabled[id] = enabled
	if layout {
		next.Layout = id
	} else if next.Layout == id {
		next.Layout = ""
	}
	return r.save(next)
}

func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	next := r.cloneState()
	delete(next.Versions, id)
	delete(next.Enabled, id)
	if next.Layout == id {
		next.Layout = ""
	}
	if err := r.save(next); err != nil {
		return err
	}
	delete(r.active, id)
	return nil
}

func (r *Registry) Snapshot() Snapshot {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out Snapshot
	out.Plugins = make([]ActivePlugin, 0, len(r.active))
	for id, p := range r.active {
		out.Plugins = append(out.Plugins, ActivePlugin{Manifest: p.manifest, Enabled: r.state.Enabled[id]})
	}
	slices.SortFunc(out.Plugins, func(a, b ActivePlugin) int { return strings.Compare(a.Manifest.ID, b.Manifest.ID) })
	if p, ok := r.active[r.state.Layout]; ok && r.state.Enabled[r.state.Layout] {
		out.LayoutProvider, out.Layout = r.state.Layout, p.manifest.Layout
	}
	return out
}

func (r *Registry) Call(ctx context.Context, id, command string, args []byte) ([]byte, error) {
	r.mu.RLock()
	p, ok := r.active[id]
	enabled := r.state.Enabled[id]
	r.mu.RUnlock()
	if !ok || !enabled {
		return nil, errors.New("插件未启用")
	}
	for _, cmd := range p.manifest.Commands {
		if cmd.ID == command {
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			vm, err := r.instantiate(ctx, p)
			if err != nil {
				return nil, err
			}
			defer vm.Close(context.Background())
			code, result, err := vm.CallWithContext(ctx, cmd.Export, args)
			if err != nil || code != 0 || len(result) > 64<<10 {
				return nil, fmt.Errorf("插件命令失败 (%d): %v", code, err)
			}
			return result, nil
		}
	}
	return nil, errors.New("插件命令不存在")
}
