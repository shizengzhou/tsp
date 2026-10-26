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

# `exit` would close the caller's terminal when the script is piped into
# Invoke-Expression (`irm … | iex`), so it is only used when running from a file.
$script:fromFile = -not [string]::IsNullOrEmpty($MyInvocation.MyCommand.Path)

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

The install directory is added to your user PATH (and to the PATH of this
session), so tsp can be run from any new terminal.

Environment variables:
  TSP_VERSION      Same as -Version
  TSP_INSTALL_DIR  Same as -InstallDir
  TSP_BASE_URL     Override the download base URL
  TSP_API_URL      Override the GitHub API URL used to find the latest release
'@
}

if ($Help) {
    Show-Usage
    return
}

# Get-InstallStatePath returns the file that records where tsp was installed, so
# uninstall.ps1 can find a custom directory without being told.
function Get-InstallStatePath {
    $base = $env:LOCALAPPDATA
    if (-not $base) {
        $base = Join-Path $HOME 'AppData\Local'
    }
    return Join-Path $base 'tsp\install.txt'
}

function Set-RecordedInstallDir {
    param([Parameter(Mandatory = $true)][string]$Directory)

    $path = Get-InstallStatePath
    New-Item -ItemType Directory -Path (Split-Path -Parent $path) -Force | Out-Null
    [IO.File]::WriteAllText($path, $Directory)
}

# Get-MergedPath returns the PATH value with the directory appended, or $null
# when the directory is already listed.
function Get-MergedPath {
    param([string]$Current, [string]$Directory)

    $trimmed = $Directory.TrimEnd('\')
    $entries = @($Current -split ';' | Where-Object { $_.Trim() -ne '' })
    foreach ($entry in $entries) {
        if ($entry.TrimEnd('\') -ieq $trimmed) {
            return $null
        }
    }
    return (@($entries) + $Directory) -join ';'
}

# Add-ToUserPath appends the directory to the user PATH. The raw registry value
# is read and written back with its original kind, because
# [Environment]::GetEnvironmentVariable expands %USERPROFILE% and writing that
# back bakes the expanded paths in and downgrades REG_EXPAND_SZ to REG_SZ.
function Add-ToUserPath {
    param([Parameter(Mandatory = $true)][string]$Directory)

    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
    if (-not $key) {
        throw 'cannot open HKCU\Environment'
    }
    try {
        $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
        $current = ''
        if ($key.GetValueNames() -contains 'Path') {
            $kind = $key.GetValueKind('Path')
            $current = [string]$key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        }

        $updated = Get-MergedPath -Current $current -Directory $Directory
        if ($null -eq $updated) {
            return $false
        }

        $key.SetValue('Path', $updated, $kind)
        return $true
    } finally {
        $key.Close()
    }
}

# Publish-EnvironmentChange broadcasts WM_SETTINGCHANGE so already running
# processes (Explorer, terminals started from it) pick up the new PATH without
# a sign-out. Best effort: the value is persisted either way.
function Publish-EnvironmentChange {
    try {
        if (-not ('TspInstaller.NativeMethods' -as [type])) {
            Add-Type -Namespace TspInstaller -Name NativeMethods -MemberDefinition @'
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam, uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
'@
        }
        $result = [UIntPtr]::Zero
        [void][TspInstaller.NativeMethods]::SendMessageTimeout([IntPtr]0xffff, 0x1A, [UIntPtr]::Zero, 'Environment', 2, 1000, [ref]$result)
    } catch {
    }
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

    try {
        Set-RecordedInstallDir -Directory $InstallDir
    } catch {
        Write-Host "  warning: could not record the install directory: $($_.Exception.Message)"
    }

    $pathChanged = $false
    try {
        $pathChanged = Add-ToUserPath -Directory $InstallDir
    } catch {
        Write-Host "  warning: could not add $InstallDir to your user PATH: $($_.Exception.Message)"
    }

    $onPath = ($env:Path -split ';') | Where-Object { $_.TrimEnd('\') -ieq $InstallDir.TrimEnd('\') }
    if (-not $onPath) {
        # Make tsp available in this session instead of only after a restart.
        $env:Path = if ($env:Path) { "$InstallDir;$env:Path" } else { $InstallDir }
    }
    if ($pathChanged) {
        Publish-EnvironmentChange
        Write-Host "  added $InstallDir to your user PATH"
    }
    Write-Host '  run "tsp omz themes" to browse your oh-my-zsh themes'
} catch {
    [Console]::Error.WriteLine("error: $($_.Exception.Message)")
    $failed = $true
} finally {
    if (Test-Path -LiteralPath $tmp) {
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
    }
}

if ($failed -and $script:fromFile) {
    exit 1
}
