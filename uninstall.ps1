#Requires -Version 5.1
<#
.SYNOPSIS
  Uninstall tsp on Windows.

.EXAMPLE
  irm https://raw.githubusercontent.com/shizengzhou/tsp/main/uninstall.ps1 | iex

.EXAMPLE
  .\uninstall.ps1 -InstallDir C:\Tools

.NOTES
  Only the binary and, when it becomes empty, the install directory are
  removed. Configuration that tsp wrote (zsh and PowerShell profiles,
  alacritty.toml imports, downloaded themes) is left untouched.
#>
[CmdletBinding()]
param(
    [Alias('Dir')]
    [string]$InstallDir = $env:TSP_INSTALL_DIR,

    [switch]$Help
)

$ErrorActionPreference = 'Stop'

# `exit` would close the caller's terminal when the script is piped into
# Invoke-Expression (`irm … | iex`), so it is only used when running from a file.
$script:fromFile = -not [string]::IsNullOrEmpty($MyInvocation.MyCommand.Path)

function Show-Usage {
    @'
Uninstall tsp on Windows.

Usage:
  uninstall.ps1 [-InstallDir DIR]

Options:
  -InstallDir DIR  Where tsp.exe was installed (default: the directory recorded
                   by install.ps1, or %USERPROFILE%\.local\bin)
  -Help            Show this help

Environment variables:
  TSP_INSTALL_DIR  Same as -InstallDir

The install directory is removed from your user PATH only when it is empty
after tsp.exe is deleted, so other tools installed next to it keep working.
'@
}

if ($Help) {
    Show-Usage
    return
}

# Get-InstallStatePath returns the file install.ps1 uses to record where tsp was
# installed, so a custom directory can be found without -InstallDir.
function Get-InstallStatePath {
    $base = $env:LOCALAPPDATA
    if (-not $base) {
        $base = Join-Path $HOME 'AppData\Local'
    }
    return Join-Path $base 'tsp\install.txt'
}

function Get-RecordedInstallDir {
    $path = Get-InstallStatePath
    try {
        if (-not (Test-Path -LiteralPath $path)) {
            return ''
        }
        return ([IO.File]::ReadAllText($path)).Trim()
    } catch {
        return ''
    }
}

function Remove-InstallRecord {
    $path = Get-InstallStatePath
    $dir = Split-Path -Parent $path
    try {
        if (Test-Path -LiteralPath $path) {
            Remove-Item -LiteralPath $path -Force
        }
        $left = @(Get-ChildItem -LiteralPath $dir -Force -ErrorAction SilentlyContinue)
        if ($left.Count -eq 0) {
            Remove-Item -LiteralPath $dir -Force
        }
    } catch {
    }
}

# Remove-FromUserPath drops the directory from the user PATH. The raw registry
# value is read and written back with its original kind, because
# [Environment]::GetEnvironmentVariable expands %USERPROFILE% and writing that
# back bakes the expanded paths in and downgrades REG_EXPAND_SZ to REG_SZ.
function Remove-FromUserPath {
    param([Parameter(Mandatory = $true)][string]$Directory)

    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment', $true)
    if (-not $key) {
        return $false
    }
    try {
        if ($key.GetValueNames() -notcontains 'Path') {
            return $false
        }
        $kind = $key.GetValueKind('Path')
        $current = [string]$key.GetValue('Path', '', [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        $trimmed = $Directory.TrimEnd('\')
        $entries = @($current -split ';' | Where-Object { $_.Trim() -ne '' })
        $kept = @($entries | Where-Object { $_.TrimEnd('\') -ne $trimmed })
        if ($kept.Count -eq $entries.Count) {
            return $false
        }

        $key.SetValue('Path', ($kept -join ';'), $kind)
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

try {
    $recorded = Get-RecordedInstallDir
    if (-not $InstallDir) {
        # Prefer the directory install.ps1 recorded, so a custom location is
        # found without repeating -InstallDir.
        $InstallDir = if ($recorded) { $recorded } else { Join-Path $HOME '.local\bin' }
    }
    $usesRecord = $recorded -and ($recorded.TrimEnd('\') -ieq $InstallDir.TrimEnd('\'))

    $target = Join-Path $InstallDir 'tsp.exe'
    if (-not (Test-Path -LiteralPath $target)) {
        Write-Host "  tsp is not installed in $InstallDir"
        if ($usesRecord) {
            Remove-InstallRecord
        }
        return
    }

    Remove-Item -LiteralPath $target -Force
    Write-Host "  removed $target"

    # tsp update renames the running binary to tsp.exe.old on Windows; drop the
    # leftover too, otherwise the directory never becomes empty.
    $stale = Join-Path $InstallDir 'tsp.exe.old'
    if (Test-Path -LiteralPath $stale) {
        Remove-Item -LiteralPath $stale -Force -ErrorAction SilentlyContinue
        if (-not (Test-Path -LiteralPath $stale)) {
            Write-Host "  removed $stale"
        }
    }

    # Keep the directory when other tools live in it, so uninstalling tsp never
    # breaks them or their PATH entry.
    $remaining = @(Get-ChildItem -LiteralPath $InstallDir -Force -ErrorAction SilentlyContinue)
    if ($remaining.Count -gt 0) {
        Write-Host "  kept ${InstallDir}: it still contains other files"
    } else {
        Remove-Item -LiteralPath $InstallDir -Force
        Write-Host "  removed $InstallDir"

        $trimmed = $InstallDir.TrimEnd('\')
        if ($env:Path) {
            $env:Path = (@($env:Path -split ';' | Where-Object { $_.TrimEnd('\') -ne $trimmed }) | Where-Object { $_ -ne '' }) -join ';'
        }
        if (Remove-FromUserPath -Directory $InstallDir) {
            Publish-EnvironmentChange
            Write-Host "  removed $InstallDir from your user PATH"
        }
    }

    if ($usesRecord) {
        Remove-InstallRecord
    }

    Write-Host '  profiles and configs written by tsp were left untouched'
} catch {
    [Console]::Error.WriteLine("error: $($_.Exception.Message)")
    $failed = $true
}

if ($failed -and $script:fromFile) {
    exit 1
}
