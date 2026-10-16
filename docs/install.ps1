# marz one-line installer for Windows (PowerShell 5.1+ or 7):
#   irm https://mrzroot.github.io/marz/install.ps1 | iex
# Downloads marz.exe from the latest GitHub release, verifies its SHA-256, installs it to
# %LOCALAPPDATA%\marz\bin and adds that folder to your user PATH. No admin rights needed.
# Options (environment): MARZ_VERSION=v0.1.0, MARZ_BIN_DIR, MARZ_BASE_URL (mirror of the release assets)
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

$arch = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64' -or $env:PROCESSOR_ARCHITEW6432 -eq 'ARM64') { 'arm64' } else { 'amd64' }
$asset = "marz_windows_$arch.exe"
if ($env:MARZ_BASE_URL) { $base = $env:MARZ_BASE_URL }
elseif ($env:MARZ_VERSION) { $base = "https://github.com/mrzroot/marz/releases/download/$($env:MARZ_VERSION)" }
else { $base = 'https://github.com/mrzroot/marz/releases/latest/download' }
$binDir = if ($env:MARZ_BIN_DIR) { $env:MARZ_BIN_DIR } else { Join-Path $env:LOCALAPPDATA 'marz\bin' }
New-Item -ItemType Directory -Force -Path $binDir | Out-Null
$tmp = Join-Path ([IO.Path]::GetTempPath()) ("marz-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

function Fetch($url, $out) {
  for ($i = 1; $i -le 15; $i++) {
    try {
      if ($url -match '^[A-Za-z]:\\|^\\\\|^file:') { Copy-Item ($url -replace '^file:///', '') $out -Force; return }
      Invoke-WebRequest -UseBasicParsing -Uri $url -OutFile $out -TimeoutSec 120
      return
    } catch {
      if ($i -eq 15) { throw }
      Write-Host "  connection dropped; retrying ($i)..." -ForegroundColor Yellow
      Start-Sleep -Seconds 2
    }
  }
}
function Join-Url($b, $n) { if ($b -match '^[A-Za-z]:\\|^\\\\') { return (Join-Path $b $n) } else { return "$b/$n" } }

try {
  Write-Host "==> Downloading $asset" -ForegroundColor Green
  $exe = Join-Path $tmp 'marz.exe'
  Fetch (Join-Url $base $asset) $exe
  $sums = Join-Path $tmp 'SHA256SUMS'
  Fetch (Join-Url $base 'SHA256SUMS') $sums
  $line = Get-Content $sums | Where-Object { $_ -match (" " + [regex]::Escape($asset) + '$') } | Select-Object -First 1
  if (-not $line) { throw "$asset is not listed in SHA256SUMS" }
  $want = ($line -split '\s+')[0].ToLower()
  $got = (Get-FileHash -Algorithm SHA256 $exe).Hash.ToLower()
  if ($want -ne $got) { throw "checksum mismatch (expected $want, got $got)" }
  Write-Host '  checksum ok'
  $dest = Join-Path $binDir 'marz.exe'
  Copy-Item $exe $dest -Force
  $ver = & $dest version
  Write-Host "==> Installed marz $ver to $dest" -ForegroundColor Green
  $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
  if (-not (($userPath -split ';') -contains $binDir)) {
    $newPath = if ($userPath) { "$userPath;$binDir" } else { $binDir }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "  added $binDir to your user PATH; open a new terminal to use 'marz' everywhere" -ForegroundColor Yellow
  }
  $env:Path = "$binDir;$env:Path"
  Write-Host ''
  Write-Host 'Try:  cd your-project; marz'
  Write-Host '      marz probe fonts.googleapis.com api.openai.com'
} finally {
  Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
