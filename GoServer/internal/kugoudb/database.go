package kugoudb

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"broadcast-tool/paths"

	"github.com/google/uuid"
)

const (
	FileName = "KGMusicV3.db"
	maxSize  = 128 << 20
)

// Database owns a private copy and an in-memory index. The source file is read-only.
type Database struct {
	dir      string
	discover func() string
	update   sync.Mutex
	mu       sync.RWMutex
	state    savedSource
	keys     map[string]string
}

type savedSource struct {
	Selected string    `json:"selected,omitempty"`
	Source   string    `json:"source"`
	Snapshot uuid.UUID `json:"snapshot"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
}

func New(dataDir string) *Database {
	return &Database{dir: paths.GetKugouDir(dataDir), discover: discoverDatabase}
}

func (d *Database) Status() (custom bool, keyCount int) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.state.Selected != "", len(d.keys)
}

func (d *Database) Key(hash string) string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.keys[hash]
}

func (d *Database) snapshot(id uuid.UUID) string {
	return filepath.Join(d.dir, id.String()+".db")
}

func (d *Database) publish(state savedSource, keys map[string]string) {
	d.mu.Lock()
	d.state, d.keys = state, keys
	d.mu.Unlock()
}

// Initialize restores the current snapshot and checks the source once per launch.
// A missing source leaves the last usable snapshot active.
func (d *Database) Initialize(ctx context.Context) error {
	d.update.Lock()
	defer d.update.Unlock()
	state, err := d.restore(ctx)
	return errors.Join(err, d.refresh(ctx, state, state.Selected))
}

func (d *Database) restore(ctx context.Context) (savedSource, error) {
	var state savedSource
	data, err := os.ReadFile(filepath.Join(d.dir, "source.json"))
	if errors.Is(err, os.ErrNotExist) {
		return state, nil
	}
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return savedSource{}, err
	}
	d.publish(state, nil)
	if state.Snapshot == uuid.Nil {
		return state, nil
	}
	data, _, err = readDatabase(d.snapshot(state.Snapshot))
	if err != nil {
		return state, err
	}
	keys, err := extractKeys(ctx, data)
	if err == nil {
		d.publish(state, keys)
	}
	return state, err
}

// Select validates and copies the explicitly selected file before committing it.
// An empty selection restores the standard Windows search locations.
func (d *Database) Select(ctx context.Context, source string) error {
	d.update.Lock()
	defer d.update.Unlock()
	if source != "" {
		if !filepath.IsAbs(source) || !strings.EqualFold(filepath.Ext(source), ".db") {
			return errors.New("请选择酷狗音乐的 .db 数据库文件")
		}
		if paths.IsKugouDataPath(filepath.Dir(d.dir), source) {
			return errors.New("请选择酷狗音乐的源数据库文件")
		}
	}
	return d.refresh(ctx, d.state, source)
}

func (d *Database) refresh(ctx context.Context, state savedSource, selected string) error {
	source := selected
	if source == "" {
		source = d.discover()
	}
	if source != "" {
		info, err := os.Stat(source)
		if err == nil {
			if source == state.Source && state.Selected == selected && d.keys != nil &&
				info.Size() == state.Size && info.ModTime().Equal(state.Modified) {
				return nil
			}
			err = d.replace(ctx, source, selected, state.Snapshot)
			if !errors.Is(err, os.ErrNotExist) || selected != state.Selected {
				return err
			}
		}
		if !errors.Is(err, os.ErrNotExist) || selected != state.Selected {
			return err
		}
	}
	if state.Selected != selected {
		state.Selected = selected
		if err := d.save(state); err != nil {
			return err
		}
		d.publish(state, d.keys)
	}
	return nil
}

func (d *Database) replace(ctx context.Context, source, selected string, previous uuid.UUID) error {
	data, info, err := readDatabase(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d.dir, 0700); err != nil {
		return err
	}
	id := uuid.New()
	snapshot := d.snapshot(id)
	if err := atomicWrite(snapshot, data); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(snapshot)
		}
	}()
	keys, err := extractKeys(ctx, data)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	state := savedSource{selected, source, id, info.Size(), info.ModTime()}
	if err := d.save(state); err != nil {
		return err
	}
	committed = true
	d.publish(state, keys)
	if previous != uuid.Nil {
		_ = os.Remove(d.snapshot(previous))
	}
	return nil
}

func (d *Database) save(state savedSource) error {
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(d.dir, "source.json"), data)
}

func readDatabase(path string) ([]byte, os.FileInfo, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()
	before, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}
	if !before.Mode().IsRegular() || before.Size() < 1024 || before.Size() > maxSize {
		return nil, nil, errors.New("酷狗数据库大小或文件类型无效")
	}
	data := make([]byte, int(before.Size()))
	if _, err := io.ReadFull(file, data); err != nil {
		return nil, nil, err
	}
	after, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return nil, nil, errors.New("酷狗数据库在读取期间发生变化，请稍后重试")
	}
	return data, before, nil
}

func atomicWrite(path string, data []byte) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".kugou-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, err = file.Write(data)
	if err == nil {
		err = file.Sync()
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
