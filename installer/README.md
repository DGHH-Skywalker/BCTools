# BCTools Inno Setup installer

The default installer is defined in `BCTools.iss`. The former C++/Win32 source
under `src/` and its `Makefile` are retained temporarily for comparison, but are
not used by the root build or CI.

## Layout and data

- Current-user application: `%LOCALAPPDATA%\Programs\BCTools\bctools.exe`.
- All-users application: `%ProgramFiles%\BCTools\bctools.exe`.
- User data: `<install directory>\BctoolData`.
- The Go executable embeds both web frontends and ffmpeg/ffprobe. At runtime it
  extracts the media tools under `BctoolData\bin`.

Inno Setup owns only application files and shortcuts. It does not register
`BctoolData` as an installed file, so upgrades and ordinary uninstalls preserve
the directory. Interactive uninstall asks whether to remove it. Silent uninstall
preserves it unless `/REMOVEUSERDATA` is supplied explicitly.

## Installation modes

The stable Inno Setup `AppId` is the installation identity. When it is absent,
Setup performs a fresh install. When it exists, Setup reuses the recorded path
and performs an upgrade or reinstall; running the same package is the supported
repair/reinstall flow. Stable Desktop and Start Menu shortcut names are replaced
in place, preventing duplicates.

Before files are replaced, Setup checks for `bctools.exe` and calls the existing
`POST http://127.0.0.1:1743/api/shutdown` endpoint. If the process does not exit,
interactive setup asks the user to close it and silent setup aborts without
replacing files.

## Build

Install Inno Setup 6 and either use its default install path or set `ISCC_PATH`:

```powershell
npm run build
```

The result is `dist\BCTools-Setup.exe`.

## Silent commands

```powershell
& .\BCTools-Setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CURRENTUSER
& "$env:LOCALAPPDATA\Programs\BCTools\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
& "$env:LOCALAPPDATA\Programs\BCTools\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /REMOVEUSERDATA
```

Use `/DIR="D:\Path With Spaces\BCTools"` to choose an install directory during
silent installation. Use `/ALLUSERS` for a machine-wide installation and
`/CURRENTUSER` for a per-user installation. Inno Setup uses Unicode paths and
supports Windows 10/11.
