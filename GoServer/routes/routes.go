package routes

import (
	"broadcast-tool/handlers"
	"broadcast-tool/internal/contracts"
	"broadcast-tool/internal/securelocal"

	"github.com/go-chi/chi/v5"
)

type HandlerSet struct {
	Extensions *handlers.ExtensionsHandler
	Songs      *handlers.SongHandler
	Files      *handlers.FileHandler
	Auth       *handlers.AuthHandler
	Settings   *handlers.SettingsHandler
	Snapshot   *handlers.SnapshotHandler
	Update     *handlers.UpdateHandler
	UpdateLog  *handlers.UpdateLogHandler
	Network    *handlers.NetworkHandler
	System     *handlers.SystemHandler
	Decrypt    *handlers.DecryptHandler
	Lifecycle  *handlers.LifecycleHandler
}

func RegisterRoutes(r chi.Router, hs HandlerSet) {
	r.Route("/api", func(api chi.Router) {
		registerAPI(api, hs)
		api.Route("/v1", func(v1 chi.Router) {
			registerAPI(v1, hs)
			v1.Get("/openapi.json", contracts.Serve)
			if hs.Extensions != nil {
				registerExtensions(v1, hs.Extensions)
			}
		})
	})
}

func registerAPI(r chi.Router, hs HandlerSet) {
	if hs.Lifecycle != nil {
		r.Get("/health", hs.Lifecycle.HandleHealth)
		r.Get("/lifecycle/watch", hs.Lifecycle.HandleWatch)
		r.Post("/shutdown", hs.Lifecycle.HandleShutdown)
		r.Post("/close-browser", hs.Lifecycle.HandleCloseBrowser)
	}
	r.Route("/songs", func(r chi.Router) {
		r.Get("/", hs.Songs.HandleList)
		r.Get("/{id}", hs.Songs.HandleGet)
		r.Post("/", hs.Songs.HandleCreate)
		r.Put("/{id}", hs.Songs.HandleUpdate)
		r.Delete("/{id}", hs.Songs.HandleDelete)
		r.Post("/sort", hs.Songs.HandleSort)
		r.Post("/reorder", hs.Songs.HandleReorder)
		r.Post("/import", hs.Songs.HandleImport)
		r.Get("/export", hs.Songs.HandleExport)
	})
	r.Route("/files", func(r chi.Router) {
		r.Post("/process", hs.Files.HandleProcess)
		r.Post("/stash", hs.Files.HandleStash)
		r.Post("/organize", hs.Files.HandleOrganize)
		r.Post("/select-dir", hs.Files.HandleSelectDir)
		r.Post("/export-week", hs.Files.HandleExportWeek)
		r.Get("/browse", hs.Files.HandleBrowse)
		r.Get("/stream", hs.Files.HandleStream)
		r.Get("/silent", hs.Files.HandleSilent)
		r.Post("/merge", hs.Files.HandleMerge)
		r.Delete("/source", hs.Files.HandleDeleteSource)
	})
	r.Route("/auth", func(r chi.Router) {
		r.Post("/verify", hs.Auth.HandleVerify)
	})
	r.Get("/settings", hs.Settings.HandleGet)
	r.Put("/settings", hs.Settings.HandleUpdate)
	r.Route("/snapshots", func(r chi.Router) {
		r.Get("/", hs.Snapshot.HandleList)
		r.Post("/restore", hs.Snapshot.HandleRestore)
	})
	r.Get("/check-update", hs.Update.HandleCheck)
	r.Get("/update-log/latest", hs.UpdateLog.HandleLatest)
	r.Post("/update-log/claim", hs.UpdateLog.HandleClaim)
	r.Get("/system/status", hs.System.HandleStatus)
	r.Route("/network", func(r chi.Router) {
		r.Get("/info", hs.Network.HandleGetInfo)
		r.Get("/qr", hs.Network.HandleGetQR)
	})
	r.Route("/decrypt", func(r chi.Router) {
		r.Post("/stage", hs.Decrypt.HandleStage)
		r.Get("/stage/{id}", hs.Decrypt.HandleStageMeta)
		r.Get("/stage/{id}/file", hs.Decrypt.HandleStageFile)
		r.Post("/stage/{id}/import", hs.Decrypt.HandleStageImport)
		r.Post("/stage/{id}/imported", hs.Decrypt.HandleStageImported)
		r.Delete("/stage/{id}", hs.Decrypt.HandleStageDelete)
	})
}

func registerExtensions(r chi.Router, h *handlers.ExtensionsHandler) {
	local := securelocal.Require
	r.Get("/retention", local(h.Retention))
	r.Put("/retention", local(h.SetRetention))
	r.Post("/retention/collect", local(h.Collect))
	r.Post("/retention/recoveries/{id}/restore", local(h.Restore))
	r.Get("/plugins", h.PluginState)
	r.Post("/plugins", local(h.InstallPlugin))
	r.Post("/plugins/example", local(h.InstallExample))
	r.Put("/plugins/{id}", local(h.ConfigurePlugin))
	r.Delete("/plugins/{id}", local(h.RemovePlugin))
	r.Post("/plugins/{id}/commands/{command}", local(h.CallPlugin))
	r.Get("/updates", h.UpdateStatus)
	r.Get("/updates/resources", h.UpdateResources)
	r.Post("/updates/check", local(h.CheckUpdates))
	r.Post("/updates/{id}/apply", local(h.ApplyUpdate))
	r.Post("/updates/web/{digest}/confirm", local(h.ConfirmWebUpdate))
	r.Post("/updates/web/rollback", local(h.RollbackWeb))
	r.Post("/updates/restart", local(h.RestartForUpdate))
}
