package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"broadcast-tool/converter"
	"broadcast-tool/handlers"
	"broadcast-tool/internal/binembed"
	"broadcast-tool/internal/plugins"
	"broadcast-tool/internal/repo"
	"broadcast-tool/internal/updates"
	"broadcast-tool/internal/version"
	"broadcast-tool/launcher"
	"broadcast-tool/middleware"
	"broadcast-tool/paths"
	"broadcast-tool/platform"
	"broadcast-tool/routes"
	"broadcast-tool/server"
	"broadcast-tool/services"
	"broadcast-tool/store/deletedlogstore"
	"broadcast-tool/store/settingstore"
	"broadcast-tool/store/snapshotstore"
	"broadcast-tool/store/songstore"
	"broadcast-tool/tray"

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
//	正式版（默认）: port=1743, ephemeral=false, 数据持久化于 <exe>\BctoolData
//	一次性测试版  : port=712,  ephemeral=true,  数据写 %TEMP%\BCTools-test-<pid>，退出即清
var (
	portStr      = "1743"
	ephemeralStr = "false"
)

func main() {
	if err := run(); err != nil {
		log.Print(err)
		if !slices.Contains(os.Args[1:], "--shutdown") {
			showLaunchError(err.Error())
		}
		os.Exit(1)
	}
}

func run() error {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	devMode := flag.Bool("dev", false, "开发模式")
	shutdownOnly := flag.Bool("shutdown", false, "关闭正在运行的 BCTools")
	applyUpdate := flag.Bool("apply-staged-update", false, "更新进程")
	flag.Parse()
	if *applyUpdate {
		return updates.RunCoreHelper(version.Version, flag.NArg() == 1 && flag.Arg(0) == "desktop")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		port = 1743
	}
	desktop := flag.NArg() == 1 && flag.Arg(0) == "desktop"
	ephemeral, _ := strconv.ParseBool(ephemeralStr)
	dataDir, err := paths.GetDataRootDir()
	if err != nil {
		return err
	}
	if ephemeral && !*shutdownOnly {
		dataDir, err = os.MkdirTemp("", "BCTools-test-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(dataDir)
	}
	if desktop {
		if err := checkDesktop(); err != nil {
			return err
		}
	}
	if *shutdownOnly {
		return launcher.SendControl(dataDir, "shutdown")
	}

	mode := "browser"
	if desktop {
		mode = "desktop"
	}
	control := make(chan string, 8)
	ownership, err := claimBackend(dataDir, mode, *devMode, port, control)
	if err != nil || ownership == nil {
		return err
	}
	defer ownership.Close()

	address := fmt.Sprintf("0.0.0.0:%d", port)
	browserURL := fmt.Sprintf("http://localhost:%d/", port)
	listener, listenErr := net.Listen("tcp4", address)
	if listenErr != nil {
		runningVersion := launcher.BackendVersion(port)
		if runningVersion != "" && !desktop && !*devMode {
			return platform.OpenBrowser(browserURL)
		}
		if runningVersion != "" && (*devMode || strings.HasPrefix(runningVersion, "5.")) {
			if err := launcher.ShutdownBrowserBackend(port); err != nil {
				return fmt.Errorf("无法关闭现有 BCTools 后端: %w", err)
			}
			if !launcher.WaitForPortFree(port, 20*time.Second) {
				return errors.New("等待现有 BCTools 后端退出超时")
			}
			listener, listenErr = net.Listen("tcp4", address)
		}
	}
	if listenErr != nil && !desktop {
		return fmt.Errorf("无法启动 BCTools HTTP 服务: %w", listenErr)
	}
	if listener != nil {
		defer listener.Close()
	}

	legacy := !ephemeral && services.PrepareLegacyMigration(dataDir, log.Printf)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	requestStop := make(chan struct{})
	var stopOnce sync.Once
	stop := func() { stopOnce.Do(func() { close(requestStop) }) }
	app, err := newBackend(ctx, dataDir, port, legacy, desktop, listenErr, stop)
	if err != nil {
		return err
	}
	defer app.close()
	if listener != nil {
		go func() {
			if err := app.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Printf("HTTP 服务停止: %v", err)
				app.network.SetUnavailable(err)
			}
		}()
	}
	signals, stopSignals := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()
	go func() {
		<-signals.Done()
		stop()
	}()

	if desktop {
		return runDesktop(app.router, requestStop, control, dataDir)
	}

	openBrowser := func() {
		if err := platform.OpenBrowser(browserURL); err != nil {
			log.Printf("打开浏览器失败: %v", err)
		}
	}
	updates.SignalReady(dataDir)
	if !*devMode {
		openBrowser()
	}
	go func() {
		select {
		case <-requestStop:
		case <-control:
		}
		app.close()
		tray.Stop()
	}()
	tray.Run(tray.Config{
		Tooltip: fmt.Sprintf("小播点歌工具 v%s", version.Version),
		OnOpen:  openBrowser,
		OnExit:  app.close,
	})
	return nil
}

