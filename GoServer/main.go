package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"broadcast-tool/browser"
	"broadcast-tool/converter"
	"broadcast-tool/handlers"
	"broadcast-tool/internal/binembed"
	"broadcast-tool/internal/repo"
	"broadcast-tool/internal/version"
	"broadcast-tool/launcher"
	"broadcast-tool/middleware"
	"broadcast-tool/network"
	"broadcast-tool/paths"
	"broadcast-tool/routes"
	"broadcast-tool/server"
	"broadcast-tool/store/deletedlogstore"
	"broadcast-tool/store/settingstore"
	"broadcast-tool/store/snapshotstore"
	"broadcast-tool/store/songstore"

	"github.com/go-chi/chi/v5"
	"github.com/natefinch/lumberjack"
)

//go:embed embed/dist/*
//go:embed embed/dist/assets/*
var embeddedFiles embed.FS

//go:embed embed/um-react/*
var umReactFiles embed.FS

// portStr / ephemeralStr 通过 ldflags -X 注入：
//
//	正式版（默认）: port=1743, ephemeral=false, 数据持久化于 %APPDATA%\BroadcastTool
//	一次性测试版  : port=712,  ephemeral=true,  数据写 %TEMP%\BCTools-test-<pid>，退出即清
var (
	portStr      = "1743"
	ephemeralStr = "false"
)

var ephemeralMode bool

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	var (
		devMode = flag.Bool("dev", false, "开发模式：替换已有后端")
	)
	flag.Parse()

	// 解析 ldflags 注入的端口与无痕模式
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		port = 1743
	}
	ephemeralMode, _ = strconv.ParseBool(ephemeralStr)

	appDataDir, err := paths.GetAppDataDir()
	if err != nil {
		log.Fatalf("Cannot get APPDATA: %v", err)
	}

	if ephemeralMode {
		// 一次性测试版：数据目录放临时目录，绝不触碰 %APPDATA%\BroadcastTool
		appDataDir = filepath.Join(os.TempDir(), fmt.Sprintf("BCTools-test-%d", os.Getpid()))
		cleanupStaleEphemeralDirs()
		log.Printf("一次性测试版：端口 %d，临时数据目录 %s（退出后清理）", port, appDataDir)
	}

	running := launcher.IsBackendRunning(port)

	// 一次性测试版：若端口被占用，直接结束占用进程
	if ephemeralMode && running {
		log.Printf("一次性测试版：端口 %d 被占用，尝试结束占用进程", port)
		if err := launcher.KillExistingBackend(port); err != nil {
			log.Printf("Failed to kill existing backend: %v", err)
		}
		if !launcher.WaitForPortFree(port, 10*time.Second) {
			log.Fatalf("Port %d is still in use after cleanup", port)
		}
		running = false
	}

	// 普通模式：后端已存在则只打开浏览器并退出
	if !*devMode && running {
		url := fmt.Sprintf("http://localhost:%d/", port)
		if err := browser.Open(url); err != nil {
			log.Printf("Failed to open browser: %v", err)
		}
		return
	}

	// 开发模式：强制替换已有后端
	if *devMode && running {
		if err := launcher.KillExistingBackend(port); err != nil {
			log.Printf("Failed to kill existing backend: %v", err)
		}
		if !launcher.WaitForPortFree(port, 10*time.Second) {
			log.Fatalf("Port %d is still in use after cleanup", port)
		}
	}

	runBackend(appDataDir, port, true)
}

