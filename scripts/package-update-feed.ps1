$ErrorActionPreference = 'Stop'
$repo = Split-Path $PSScriptRoot -Parent
$version = (Get-Content (Join-Path $repo 'package.json') | ConvertFrom-Json).version
$dist = Join-Path $repo 'dist'
$staging = Join-Path ([System.IO.Path]::GetTempPath()) ('bctools-feed-' + [guid]::NewGuid())
try {
  $web = Join-Path $staging 'web'
  $targets = Join-Path $staging 'targets'
  New-Item -ItemType Directory -Path $web, $targets | Out-Null
  Copy-Item -Recurse (Join-Path $repo 'GoServer/embed/dist') (Join-Path $web 'main')
  Copy-Item -Recurse (Join-Path $repo 'GoServer/embed/um-react') (Join-Path $web 'um-react')
  $webName = "bctools-web-$version.zip"
  $appName = "bctools-$version.exe"
  $setupName = "bctools-setup-$version.exe"
  Compress-Archive -Path (Join-Path $web '*') -DestinationPath (Join-Path $targets $webName)
  Copy-Item (Join-Path $dist 'bctools.exe') (Join-Path $targets $appName)
  Copy-Item (Join-Path $dist 'BCTools-Setup.exe') (Join-Path $targets $setupName)
  @{
    schema = 1
    items = @(
      @{ id = 'web'; kind = 'web'; version = $version; target = $webName; apiVersion = 1; description = 'BCTools Web and Unlock Music' }
      @{ id = 'core'; kind = 'core'; version = $version; target = $appName; installerTarget = $setupName; platform = 'windows'; arch = 'amd64'; storageVersion = 2; apiVersion = 1; description = 'BCTools for Windows x64' }
    )
  } | ConvertTo-Json -Depth 6 | Set-Content -Encoding utf8NoBOM (Join-Path $targets 'catalog.json')
  Compress-Archive -Path $targets -DestinationPath (Join-Path $dist 'BCTools-UpdateTargets.zip') -Force
} finally {
  Remove-Item -LiteralPath $staging -Recurse -Force -ErrorAction SilentlyContinue
}
