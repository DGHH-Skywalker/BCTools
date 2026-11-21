//go:build windows

package main

import (
	_ "embed"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"broadcast-tool/internal/requestscope"
	"broadcast-tool/internal/securelocal"
	"broadcast-tool/internal/updates"
	"broadcast-tool/response"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

//go:embed assets/icon-bc.ico
var desktopIcon []byte

func checkDesktop() error {
	if runtime.GOARCH != "amd64" {
		return errors.New("桌面模式支持 Windows x64")
	}
	const client = `SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}`
	for _, hive := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, view := range []uint32{registry.WOW64_32KEY, registry.WOW64_64KEY} {
			key, err := registry.OpenKey(hive, client, registry.QUERY_VALUE|view)
			if err != nil {
				continue
			}
			version, _, err := key.GetStringValue("pv")
			key.Close()
			if err == nil && version != "" && version != "0.0.0.0" {
				return nil
			}
		}
	}
	return errors.New("桌面模式需要 Microsoft Edge WebView2 Runtime。可从 Microsoft 安装运行时，或双击程序使用浏览器模式")
}

func showLaunchError(message string) {
	text, err := windows.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	title, _ := windows.UTF16PtrFromString("BCTools")
	windows.MessageBox(0, text, title, windows.MB_OK|windows.MB_ICONERROR)
}

func runDesktop(router http.Handler, stop <-chan struct{}, commands <-chan string, dataDir string) error {
	done := make(chan struct{})
	started := make(chan struct{})
	defer close(done)
	var app *application.App
	windowsByName := make(map[string]*application.WebviewWindow)
	var closing atomic.Bool
	var startedOnce sync.Once
	var openWindow func(name, address string) *application.WebviewWindow
	openWindow = func(name, address string) *application.WebviewWindow {
		win := windowsByName[name]
		if win == nil {
			win = app.Window.NewWithOptions(application.WebviewWindowOptions{
				Name: name, Title: "小播点歌工具", URL: address, Width: 1150, Height: 760,
			})
			windowsByName[name] = win
			win.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
				if !closing.Load() {
					win.Hide()
					event.Cancel()
				}
			})
		} else if address != "" {
			win.SetURL(address)
		}
		win.Show()
		win.Focus()
		return win
	}

	assets := requestscope.InDesktop(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/desktop/window":
			securelocal.Require(func(w http.ResponseWriter, r *http.Request) {
				name, address, ok := desktopDestination(r.URL.Query())
				if r.Method != http.MethodPost || !ok {
					response.WriteError(w, http.StatusBadRequest, "INVALID_WINDOW", "无效的桌面窗口")
					return
				}
				application.InvokeSync(func() {
					openWindow(name, address)
					if settings := windowsByName["settings"]; name == "main" && settings != nil {
						settings.Hide()
					}
				})
				response.WriteJSON(w, http.StatusOK, map[string]bool{"opened": true})
			})(w, r)
		case "/api/desktop/save":
			securelocal.Require(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				name := filepath.Base(r.URL.Query().Get("name"))
				if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `:/\`) {
					response.WriteError(w, http.StatusBadRequest, "INVALID_FILENAME", "无效的文件名")
					return
				}
				saveDesktopFile(app, w, r, name)
			})(w, r)
		default:
			router.ServeHTTP(w, r)
		}
	}))

	app = application.New(application.Options{
		Name: "BCTools", Icon: desktopIcon,
		Assets: application.AssetOptions{Handler: assets},
	})
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "main", Title: "小播点歌工具", URL: "/", Width: 1240, Height: 840,
	})
	windowsByName["main"] = mainWindow
	mainWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !closing.Load() {
			mainWindow.Hide()
			event.Cancel()
		}
	})
	show := func() { mainWindow.Show(); mainWindow.Focus() }
	quit := func() { closing.Store(true); app.Quit() }
	tray := app.SystemTray.New()
	tray.SetIcon(desktopIcon)
	tray.SetTooltip("小播点歌工具")
	menu := application.NewMenu()
	menu.Add("打开界面").OnClick(func(*application.Context) { show() })
	menu.Add("退出程序").OnClick(func(*application.Context) { quit() })
	tray.SetMenu(menu)

	go func() {
		select {
		case <-started:
		case <-done:
			return
		}
		for {
			select {
			case <-stop:
				application.InvokeSync(quit)
				return
			case command := <-commands:
				if command == "shutdown" {
					application.InvokeSync(quit)
					return
				}
				application.InvokeSync(show)
			case <-done:
				return
			}
		}
	}()
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		startedOnce.Do(func() { close(started) })
	})
	mainWindow.OnWindowEvent(events.Common.WindowRuntimeReady, func(*application.WindowEvent) {
		updates.SignalReady(dataDir)
	})
	return app.Run()
}

func desktopDestination(q url.Values) (string, string, bool) {
	switch q.Get("view") {
	case "main":
		return "main", "", true
	case "unlock":
		return "unlock", "/um-react/", true
	case "settings":
		switch route := q.Get("route"); route {
		case "/settings", "/settings/slots", "/settings/broadcast", "/settings/export", "/settings/guide":
			return "settings", "/#" + route, true
		}
	case "import":
		stage := q.Get("stage")
		if len(stage) == 36 && !strings.ContainsAny(stage, "/?#&") {
			return "import", "/#/dorm/manage?stage=" + url.QueryEscape(stage), true
		}
	}
	return "", "", false
}

func saveDesktopFile(app *application.App, w http.ResponseWriter, r *http.Request, name string) {
	dialog := app.Dialog.SaveFile().SetFilename(name)
	path, err := dialog.PromptForSingleSelection()
	if err != nil || path == "" {
		response.WriteJSON(w, http.StatusOK, map[string]bool{"saved": false})
		return
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".bctools-save-*")
	if err == nil {
		defer os.Remove(file.Name())
		_, err = io.Copy(file, http.MaxBytesReader(w, r.Body, 512<<20))
		if err == nil {
			err = file.Sync()
		}
		err = errors.Join(err, file.Close())
		if err == nil {
			err = os.Rename(file.Name(), path)
		}
	}
	if err != nil {
		response.WriteError(w, http.StatusInternalServerError, "SAVE_FAILED", "无法保存文件")
		return
	}
	response.WriteJSON(w, http.StatusOK, map[string]bool{"saved": true})
}
