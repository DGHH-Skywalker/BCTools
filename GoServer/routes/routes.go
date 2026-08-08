package routes

import (
	"broadcast-tool/handlers"

	"github.com/go-chi/chi/v5"
)

type HandlerSet struct {
	Songs    *handlers.SongHandler
	Files    *handlers.FileHandler
	Auth     *handlers.AuthHandler
	Settings *handlers.SettingsHandler
	Snapshot *handlers.SnapshotHandler
	Update   *handlers.UpdateHandler
	Network  *handlers.NetworkHandler
	System   *handlers.SystemHandler
	Decrypt  *handlers.DecryptHandler
}

func RegisterRoutes(r chi.Router, hs HandlerSet) {
	r.Route("/api", func(r chi.Router) {
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
			r.Get("/browse", hs.Files.HandleBrowse)
			r.Get("/stream", hs.Files.HandleStream)
			r.Get("/silent", hs.Files.HandleSilent)
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
		r.Get("/system/status", hs.System.HandleStatus)
		r.Route("/network", func(r chi.Router) {
			r.Get("/info", hs.Network.HandleGetInfo)
			r.Get("/qr", hs.Network.HandleGetQR)
		})
		r.Route("/decrypt", func(r chi.Router) {
			r.Post("/stage", hs.Decrypt.HandleStage)
			r.Get("/stage/{id}", hs.Decrypt.HandleStageMeta)
			r.Get("/stage/{id}/file", hs.Decrypt.HandleStageFile)
			r.Post("/stage/{id}/imported", hs.Decrypt.HandleStageImported)
			r.Delete("/stage/{id}", hs.Decrypt.HandleStageDelete)
		})
	})
}
