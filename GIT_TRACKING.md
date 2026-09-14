# Git File Tracking

This repository stores the files needed to reproduce the application from source.

## Track in Git

- Application source: `GoServer/`, `Web/src/`, `Web/um-react/src/`, and `installer/src/`
- Required static assets, test fixtures, scripts, documentation, and CI workflows
- Dependency manifests and lock files: `package.json`, `package-lock.json`, `go.mod`, `go.sum`, and `pnpm-lock.yaml`
- Shareable editor settings such as `Web/um-react/.vscode/extensions.json`
- Environment variable templates such as `.env.example`, with no credentials in them

## Do Not Track

- Installed packages: every `node_modules/` directory and local package caches
- Generated files: `dist/`, embedded frontend bundles, installer objects, executables, archives, coverage, and test reports
- Local runtime data: `GoServer/BctoolData/`, logs, temporary files, and profiling output
- Private machine configuration and credentials: `.env`, `.env.*`, certificate/private-key files, and local IDE settings

Before committing, inspect `git status` and keep generated output and credentials out of the staged changes.
