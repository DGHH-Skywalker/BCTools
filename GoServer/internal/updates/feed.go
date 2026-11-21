package updates

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/theupdateframework/go-tuf/v2/metadata"
	tufconfig "github.com/theupdateframework/go-tuf/v2/metadata/config"
	tuf "github.com/theupdateframework/go-tuf/v2/metadata/updater"
)

//go:embed source.json
var sourceConfiguration []byte

var ErrUnconfigured = errors.New("热更新仓库或签名根尚未配置")
var repositoryPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var branchPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

type SourceConfig struct {
	Repository string          `json:"repository"`
	Branch     string          `json:"branch"`
	Root       json.RawMessage `json:"root"`
	Mirrors    []string        `json:"mirrors"`
}

type source struct {
	name   string
	meta   string
	target string
}

type networkFetcher struct {
	ctx    context.Context
	client *http.Client
}

func (f networkFetcher) DownloadFile(raw string, maxLength int64, _ time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(f.ctx, 7*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Cache-Control", "no-cache")
	res, err := f.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, &metadata.ErrDownloadHTTP{StatusCode: res.StatusCode, URL: raw}
	}
	b, err := io.ReadAll(io.LimitReader(res.Body, maxLength+1))
	if int64(len(b)) > maxLength {
		return nil, errors.New("超过更新元数据大小限制")
	}
	return b, err
}

func candidateSources(cfg SourceConfig) ([]source, error) {
	if cfg.Repository == "OWNER/REPO" || len(cfg.Root) == 0 || string(cfg.Root) == "null" {
		return nil, ErrUnconfigured
	}
	if !repositoryPattern.MatchString(cfg.Repository) || !branchPattern.MatchString(cfg.Branch) {
		return nil, errors.New("更新仓库地址无效")
	}
	origin := "https://raw.githubusercontent.com/" + cfg.Repository + "/" + cfg.Branch
	sources := make([]source, 0, len(cfg.Mirrors)+4)
	for _, base := range cfg.Mirrors {
		u, err := url.Parse(base)
		if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return nil, errors.New("更新镜像必须是 HTTPS 基础地址")
		}
		base = strings.TrimRight(base, "/")
		sources = append(sources, source{u.Host, base + "/metadata/", base + "/targets/"})
	}
	sources = append(sources, source{"GitHub", origin + "/metadata/", origin + "/targets/"})
	for _, host := range []string{"gh-proxy.com", "ghfast.top", "ghproxy.net"} {
		base := "https://" + host + "/" + origin
		sources = append(sources, source{host, base + "/metadata/", base + "/targets/"})
	}
	return sources, nil
}

func (m *Manager) preferredSources() []source {
	sources := slices.Clone(m.sources)
	m.mu.RLock()
	last := m.state.Source
	m.mu.RUnlock()
	if i := slices.IndexFunc(sources, func(s source) bool { return s.name == last }); i > 0 {
		preferred := sources[i]
		copy(sources[1:i+1], sources[:i])
		sources[0] = preferred
	}
	return sources
}

func (m *Manager) updater(ctx context.Context, src source, local bool) (*tuf.Updater, error) {
	cfg, err := tufconfig.New(src.meta, m.source.Root)
	if err != nil {
		return nil, err
	}
	cfg.LocalMetadataDir = filepath.Join(m.dir, "trust")
	cfg.LocalTargetsDir = filepath.Join(m.dir, "download-cache")
	cfg.RemoteTargetsURL = src.target
	cfg.Fetcher = networkFetcher{ctx: ctx, client: m.client}
	cfg.UnsafeLocalMode = local
	if err := os.MkdirAll(cfg.LocalMetadataDir, 0700); err != nil {
		return nil, err
	}
	return tuf.New(cfg)
}
