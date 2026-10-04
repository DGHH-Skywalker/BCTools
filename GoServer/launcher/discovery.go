package launcher

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

func SearchDirectories(dir, executable string) []string {
	dirs := []string{filepath.Dir(executable), filepath.Join(dir, "versions")}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, "Desktop"), filepath.Join(home, "Downloads"))
	}
	if local, err := os.UserCacheDir(); err == nil {
		dirs = append(dirs, filepath.Join(local, "Programs", "BCTools"))
	}
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		dirs = append(dirs, filepath.Join(pf, "BCTools"))
	}
	var extra []string
	if b, err := os.ReadFile(filepath.Join(dir, "search-directories.json")); err == nil {
		_ = json.Unmarshal(b, &extra)
	}
	dirs = append(dirs, extra...)
	var installs []Installation
	if b, err := os.ReadFile(filepath.Join(dir, "installations.json")); err == nil {
		_ = json.Unmarshal(b, &installs)
	}
	for _, install := range installs {
		dirs = append(dirs, filepath.Dir(install.Source))
		for _, source := range install.Sources {
			dirs = append(dirs, filepath.Dir(source))
		}
	}
	return dirs
}

// Discovery never executes uninspected candidates. Search user-selected roots,
// skipping reparse/symlink entries and limiting recursion in common folders.
func DiscoverVersions(dir, executable string) ([]Installation, error) {
	var found []Installation
	seen := map[string]bool{}
	for _, root := range SearchDirectories(dir, executable) {
		root, err := filepath.Abs(root)
		if err != nil || seen[strings.ToLower(root)] {
			continue
		}
		seen[strings.ToLower(root)] = true
		walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				if entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if entry.IsDir() {
				if strings.Count(filepath.ToSlash(rel), "/") >= 3 {
					return filepath.SkipDir
				}
				return nil
			}
			name := strings.ToLower(entry.Name())
			if !strings.HasSuffix(name, ".exe") {
				return nil
			}
			v, err := executableVersion(path)
			if err != nil {
				return nil
			}
			_, installs, err := RegisterVersion(dir, path, v)
			if err != nil {
				return err
			}
			found = installs
			return nil
		})
		if walkErr != nil {
			return nil, walkErr
		}
	}
	return found, nil
}