// runBackend 初始化并运行后端服务
func runBackend(appDataDir string, port int, shouldOpenBrowser bool) {
	// 1. Init app data directories
	for _, dir := range []string{
		appDataDir,
		paths.GetTempDir(appDataDir),
		paths.GetSongsDir(appDataDir),
		paths.GetSnapshotsDir(appDataDir),
		paths.GetLogsDir(appDataDir),
		paths.GetBinDir(appDataDir),
	} {
		if err := paths.EnsureDir(dir); err != nil {
			log.Fatalf("Cannot create dir %s: %v", dir, err)
		}
	}

	// 2. Setup logging
	logFile := &lumberjack.Logger{
		Filename:  filepath.Join(appDataDir, "logs", "app.log"),
		MaxSize:   10,
		MaxAge:    7,
		LocalTime: true,
	}
	log.SetOutput(logFile)

	log.Printf("Starting XiaoBo Song Request Tool, data dir: %s", appDataDir)

	// 3. Clean temp directory
	tempDir := paths.GetTempDir(appDataDir)
	entries, _ := os.ReadDir(tempDir)
	for _, e := range entries {
		os.RemoveAll(filepath.Join(tempDir, e.Name()))
	}

	// 4. Init repository and domain stores
	logger := func(format string, args ...interface{}) {
		log.Printf(format, args...)
	}
	repo, err := repo.New(appDataDir, logger)
	if err != nil {
		log.Fatalf("Failed to init repo: %v", err)
	}
	songStore := songstore.New(repo)
	settingsStore := settingstore.New(repo)
	snapshotStore := snapshotstore.New(repo, logger)
	deletedLogStore := deletedlogstore.New(appDataDir)

	// 5. Init converter (ffmpeg/ffprobe)
	binDir := paths.GetBinDir(appDataDir)
	if err := binembed.Extract(binDir); err != nil {
		log.Printf("WARNING: failed to extract embedded ffmpeg/ffprobe: %v", err)
	}
	exeDir := paths.GetExecutableDir()
	ffmpegPath := launcher.ResolveBinary("ffmpeg.exe", binDir, exeDir)
	ffprobePath := launcher.ResolveBinary("ffprobe.exe", binDir, exeDir)

	ffmpegFound := true
	if ffmpegPath == "" {
		ffmpegFound = false
		log.Printf("WARNING: ffmpeg.exe not found. Audio conversion will fail. Place ffmpeg.exe next to the exe, in %s, or on system PATH.", binDir)
	} else {
		log.Printf("Using ffmpeg: %s", ffmpegPath)
	}
	ffprobeFound := true
	if ffprobePath == "" {
		ffprobeFound = false
		log.Printf("WARNING: ffprobe.exe not found. Audio metadata extraction will fall back to ffmpeg.")
	} else {
		log.Printf("Using ffprobe: %s", ffprobePath)
	}
	if !ffmpegFound || !ffprobeFound {
		log.Println("WARNING: Audio conversion requires ffmpeg.exe and ffprobe.exe.")
	}
	conv := converter.New(ffmpegPath, ffprobePath)

	// 7. Setup router with middleware
	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)

	log.Println("SECURITY NOTICE: This release does not implement backend session authentication for sensitive endpoints. Any device that can reach this server port may call admin interfaces. Run only in trusted local networks.")

	// Health check
	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Close-browser endpoint: only allow localhost; does NOT exit the backend
	r.Post("/api/close-browser", func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil || !network.IsLocalhost(host) {
			log.Printf("Close-browser request rejected from %s", r.RemoteAddr)
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		log.Println("Close-browser requested via API")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// 8. API routes
	routes.RegisterRoutes(r, routes.HandlerSet{
		Songs:    handlers.NewSongHandler(songStore, snapshotStore),
		Files:    handlers.NewFileHandler(appDataDir, settingsStore, conv),
		Auth:     handlers.NewAuthHandler(settingsStore),
		Settings: handlers.NewSettingsHandler(settingsStore, songStore, deletedLogStore),
		Snapshot: handlers.NewSnapshotHandler(snapshotStore),
		Update:   handlers.NewUpdateHandler(settingsStore),
		Network:  handlers.NewNetworkHandler(port),
		System:   handlers.NewSystemHandler(appDataDir, version.Version),
		Decrypt:  handlers.NewDecryptHandler(appDataDir),
	})

	// 10. Static file serving for frontend SPA
	// um-react (Unlock Music) 必须先注册，确保 /um-react/* 优先于下方 /* 兜底匹配
	server.RegisterUMReact(r, umReactFiles)
	server.RegisterStatic(r, embeddedFiles)

	// 11. Start server
	addr := fmt.Sprintf("0.0.0.0:%d", port)
	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	launcher.SafeGo("server", func() {
		log.Printf("小播点歌工具 v%s 已启动: http://localhost:%d", version.Version, port)
		log.Printf("局域网访问: http://%s:%d", network.GetLocalIP(), port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	})

	// 12. Auto-open browser
	if shouldOpenBrowser {
		launcher.SafeGo("browser", func() {
			time.Sleep(500 * time.Millisecond)
			url := fmt.Sprintf("http://localhost:%d/", port)
			if err := browser.Open(url); err != nil {
				log.Printf("Failed to open browser: %v", err)
			}
		})
	}

	<-quit
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("Server stopped")

	// 一次性测试版：退出时清理临时数据目录，不留痕迹
	if ephemeralMode {
		log.SetOutput(os.Stderr) // 切离日志文件，避免占用导致删除失败
		logFile.Close()
		if rmErr := os.RemoveAll(appDataDir); rmErr != nil {
			log.Printf("清理临时目录失败: %v", rmErr)
		} else {
			log.Printf("已清理临时数据目录: %s", appDataDir)
		}
	}
}

// cleanupStaleEphemeralDirs 清理 %TEMP% 下残留的旧 BCTools-test-* 目录
// （应对上次测试版被强杀未能正常清理的情况）。
func cleanupStaleEphemeralDirs() {
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), "BCTools-test-") {
			os.RemoveAll(filepath.Join(os.TempDir(), e.Name()))
		}
	}
}
