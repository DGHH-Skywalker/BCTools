$ErrorActionPreference = "Stop"
$repoRoot = Split-Path $PSScriptRoot -Parent
$testRoot = Join-Path $env:RUNNER_TEMP ("bctools-startup-" + [guid]::NewGuid())
if (-not $env:RUNNER_TEMP) { $testRoot = Join-Path ([IO.Path]::GetTempPath()) ("bctools-startup-" + [guid]::NewGuid()) }
$port = 1753
$base = "http://127.0.0.1:$port/api"
$envNames = @("LOCALAPPDATA", "APPDATA", "USERPROFILE")
$oldEnv = @{}
foreach ($name in $envNames) { $oldEnv[$name] = [Environment]::GetEnvironmentVariable($name) }
$versionInfoPath = Join-Path $repoRoot "GoServer/versioninfo.json"
$originalVersionInfo = [IO.File]::ReadAllText($versionInfoPath)
$started = @()
$runtimePID = $null

function Call-Api([string] $Method, [string] $Path, $Body = $null, [string] $Instance = "") {
  $headers = @{}
  if ($Instance) { $headers["X-BCTools-Instance"] = $Instance }
  $requestOptions = @{ Method = $Method; Uri = "$base$Path"; TimeoutSec = 8; Headers = $headers }
  if ($null -ne $Body) { $requestOptions.Body = ConvertTo-Json -InputObject $Body -Depth 20 -Compress; $requestOptions.ContentType = "application/json; charset=utf-8" }
  return Invoke-RestMethod @requestOptions
}
function Wait-Version([string] $Version) {
  $deadline = [DateTime]::UtcNow.AddSeconds(60)
  while ([DateTime]::UtcNow -lt $deadline) {
    try { $health = Call-Api GET "/health"; if ($health.version -eq $Version) { return $health } } catch {}
    Start-Sleep -Milliseconds 150
  }
  throw "Backend $Version did not become ready"
}
function Assert([bool] $Condition, [string] $Message) { if (-not $Condition) { throw $Message } }

