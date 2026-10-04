// Package version provides a single source of truth for the application version.
// Update this value when bumping Web/package.json and GoServer/versioninfo.json.
package version

// Variable permits isolated cross-version integration fixtures via ldflags.
var Version = "5.9.0.0"
