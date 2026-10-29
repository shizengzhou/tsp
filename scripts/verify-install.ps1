# Verifies install.ps1 against a local fake release server: one install that
# succeeds, and the three ways checksum verification must refuse to install.
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$work = Join-Path ([IO.Path]::GetTempPath()) ('tsp-verify-' + [guid]::NewGuid().ToString('N'))
$version = '0.0.1'
$serverJob = $null
$oldLocalAppData = $env:LOCALAPPDATA

# Start-FakeReleaseServer serves the files under Root over HTTP, so install.ps1
# can download from a local URL instead of GitHub. It runs in a background job
# because the listener blocks while waiting for requests.
function Start-FakeReleaseServer {
	param(
		[Parameter(Mandatory = $true)][string]$Prefix,
		[Parameter(Mandatory = $true)][string]$Root
	)

	return Start-Job -ArgumentList $Prefix, $Root -ScriptBlock {
		param([string]$Prefix, [string]$Root)

		$listener = New-Object System.Net.HttpListener
		$listener.Prefixes.Add($Prefix)
		$listener.Start()

		while ($listener.IsListening) {
			$context = $null
			try {
				$context = $listener.GetContext()
			} catch {
				break
			}
			try {
				$relative = [Uri]::UnescapeDataString($context.Request.Url.AbsolutePath.TrimStart('/'))
				$file = Join-Path $Root $relative
				if (Test-Path -LiteralPath $file -PathType Leaf) {
					$bytes = [IO.File]::ReadAllBytes($file)
					$context.Response.ContentLength64 = $bytes.Length
					$context.Response.OutputStream.Write($bytes, 0, $bytes.Length)
				} else {
					$context.Response.StatusCode = 404
				}
			} finally {
				$context.Response.OutputStream.Close()
			}
		}
	}
}

# Wait-Server blocks until the listener answers, because Start-Job returns
# before its script block starts.
function Wait-Server {
	param([Parameter(Mandatory = $true)][string]$Base)

	for ($i = 0; $i -lt 60; $i++) {
		try {
			Invoke-WebRequest -Uri "$Base/__ready" -UseBasicParsing -TimeoutSec 2 | Out-Null
			return
		} catch {
			# An HTTP error still proves the server is up; only a transport
			# failure means it is not listening yet.
			if ($_.Exception.Response) {
				return
			}
			Start-Sleep -Milliseconds 500
		}
	}
	throw 'the fake release server did not start'
}

# Invoke-Install runs install.ps1 in a child process, so its exit code and
# captured output can be checked without the script altering this session.
$shellPath = (Get-Process -Id $PID).Path

function Invoke-Install {
	param([Parameter(Mandatory = $true)][string]$Dir)

	$output = & $shellPath -NoProfile -File (Join-Path $repoRoot 'install.ps1') `
		-Version "v$version" -BaseUrl $base -InstallDir $Dir 2>&1 | Out-String
	return [pscustomobject]@{ Code = $LASTEXITCODE; Output = $output }
}

function Assert-Refusal {
	param(
		[Parameter(Mandatory = $true)][string]$Label,
		[Parameter(Mandatory = $true)][string]$Want,
		[Parameter(Mandatory = $true)][string]$Dir
	)

	$result = Invoke-Install -Dir $Dir
	if ($result.Code -eq 0) {
		throw "FAIL: $Label installed anyway"
	}
	if (-not $result.Output.Contains($Want)) {
		throw "FAIL: $Label did not mention '$Want'`n$($result.Output)"
	}
	if (Test-Path -LiteralPath (Join-Path $Dir 'tsp.exe')) {
		throw "FAIL: $Label left a binary behind"
	}
	Write-Host "ok: $Label"
}

try {
	$osName = if ($env:TSP_OS) { $env:TSP_OS } else { 'windows' }
	$archSource = if ($env:PROCESSOR_ARCHITEW6432) {
		$env:PROCESSOR_ARCHITEW6432
	} elseif ($env:PROCESSOR_ARCHITECTURE) {
		$env:PROCESSOR_ARCHITECTURE
	} else {
		'AMD64'
	}
	$arch = if ($archSource -eq 'ARM64') { 'arm64' } else { 'amd64' }

	New-Item -ItemType Directory -Path $work -Force | Out-Null
	$env:LOCALAPPDATA = Join-Path $work 'localappdata'
	New-Item -ItemType Directory -Path $env:LOCALAPPDATA -Force | Out-Null

	$serverRoot = Join-Path $work 'server'
	$releaseDir = Join-Path (Join-Path $serverRoot 'releases/download') "v$version"
	New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null

	$archiveName = "tsp_${version}_${osName}_${arch}.zip"
	$archivePath = Join-Path $releaseDir $archiveName
	$payload = Join-Path $work 'tsp.exe'
	[IO.File]::WriteAllText($payload, 'fake tsp.exe')
	Compress-Archive -LiteralPath $payload -DestinationPath $archivePath -Force

	$sum = (Get-FileHash -Algorithm SHA256 -LiteralPath $archivePath).Hash.ToLowerInvariant()
	$checksumsPath = Join-Path $releaseDir 'checksums.txt'
	Set-Content -LiteralPath $checksumsPath -Value "$sum  $archiveName" -Encoding ASCII

	$listener = New-Object System.Net.Sockets.TcpListener([System.Net.IPAddress]::Loopback, 0)
	$listener.Start()
	$port = $listener.LocalEndpoint.Port
	$listener.Stop()

	$base = "http://127.0.0.1:$port"
	$serverJob = Start-FakeReleaseServer -Prefix "$base/" -Root $serverRoot
	Wait-Server -Base $base

	Write-Host '=== checksums present ==='
	$ok = Invoke-Install -Dir (Join-Path $work 'bin-ok')
	if ($ok.Code -ne 0) {
		throw "FAIL: a verified install failed`n$($ok.Output)"
	}
	if (-not $ok.Output.Contains('checksum verified')) {
		throw "FAIL: the install did not report a verified checksum`n$($ok.Output)"
	}
	if (-not (Test-Path -LiteralPath (Join-Path $work 'bin-ok/tsp.exe'))) {
		throw 'FAIL: tsp.exe was not installed'
	}
	Write-Host 'ok: installed'

	Write-Host '=== checksums missing ==='
	Remove-Item -LiteralPath $checksumsPath
	Assert-Refusal -Label 'missing checksums' -Want 'checksums.txt' -Dir (Join-Path $work 'bin-missing')

	Write-Host '=== archive not listed ==='
	Set-Content -LiteralPath $checksumsPath -Value "$sum  other.zip" -Encoding ASCII
	Assert-Refusal -Label 'unlisted archive' -Want 'not listed' -Dir (Join-Path $work 'bin-unlisted')

	Write-Host '=== checksum mismatch ==='
	Set-Content -LiteralPath $checksumsPath -Value ("{0}  {1}" -f ('0' * 64), $archiveName) -Encoding ASCII
	Assert-Refusal -Label 'checksum mismatch' -Want 'checksum mismatch' -Dir (Join-Path $work 'bin-mismatch')

	Write-Host 'all install.ps1 checks passed'
} finally {
	if ($serverJob) {
		Stop-Job -Job $serverJob -ErrorAction SilentlyContinue
		Remove-Job -Job $serverJob -Force -ErrorAction SilentlyContinue
	}
	Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
	$env:LOCALAPPDATA = $oldLocalAppData
}