try {
  New-Item -ItemType Directory -Path $testRoot -Force | Out-Null
  $env:LOCALAPPDATA = Join-Path $testRoot "Local"
  $env:APPDATA = Join-Path $testRoot "Roaming"
  $env:USERPROFILE = Join-Path $testRoot "User"
  foreach ($path in @($env:LOCALAPPDATA, $env:APPDATA, $env:USERPROFILE)) { New-Item -ItemType Directory -Path $path -Force | Out-Null }

  # This older VERSION fixture uses the new protocol; it does not pretend that
  # an unmodified historical 5.8 executable supports browser-save handoff.
  $olderVersion = "5.8.99.0"
  $info = $originalVersionInfo | ConvertFrom-Json
  foreach ($field in @("FileVersion", "ProductVersion")) {
    $info.FixedFileInfo.$field.Major = 5; $info.FixedFileInfo.$field.Minor = 8
    $info.FixedFileInfo.$field.Patch = 99; $info.FixedFileInfo.$field.Build = 0
    $info.StringFileInfo.$field = $olderVersion
  }
  [IO.File]::WriteAllText($versionInfoPath, (ConvertTo-Json $info -Depth 20))
  & go -C (Join-Path $repoRoot "GoServer") run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 -o resource.syso versioninfo.json
  Assert ($LASTEXITCODE -eq 0) "Fixture resource generation failed"
  $olderDir = Join-Path $testRoot "Original"
  New-Item -ItemType Directory -Path $olderDir -Force | Out-Null
  $olderExe = Join-Path $olderDir "bctools.exe"
  & go -C (Join-Path $repoRoot "GoServer") build "-ldflags=-s -w -H windowsgui -X broadcast-tool/internal/version.Version=$olderVersion" -o $olderExe .
  Assert ($LASTEXITCODE -eq 0) "Fixture build failed"
  [IO.File]::WriteAllText($versionInfoPath, $originalVersionInfo)
  & go -C (Join-Path $repoRoot "GoServer") run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@v1.7.0 -o resource.syso versioninfo.json
  Assert ($LASTEXITCODE -eq 0) "Resource restoration failed"

  $legacy = Join-Path $olderDir "BctoolData"
  New-Item -ItemType Directory -Path $legacy -Force | Out-Null
  $legacyData = @{
    dormSongs = @(); broadcastSongs = @(@{ id = 1; date = "2026-10-04"; title = "继承的本周播音歌单"; period = "noon"; createdAt = "2026-10-04T01:00:00Z"; filePath = ""; timeSlotId = $null; artist = ""; remark = "" })
    settings = @{ timeSlots = @(@{ id = "custom-morning"; dayIndex = 1; order = 1; time = "06:10" }); locale = "zh-CN" }
  }
  [IO.File]::WriteAllText((Join-Path $legacy "data.json"), (ConvertTo-Json $legacyData -Depth 20))
  $legacyBefore = [IO.File]::ReadAllBytes((Join-Path $legacy "data.json"))
  $started += Start-Process $olderExe -ArgumentList @("--no-browser", "--port", "$port") -PassThru
  $oldHealth = Wait-Version $olderVersion
  $runtimePID = $oldHealth.pid
  $fixed = Join-Path $env:LOCALAPPDATA "BCTools/Data"
  Assert (Test-Path (Join-Path $fixed "data.json")) "Fixed database was not created"
  $songs = @(Call-Api GET "/songs?type=broadcast")
  Assert ($songs.Count -eq 1 -and $songs[0].title -eq "继承的本周播音歌单") "Broadcast migration lost a record"
  $settings = Call-Api GET "/settings"
  Assert ($settings.timeSlots.Count -eq 1 -and $settings.timeSlots[0].time -eq "06:10") "Custom times were overwritten"

  # Two launchers from distinct folders must use the same running instance.
  $copyDir = Join-Path $testRoot "Other Folder"
  New-Item -ItemType Directory -Path $copyDir -Force | Out-Null
  $copyExe = Join-Path $copyDir "bctools.exe"; Copy-Item $olderExe $copyExe
  $duplicates = @(
    (Start-Process $olderExe -ArgumentList @("--no-browser", "--port", "$port") -PassThru),
    (Start-Process $copyExe -ArgumentList @("--no-browser", "--port", "$port") -PassThru)
  )
  $started += $duplicates
  foreach ($process in $duplicates) { Assert ($process.WaitForExit(15000)) "Duplicate launcher did not exit after reusing the running instance" }
  Assert ((Call-Api GET "/health").pid -eq $oldHealth.pid) "Duplicate launch replaced same-version instance"
  Assert (-not (Test-Path (Join-Path $copyDir "BctoolData"))) "EXE-relative data directory was recreated"

  # Register a tab with an unfinished edit. Shutdown cannot proceed until its
  # title PUT completes and it acknowledges the exact prepare token.
  $client = "ci-tab-" + [guid]::NewGuid()
  Call-Api POST "/lifecycle/client" @{ id = $client; dirty = $true; version = $olderVersion } | Out-Null
  $downloadDir = Join-Path $env:USERPROFILE "Downloads/Latest"
  New-Item -ItemType Directory -Path $downloadDir -Force | Out-Null
  Copy-Item (Join-Path $repoRoot "dist/bctools.exe") (Join-Path $downloadDir "bctools.exe")
  # The newer EXE has NEVER run: discovery must find its product/version info.
  $started += Start-Process $copyExe -ArgumentList @("--no-browser", "--port", "$port") -PassThru
  $deadline = [DateTime]::UtcNow.AddSeconds(60); $token = ""
  while ([DateTime]::UtcNow -lt $deadline -and -not $token) {
    $state = Call-Api POST "/lifecycle/client" @{ id = $client; dirty = $true; version = $olderVersion }
    $token = $state.prepare
    if (-not $token) { Start-Sleep -Milliseconds 100 }
  }
  Assert ([bool] $token) "New-version discovery did not initiate safe handoff"
  Assert ((Call-Api GET "/health").pid -eq $oldHealth.pid) "Old instance exited before its tab saved"
  Call-Api PUT "/songs/1" @{ title = "刚输入就切换的本周歌单"; expectedTitle = "继承的本周播音歌单" } $oldHealth.instanceId | Out-Null
  Call-Api POST "/lifecycle/client" @{ id = $client; ack = $token; dirty = $false; version = $olderVersion } | Out-Null
  $newHealth = Wait-Version "5.9.0.0"; $runtimePID = $newHealth.pid
  Assert ($newHealth.pid -ne $oldHealth.pid) "New version reused old PID"
  Assert ($newHealth.databaseId -eq $oldHealth.databaseId) "Database identity changed across versions"
  $songs = @(Call-Api GET "/songs?type=broadcast")
  Assert ($songs[0].title -eq "刚输入就切换的本周歌单") "Handoff lost the last title edit"
  Assert ($null -eq (Get-Process -Id $oldHealth.pid -ErrorAction SilentlyContinue)) "Original runtime process is still running"
  Assert ((Get-FileHash (Join-Path $legacy "data.json")).Hash -eq ([Convert]::ToHexString([Security.Cryptography.SHA256]::HashData($legacyBefore)))) "Migration changed the original database"

  # A stale page's old instance header must never write to the new process.
  $rejected = $false
  try { Call-Api PUT "/songs/1" @{ title = "stale overwrite" } $oldHealth.instanceId | Out-Null } catch { $rejected = $true }
  Assert $rejected "Stale UI write was accepted"
  Call-Api POST "/shutdown" | Out-Null
  $deadline = [DateTime]::UtcNow.AddSeconds(20)
  while ((Get-Process -Id $runtimePID -ErrorAction SilentlyContinue) -and [DateTime]::UtcNow -lt $deadline) { Start-Sleep -Milliseconds 100 }
  Assert ($null -eq (Get-Process -Id $runtimePID -ErrorAction SilentlyContinue)) "Safe shutdown did not terminate the process"
  $runtimePID = $null
  Write-Host "Windows startup, discovery, migration, single-instance and durable-handoff checks passed."
}
catch {
  Write-Host "Startup test diagnostics:"
  foreach ($file in @((Join-Path $env:LOCALAPPDATA "BCTools/installations.json"), (Join-Path $env:LOCALAPPDATA "BCTools/Data/runtime.json"), (Join-Path $env:LOCALAPPDATA "BCTools/Data/logs/app.log"))) {
    if (Test-Path $file) { Write-Host $file; Get-Content $file -Tail 80 | Write-Host }
  }
  throw
}
finally {
  # Test-owned processes and directories only. Never enumerate/kill by image name.
  foreach ($process in $started) { if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue } }
  if ($runtimePID) { Stop-Process -Id $runtimePID -Force -ErrorAction SilentlyContinue }
  [IO.File]::WriteAllText($versionInfoPath, $originalVersionInfo)
  foreach ($name in $envNames) { [Environment]::SetEnvironmentVariable($name, $oldEnv[$name]) }
  if (Test-Path $testRoot) { Remove-Item $testRoot -Recurse -Force -ErrorAction SilentlyContinue }
}
