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

# The browser backend listens on IPv4. Probe it without the runner's proxy.
$baseURL = "http://127.0.0.1:1743"
$stdout = Join-Path $env:RUNNER_TEMP "bctools-stdout.log"
$stderr = Join-Path $env:RUNNER_TEMP "bctools-stderr.log"
$appLog = Join-Path $installDir "BctoolData\logs\app.log"
$backend = Start-Process -FilePath $app -ArgumentList "--dev" -WorkingDirectory $installDir `
  -PassThru -WindowStyle Hidden -RedirectStandardOutput $stdout -RedirectStandardError $stderr
try {
  $ready = $false
  $deadline = [DateTime]::UtcNow.AddSeconds(60)
  $lastError = ""
  while ([DateTime]::UtcNow -lt $deadline) {
    if ($backend.HasExited) { throw "Installed backend exited before becoming ready ($($backend.ExitCode))" }
    try {
      $response = Invoke-WebRequest -Uri "$baseURL/api/health" -NoProxy -ConnectionTimeoutSeconds 2 -OperationTimeoutSeconds 5
      $ready = $response.StatusCode -eq 200
      if ($ready) { break }
    } catch {
      $lastError = $_.Exception.Message
      Start-Sleep -Milliseconds 250
    }
  }
  if (-not $ready) { throw "Installed backend health check timed out. Last response: $lastError" }
  foreach ($path in @("/", "/um-react/", "/api/v1/openapi.json")) {
    $response = Invoke-WebRequest -Uri "$baseURL$path" -NoProxy -ConnectionTimeoutSeconds 5 -OperationTimeoutSeconds 5
    if ($response.StatusCode -ne 200) { throw "Packaged endpoint $path failed" }
  }
  Run-Checked $app @("--shutdown")
  if (-not $backend.WaitForExit(30000) -or $backend.ExitCode -ne 0) {
    throw "Installed backend did not shut down cleanly"
  }
} catch {
  foreach ($log in @($stdout, $stderr, $appLog)) {
    if (Test-Path -LiteralPath $log) {
      Write-Host "Backend log $log"
      Get-Content -LiteralPath $log -Tail 100 | Write-Host
    }
  }
  throw
} finally {
  if (-not $backend.HasExited) { Stop-Process -Id $backend.Id -Force }
}

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