func claimBackend(dataDir, mode string, replace bool, port int, control chan<- string) (io.Closer, error) {
	deadline := time.Now().Add(25 * time.Second)
	replacementRequested := false
	for {
		owner, err := launcher.Claim(dataDir, mode, control)
		if err != nil || owner != nil {
			return owner, err
		}
		if !replacementRequested {
			switch {
			case replace:
				err = launcher.SendControl(dataDir, "shutdown")
				replacementRequested = err == nil
			case mode == "desktop":
				err = launcher.SendControl(dataDir, "desktop")
				switch {
				case errors.Is(err, launcher.ErrSwitching):
					replacementRequested = true
				case err == nil:
					return nil, nil
				}
			default:
				if err = launcher.SendControl(dataDir, "status"); err == nil {
					if !launcher.IsBackendRunning(port) {
						return nil, errors.New("BCTools 正在运行，但局域网服务不可用")
					}
					return nil, platform.OpenBrowser(fmt.Sprintf("http://localhost:%d/", port))
				}
			}
		}
		if time.Now().After(deadline) {
			return nil, errors.New("等待 BCTools 启动或退出超时")
		}
		time.Sleep(100 * time.Millisecond)
	}
}

type backend struct {
	router  http.Handler
	server  *http.Server
	network *handlers.NetworkHandler
	close   func()
}

func newBackend(ctx context.Context, appDataDir string, port int, migratedLegacy, desktop bool, listenErr error, requestStop func()) (*backend, error) {
	// 1. Init app data directories
	for _, dir := range []string{
		appDataDir,
		paths.GetTempDir(appDataDir),
		paths.GetMusicFilesDir(appDataDir),
		paths.GetSongStagingDir(appDataDir),
		paths.GetSnapshotsDir(appDataDir),
		paths.GetLogsDir(appDataDir),
		paths.GetBinDir(appDataDir),
	} {
		if err := paths.EnsureDir(dir); err != nil {
			return nil, fmt.Errorf("create %s: %w", dir, err)
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
	repository, err := repo.New(appDataDir, logger)
	if err != nil {
		return nil, err
	}
	songStore := songstore.New(repository)
	settingsStore := settingstore.New(repository)
	snapshotStore := snapshotstore.New(repository, logger)
	deletedLogStore := deletedlogstore.New(appDataDir)
	updateLogService, err := services.NewUpdateLogService(appDataDir)
	if err != nil {
		return nil, err
	}

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

	// 6. MusicLibrary：周文件夹（MusicFiles\<YYYY年第N周>）布局引擎
	library := services.NewMusicLibrary(appDataDir, repository, conv, logger)
	if migratedLegacy {
		// 阶段二：旧 uuid 歌曲文件倒进暂存区后按周重建布局
		services.FinishLegacyMigration(appDataDir, library, logger)
	}

	pluginRegistry, err := plugins.New(filepath.Join(appDataDir, "plugins"), func() []byte {
		var dorm, broadcast int
		repository.Read(func(data *repo.Data) { dorm, broadcast = len(data.DormSongs), len(data.BroadcastSongs) })
		result, _ := json.Marshal(map[string]int{"dorm": dorm, "broadcast": broadcast})
		return result
	}, func(err error) { log.Print(err) })
	if err != nil {
		return nil, err
	}
	updateManager, err := updates.New(appDataDir, version.Version, pluginRegistry)
	if err != nil {
		return nil, err
	}
	workerCtx, cancelWorker := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			if err := library.CollectRetiredSongs(time.Now()); err != nil {
				log.Printf("保留期清理: %v", err)
			}
			select {
			case <-workerCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	// 7. Setup router with middleware
	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(middleware.Logging)
	r.Use(middleware.Recovery)

	log.Println("SECURITY NOTICE: This release does not implement backend session authentication for sensitive endpoints. Any device that can reach this server port may call admin interfaces. Run only in trusted local networks.")

	lifecycle := handlers.NewLifecycleHandler(requestStop)
	networkInfo := handlers.NewNetworkHandler(port)
	if listenErr != nil {
		networkInfo.SetUnavailable(listenErr)
	}

	routes.RegisterRoutes(r, routes.HandlerSet{
		Extensions: &handlers.ExtensionsHandler{
			Repository: repository, Library: library, Plugins: pluginRegistry, Updates: updateManager, Stop: requestStop, Desktop: desktop,
		},
		Songs:     handlers.NewSongHandler(songStore, snapshotStore, library),
		Files:     handlers.NewFileHandler(appDataDir, conv, library),
		Auth:      handlers.NewAuthHandler(settingsStore),
		Settings:  handlers.NewSettingsHandler(settingsStore, songStore, deletedLogStore, library),
		Snapshot:  handlers.NewSnapshotHandler(snapshotStore),
		Update:    handlers.NewUpdateHandler(settingsStore),
		UpdateLog: handlers.NewUpdateLogHandler(updateLogService),
		Network:   networkInfo,
		System:    handlers.NewSystemHandler(appDataDir, version.Version),
		Decrypt:   handlers.NewDecryptHandler(appDataDir, conv),
		Lifecycle: lifecycle,
	})

	// 10. Static file serving for frontend SPA
	// um-react (Unlock Music) 必须先注册，确保 /um-react/* 优先于下方 /* 兜底匹配
	server.RegisterUMReact(r, umReactFiles)
	server.RegisterStatic(r, embeddedFiles)

	router := server.WithUpdateAssets(r, updateManager)
	httpServer := &http.Server{
		Handler:           router,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	var once sync.Once
	return &backend{
		router:  router,
		server:  httpServer,
		network: networkInfo,
		close: func() {
			once.Do(func() {
				lifecycle.NotifyClosing()
				cancelWorker()
				<-workerDone
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				if err := httpServer.Shutdown(ctx); err != nil {
					log.Printf("HTTP shutdown: %v", err)
					_ = httpServer.Close()
				}
				library.WaitForSync()
				log.SetOutput(os.Stderr)
				_ = logFile.Close()
			})
		},
	}, nil
}
