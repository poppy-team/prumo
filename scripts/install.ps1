# Prumo Windows PowerShell Installer
# Usage: powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.ps1 | iex"

[CmdletBinding()]
param(
    [string]$Repository = $(if ($env:PRUMO_REPOSITORY) { $env:PRUMO_REPOSITORY } else { "raillen/prumo" }),
    [string]$Version = $(if ($env:PRUMO_VERSION) { $env:PRUMO_VERSION } else { "v0.6.0" }),
    [string]$InstallDir = $(if ($env:PRUMO_INSTALL_DIR) { $env:PRUMO_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\prumo" })
)

$ErrorActionPreference = "Stop"

Write-Host "Installing Prumo $Version for Windows..." -ForegroundColor Cyan

# Detect architecture
$arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
    $arch = "arm64"
} elseif ($env:PROCESSOR_ARCHITECTURE -ne "AMD64" -and $env:PROCESSOR_ARCHITEW6432 -ne "AMD64") {
    Write-Error "Unsupported processor architecture: $env:PROCESSOR_ARCHITECTURE"
    exit 1
}

$asset = "prumo-windows-$arch.exe"
$baseUrl = "https://github.com/$Repository/releases/download/$Version"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    $checksumUrl = "$baseUrl/checksums.txt"
    $assetUrl = "$baseUrl/$asset"

    Write-Host "Downloading checksums from $checksumUrl..."
    Invoke-WebRequest -Uri $checksumUrl -OutFile (Join-Path $tempDir "checksums.txt") -UseBasicParsing

    Write-Host "Downloading $asset from $assetUrl..."
    Invoke-WebRequest -Uri $assetUrl -OutFile (Join-Path $tempDir $asset) -UseBasicParsing

    # Verify SHA-256
    $checksumContent = Get-Content (Join-Path $tempDir "checksums.txt")
    $expectedHash = ""
    foreach ($line in $checksumContent) {
        $parts = -split $line
        if ($parts.Length -ge 2 -and $parts[1] -eq $asset) {
            $expectedHash = $parts[0].Trim().ToLower()
            break
        }
    }

    if (-not $expectedHash) {
        Write-Error "No checksum found for $asset in checksums.txt"
        exit 1
    }

    $actualHash = (Get-FileHash -Path (Join-Path $tempDir $asset) -Algorithm SHA256).Hash.ToLower()
    if ($expectedHash -ne $actualHash) {
        Write-Error "Checksum mismatch for $asset. Expected: $expectedHash, Got: $actualHash"
        exit 1
    }
    Write-Host "Checksum verified successfully ($actualHash)." -ForegroundColor Green

    # Install executable
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $targetExe = Join-Path $InstallDir "prumo.exe"
    Copy-Item -Path (Join-Path $tempDir $asset) -Destination $targetExe -Force
    Write-Host "Installed executable to $targetExe" -ForegroundColor Green

    # Run setup
    & $targetExe setup | Out-Null

    # Configure User PATH environment variable permanently
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathParts = $userPath -split ";" | Where-Object { $_ -ne "" }

    if ($pathParts -notcontains $InstallDir) {
        $newUserPath = if ($userPath) { "$userPath;$InstallDir" } else { $InstallDir }
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        Write-Host "Added $InstallDir to user PATH environment variable." -ForegroundColor Green
    }

    # Update active session PATH
    if (($env:Path -split ";") -notcontains $InstallDir) {
        $env:Path = "$env:Path;$InstallDir"
    }

    Write-Host ""
    Write-Host "Prumo $Version installed successfully!" -ForegroundColor Cyan
    Write-Host "Run: prumo version" -ForegroundColor Yellow
}
finally {
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
