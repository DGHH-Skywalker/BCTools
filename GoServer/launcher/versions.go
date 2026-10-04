package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Installation struct {
	Version string   `json:"version"`
	Path    string   `json:"path"`
	SHA256  string   `json:"sha256"`
	Source  string   `json:"source"`
	Sources []string `json:"sources,omitempty"`
}

func CompareVersions(a, b string) int {
	aa, bb := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(aa) || i < len(bb); i++ {
		x, y := 0, 0
		if i < len(aa) {
			x, _ = strconv.Atoi(aa[i])
		}
		if i < len(bb) {
			y, _ = strconv.Atoi(bb[i])
		}
		if x > y {
			return 1
		}
		if x < y {
			return -1
		}
	}
	return 0
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// LockStartup serializes cooperating launches across paths, including migration.
// The OS releases the listener if a launcher crashes; no stale lockfile recovery.
func LockStartup(port int) (net.Listener, error) {
	deadline := time.Now().Add(45 * time.Second)
	for {
		l, err := net.Listen("tcp4", fmt.Sprintf("127.0.0.1:%d", port+1))
		if err == nil {
			return l, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("启动协调端口被占用，未终止任何进程：%w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// RegisterVersion keeps an immutable copy so moving/deleting Downloads does not
// lose the newest registered version. Hashes detect stale/changed copies; this is
// NOT a publisher signature or a trust boundary against the current OS user.
func RegisterVersion(dir, executable, version string) (Installation, []Installation, error) {
	if !regexp.MustCompile(`^[0-9]+(?:\.[0-9]+){1,3}$`).MatchString(version) {
		return Installation{}, nil, fmt.Errorf("无效的程序版本：%q", version)
	}
	registry := filepath.Join(dir, "installations.json")
	var installs []Installation
	if b, err := os.ReadFile(registry); err == nil {
		if err := json.Unmarshal(b, &installs); err != nil {
			return Installation{}, nil, err
		}
	} else if !os.IsNotExist(err) {
		return Installation{}, nil, err
	}
	digest, err := fileHash(executable)
	if err != nil {
		return Installation{}, nil, err
	}
	target := filepath.Join(dir, "versions", version+"-"+digest[:16], "bctools.exe")
	if filepath.Clean(executable) != filepath.Clean(target) {
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return Installation{}, nil, err
		}
		got, _ := fileHash(target)
		if got != digest {
			in, err := os.Open(executable)
			if err != nil {
				return Installation{}, nil, err
			}
			defer in.Close()
			out, err := os.OpenFile(target+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
			if err != nil {
				return Installation{}, nil, err
			}
			_, err = io.Copy(out, in)
			if err == nil {
				err = out.Sync()
			}
			closeErr := out.Close()
			if err != nil {
				return Installation{}, nil, err
			}
			if closeErr != nil {
				return Installation{}, nil, closeErr
			}
			if err := os.Rename(target+".tmp", target); err != nil {
				return Installation{}, nil, err
			}
		}
	}
	current := Installation{Version: version, Path: target, SHA256: digest, Source: executable}
	best := current
	found := false
	for i := range installs {
		item := installs[i]
		if item.Path == target {
			found = true
			if executable != target && executable != item.Source {
				present := false
				for _, source := range item.Sources {
					if source == executable {
						present = true
					}
				}
				if !present {
					installs[i].Sources = append(installs[i].Sources, executable)
				}
			}
		}
		// Do not launch executables outside the private managed versions tree.
		rel, err := filepath.Rel(filepath.Join(dir, "versions"), item.Path)
		if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
			continue
		}
		if hash, err := fileHash(item.Path); err == nil && hash == item.SHA256 && CompareVersions(item.Version, best.Version) > 0 {
			best = item
		}
	}
	if !found {
		installs = append(installs, current)
	}
	b, err := json.MarshalIndent(installs, "", "  ")
	if err != nil {
		return Installation{}, nil, err
	}
	if err := writeRegistryFile(registry, b); err != nil {
		return Installation{}, nil, err
	}
	return best, installs, nil
}

type RuntimeInfo struct {
	Application   string `json:"application"`
	Version       string `json:"version"`
	Protocol      int    `json:"handoffProtocol"`
	PID           int    `json:"pid"`
	Executable    string `json:"executable"`
	InstanceID    string `json:"instanceId"`
	DatabaseID    string `json:"databaseId"`
	Port          int    `json:"port"`
	DataDirectory string `json:"dataDirectory"`
}

func RunningInfo(port int) (RuntimeInfo, error) {
	var info RuntimeInfo
	c := &http.Client{Timeout: 3 * time.Second}
	r, err := c.Get(fmt.Sprintf("http://127.0.0.1:%d/api/health", port))
	if err != nil {
		return info, err
	}
	defer r.Body.Close()
	err = json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&info)
	if err != nil {
		return info, err
	}
	if r.StatusCode != http.StatusOK || info.Application != "BCTools" || info.Protocol != 2 || info.InstanceID == "" {
		return info, fmt.Errorf("正在运行旧版或无法验证的服务。请先在旧页面保存歌单并从托盘退出，再启动新版；不会强制结束它")
	}
	return info, nil
}

func StopSafely(port int) error {
	info, err := RunningInfo(port)
	if err != nil {
		return err
	}
	process, err := openRuntimeProcess(info)
	if err != nil {
		return err
	}
	defer process.Close()
	c := &http.Client{Timeout: 75 * time.Second}
	r, err := c.Post(fmt.Sprintf("http://127.0.0.1:%d/api/shutdown", port), "application/json", nil)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(io.LimitReader(r.Body, 4096))
		return fmt.Errorf("旧程序未确认安全保存，已停止切换：%s", b)
	}
	var ack map[string]string
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&ack); err != nil || ack["saved"] != "true" || ack["instanceId"] != info.InstanceID {
		return fmt.Errorf("旧实例未提供完整保存确认，已取消切换")
	}
	if !process.Wait(30 * time.Second) {
		// This handle was opened and identity-checked before the shutdown
		// request, preventing PID reuse from selecting another process.
		if err := process.Terminate(); err != nil {
			return err
		}
		if !process.Wait(5 * time.Second) {
			return fmt.Errorf("保存已确认，但旧进程未退出")
		}
	}
	if !WaitForPortFree(port, 5*time.Second) {
		return fmt.Errorf("端口仍在使用，已停止启动")
	}
	return nil
}

type runtimeProcess interface {
	Close() error
	Wait(time.Duration) bool
	Terminate() error
}

// The runtime locator lets a launcher find the owner of the fixed database
// even when a diagnostic/test build uses a different HTTP port.
func FindRunningPort(root string, fallback int) int {
	var previous RuntimeInfo
	if b, err := os.ReadFile(filepath.Join(root, "runtime.json")); err == nil && json.Unmarshal(b, &previous) == nil && previous.Port > 0 {
		if live, err := RunningInfo(previous.Port); err == nil && live.InstanceID == previous.InstanceID && live.DataDirectory == root {
			return previous.Port
		}
	}
	return fallback
}
func WriteRuntime(root string, port int) error {
	info, err := RunningInfo(port)
	if err != nil {
		return err
	}
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return writeRegistryFile(filepath.Join(root, "runtime.json"), b)
}

func writeRegistryFile(path string, b []byte) error {
	f, err := os.OpenFile(path+".tmp", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(path+".tmp", path)
}
