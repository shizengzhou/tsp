#Requires -Version 5.1
<#
.SYNOPSIS
  Install tsp from GitHub releases on Windows.

.EXAMPLE
  irm https://raw.githubusercontent.com/shizengzhou/tsp/main/install.ps1 | iex

.EXAMPLE
  .\install.ps1 -Version v0.1.0 -InstallDir C:\Tools

.NOTES
  tsp needs zsh and oh-my-zsh on PATH. Inside WSL, use install.sh with the
  Linux build instead.
#>
[CmdletBinding()]
param(
    [Alias('Dir')]
    [string]$InstallDir = $env:TSP_INSTALL_DIR,

    [string]$Version = $env:TSP_VERSION,

    [string]$BaseUrl = $env:TSP_BASE_URL,

    [string]$ApiUrl = $env:TSP_API_URL,

    [switch]$Help
)

$ErrorActionPreference = 'Stop'

# Windows PowerShell 5.1 may still default to TLS 1.0, which GitHub rejects.
if ($PSVersionTable.PSVersion.Major -lt 6) {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
}

function Show-Usage {
    @'
Install tsp from GitHub releases on Windows.

Usage:
  install.ps1 [-Version TAG] [-InstallDir DIR]

Options:
  -Version TAG     Release tag to install, e.g. v0.1.0 (default: latest)
  -InstallDir DIR  Where to put tsp.exe (default: %USERPROFILE%\.local\bin)
  -Help            Show this help

Environment variables:
  TSP_VERSION      Same as -Version
  TSP_INSTALL_DIR  Same as -InstallDir
  TSP_BASE_URL     Override the download base URL
  TSP_API_URL      Override the GitHub API URL used to find the latest release
'@
}

if ($Help) {
    Show-Usage
    exit 0
}

$failed = $false
$tmp = Join-Path ([IO.Path]::GetTempPath()) ('tsp-install-' + [guid]::NewGuid().ToString('N'))

try {
    if (-not $BaseUrl) {
        $BaseUrl = 'https://github.com/shizengzhou/tsp'
    }
    $BaseUrl = $BaseUrl.TrimEnd('/')

    if (-not $ApiUrl) {
        $ApiUrl = 'https://api.github.com/repos/shizengzhou/tsp/releases/latest'
    }

    if (-not $InstallDir) {
        $InstallDir = Join-Path $HOME '.local\bin'
    }

    $osName = if ($env:TSP_OS) {
        $env:TSP_OS
    } elseif ($env:OS -eq 'Windows_NT') {
        'windows'
    } else {
        throw 'this installer targets Windows; use install.sh on macOS/Linux'
    }

    $procArch = if ($env:PROCESSOR_ARCHITEW6432) {
        $env:PROCESSOR_ARCHITEW6432
    } else {
        $env:PROCESSOR_ARCHITECTURE
    }
    $arch = switch ($procArch) {
        'AMD64' { 'amd64' }
        'ARM64' { 'arm64' }
        default { throw "unsupported architecture: $procArch" }
    }

    if (-not $Version) {
        Write-Host '  resolving the latest release...'
        $latest = Invoke-RestMethod -Uri $ApiUrl -Headers @{ 'User-Agent' = 'tsp-installer' } -UseBasicParsing
        $Version = $latest.tag_name
        if (-not $Version) {
            throw 'no releases found; pass -Version'
        }
    }

    if ($Version -notmatch '^v') {
        $Version = "v$Version"
    }
    $num = $Version.Substring(1)

    $archive = "tsp_${num}_${osName}_${arch}.zip"
    $archivePath = Join-Path $tmp $archive
    $downloadUrl = "$BaseUrl/releases/download/$Version/$archive"

    New-Item -ItemType Directory -Path $tmp -Force | Out-Null
    Write-Host "  downloading $archive ($osName/$arch)"
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing

    $checksumsPath = Join-Path $tmp 'checksums.txt'
    $haveChecksums = $true
    try {
        Invoke-WebRequest -Uri "$BaseUrl/releases/download/$Version/checksums.txt" -OutFile $checksumsPath -UseBasicParsing
    } catch {
        $haveChecksums = $false
        Write-Host '  warning: checksums.txt unavailable, skipping verification'
    }

    if ($haveChecksums) {
        $entry = Get-Content -LiteralPath $checksumsPath |
            Where-Object { (($_ -split '\s+')[-1]) -eq $archive } |
            Select-Object -First 1
        if ($entry) {
            $expected = ($entry -split '\s+')[0]
            $actual = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash
            if ($actual.ToLowerInvariant() -ne $expected.ToLowerInvariant()) {
                throw "checksum mismatch for $archive"
            }
            Write-Host '  checksum verified'
        } else {
            Write-Host "  warning: $archive is not listed in checksums.txt"
        }
    }

    $extractDir = Join-Path $tmp 'extract'
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force
    $binary = Join-Path $extractDir 'tsp.exe'
    if (-not (Test-Path -LiteralPath $binary)) {
        throw 'archive does not contain tsp.exe'
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $target = Join-Path $InstallDir 'tsp.exe'
    $staged = Join-Path $InstallDir ('.tsp.' + [guid]::NewGuid().ToString('N') + '.tmp.exe')
    Copy-Item -LiteralPath $binary -Destination $staged -Force
    Move-Item -LiteralPath $staged -Destination $target -Force

    Write-Host "  installed tsp $num to $target"

    $onPath = ($env:Path -split ';') | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') }
    if (-not $onPath) {
        Write-Host ''
        Write-Host "$InstallDir is not on your PATH. Add it for this session with:"
        Write-Host "  `$env:Path += `";$InstallDir`""
        Write-Host 'Persist it by adding the directory to your user PATH environment variable.'
    }

    Write-Host ''
    Write-Host 'Run "tsp omz themes" to browse your oh-my-zsh themes.'
} catch {
    [Console]::Error.WriteLine("error: $($_.Exception.Message)")
    $failed = $true
} finally {
    if (Test-Path -LiteralPath $tmp) {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

if ($failed) {
    exit 1
}
