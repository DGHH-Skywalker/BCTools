package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"broadcast-tool/converter"
	"broadcast-tool/handlers"
	"broadcast-tool/middleware"
	"broadcast-tool/paths"
	"broadcast-tool/routes"
	"broadcast-tool/store"

	"github.com/go-chi/chi/v5"
	"github.com/natefinch/lumberjack"
)

//go:embed embed/dist/*
//go:embed embed/dist/assets/*
var embeddedFiles embed.FS

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// 1. Init app data directories
	appDataDir, err := paths.GetAppDataDir()
	if err != nil {
		log.Fatalf("Cannot get APPDATA: %v", err)
	}

	for _, dir := range []string{
		appDataDir,
		paths.GetTempDir(appDataDir),
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

	log.Printf("Starting BroadcastTool, data dir: %s", appDataDir)

	// 3. Clean temp directory
	tempDir := paths.GetTempDir(appDataDir)
	entries, _ := os.ReadDir(tempDir)
	for _, e := range entries {
		os.RemoveAll(filepath.Join(tempDir, e.Name()))
	}

	// 4. Init store
	logger := func(format string, args ...interface{}) {
		log.Printf(format, args...)
	}
	st, err := store.New(appDataDir, logger)
	if err != nil {
		log.Fatalf("Failed to init store: %v", err)
	}

	// 5. Init converter (ffmpeg/ffprobe)
	// Priority: app data bin dir > exe same dir > PATH
	binDir := paths.GetBinDir(appDataDir)
	exeDir := getExecutableDir()
	ffmpegPath := resolveBinary("ffmpeg.exe", binDir, exeDir)
	ffprobePath := resolveBinary("ffprobe.exe", binDir, exeDir)

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

	// 6. Setup router with middleware
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

	// Shutdown endpoint: save data and exit
	r.Post("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Shutdown requested via API")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"shutting_down"}`))
		go func() {
			time.Sleep(200 * time.Millisecond)
			os.Exit(0)
		}()
	})

	// API routes
	routes.RegisterRoutes(r, routes.HandlerSet{

		Songs:    handlers.NewSongHandler(st),
		Files:    handlers.NewFileHandler(st, conv),
		Sync:     handlers.NewSyncHandler(st),
		Auth:     handlers.NewAuthHandler(st),
		Settings: handlers.NewSettingsHandler(st),
		Snapshot: handlers.NewSnapshotHandler(st),
		Update:   handlers.NewUpdateHandler(st),
	})

	// Static file serving for frontend SPA
	staticFS, err := fs.Sub(embeddedFiles, "embed/dist")
	if err != nil {
		log.Printf("No embedded frontend dist found, API-only mode: %v", err)
	} else {
		// Direct file serving from embedded FS (avoids http.FileServer 301 redirect issue)
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			data, err := fs.ReadFile(staticFS, "index.html")
			if err != nil {
				http.Error(w, "Not Found", 404)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-cache")
			w.Write(data)
		})
		r.HandleFunc("/*", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/") {
				return
			}
			p := strings.TrimPrefix(r.URL.Path, "/")
			if p == "" {
				p = "index.html"
			}

			data, err := fs.ReadFile(staticFS, p)
			if err != nil {
				// SPA fallback: serve index.html for client-side routing
				data, err = fs.ReadFile(staticFS, "index.html")
				if err != nil {
					http.Error(w, "Not Found", 404)
					return
				}
				w.Header().Set("Cache-Control", "no-cache")
			} else if p == "index.html" {
				w.Header().Set("Cache-Control", "no-cache")
			} else {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}

			// Content-Type by extension
			ext := filepath.Ext(p)
			switch ext {
			case ".html":
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
			case ".js":
				w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			case ".css":
				w.Header().Set("Content-Type", "text/css; charset=utf-8")
			case ".png":
				w.Header().Set("Content-Type", "image/png")
			case ".svg":
				w.Header().Set("Content-Type", "image/svg+xml")
			case ".json":
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
			}
			w.Write(data)
		})
	}

	// 7. Find available port
	port := findAvailablePort(1743)
	addr := fmt.Sprintf("0.0.0.0:%d", port)

	server := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	// 8. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("BroadcastTool v1.0.0 已启动: http://localhost:%d", port)
		log.Printf("局域网访问: http://%s:%d", getLocalIP(), port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// 9. Auto-open browser
	go func() {
		time.Sleep(500 * time.Millisecond)
		url := fmt.Sprintf("http://localhost:%d/", port)
		if err := openBrowser(url); err != nil {
			log.Printf("Failed to open browser: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	server.Shutdown(ctx)
	log.Println("Server stopped")
}

func findAvailablePort(start int) int {
	for port := start; port <= start+6; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
		if err == nil {
			ln.Close()
			return port
		}
	}
	return start
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}
	return "127.0.0.1"
}

func getExecutableDir() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

func resolveBinary(name, binDir, exeDir string) string {
	candidates := []string{
		filepath.Join(binDir, name),
		filepath.Join(exeDir, name),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func openBrowser(url string) error {
	return exec.Command("cmd", "/c", "start", "", url).Start()
}
