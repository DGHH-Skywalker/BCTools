//go:build windows

package kugoudb

import (
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const CanSelect = true

func discoverDatabase() string {
	for _, env := range []string{"APPDATA", "LOCALAPPDATA"} {
		if root := os.Getenv(env); filepath.IsAbs(root) {
			if file := databaseIn(filepath.Join(root, "KuGou8")); file != "" {
				return file
			}
		}
	}
	const windowsVersion = `Software\Microsoft\Windows\CurrentVersion\`
	for _, hive := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, view := range []uint32{registry.WOW64_64KEY, registry.WOW64_32KEY} {
			for _, exe := range []string{"KuGou.exe", "KuGouMusic.exe"} {
				path := registryPath(hive, view, windowsVersion+`App Paths\`+exe, "")
				if file := databaseIn(filepath.Dir(path)); file != "" {
					return file
				}
			}
			for _, name := range []string{"KuGou8", "KuGouMusic", "KuGou"} {
				if file := databaseIn(registryPath(hive, view, windowsVersion+`Uninstall\`+name, "InstallLocation")); file != "" {
					return file
				}
			}
		}
	}
	for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)"} {
		if root := os.Getenv(env); filepath.IsAbs(root) {
			for _, dir := range []string{"KuGou8", "KuGouMusic", `KuGou\KGMusic`, `KuGou\KuGou8`, "KuGou"} {
				if file := databaseIn(filepath.Join(root, dir)); file != "" {
					return file
				}
			}
		}
	}
	return ""
}

func registryPath(hive registry.Key, view uint32, subkey, value string) string {
	key, err := registry.OpenKey(hive, subkey, registry.QUERY_VALUE|view)
	if err != nil {
		return ""
	}
	defer key.Close()
	path, kind, err := key.GetStringValue(value)
	if err != nil {
		return ""
	}
	if kind == registry.EXPAND_SZ {
		path, err = registry.ExpandString(path)
		if err != nil {
			return ""
		}
	}
	return strings.Trim(strings.TrimSpace(path), `"`)
}

func databaseIn(dir string) string {
	if !filepath.IsAbs(dir) {
		return ""
	}
	for _, subdir := range []string{"", "Data"} {
		path := filepath.Join(dir, subdir, FileName)
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}
