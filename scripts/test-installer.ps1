$ErrorActionPreference = "Stop"

$installDir = Join-Path $env:RUNNER_TEMP "BCTools CI 路径 With Spaces"
$setup = Join-Path $PSScriptRoot "..\dist\BCTools-Setup.exe"
$setupArgs = @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/CURRENTUSER", "/DIR=`"$installDir`"")

function Run-Checked([string] $FilePath, [string[]] $ArgumentList) {
  $process = Start-Process -FilePath $FilePath -ArgumentList $ArgumentList -Wait -PassThru -WindowStyle Hidden
  if ($process.ExitCode -ne 0) { throw "$FilePath exited with $($process.ExitCode)" }
}

Run-Checked $setup $setupArgs
$app = Join-Path $installDir "bctools.exe"
$uninstaller = Join-Path $installDir "unins000.exe"
if (-not (Test-Path -LiteralPath $app)) { throw "Installed executable is missing" }
if (-not (Test-Path -LiteralPath $uninstaller)) { throw "Uninstaller is missing" }

# Running the same package exercises repair/reinstall and stable shortcut paths.
Run-Checked $setup $setupArgs
$desktopShortcut = Join-Path ([Environment]::GetFolderPath("Desktop")) "小播点歌工具.lnk"
$startShortcut = Join-Path ([Environment]::GetFolderPath("Programs")) "BCTools\小播点歌工具.lnk"
if (-not (Test-Path -LiteralPath $desktopShortcut)) { throw "Desktop shortcut is missing" }
if (-not (Test-Path -LiteralPath $startShortcut)) { throw "Start Menu shortcut is missing" }

$dataDir = Join-Path $installDir "BctoolData"
New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
$sentinel = Join-Path $dataDir "ci-preserve.txt"
[IO.File]::WriteAllText($sentinel, "preserve")
Run-Checked $uninstaller @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART")
if (-not (Test-Path -LiteralPath $sentinel)) { throw "Default uninstall removed user data" }

Run-Checked $setup $setupArgs
$uninstaller = Join-Path $installDir "unins000.exe"
Run-Checked $uninstaller @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/REMOVEUSERDATA")
if (Test-Path -LiteralPath $dataDir) { throw "Explicit user-data removal failed" }
