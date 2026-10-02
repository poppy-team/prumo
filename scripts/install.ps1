# Prumo Windows PowerShell Installer
# Usage: powershell -ExecutionPolicy ByPass -c "irm https://raw.githubusercontent.com/raillen/prumo/main/scripts/install.ps1 | iex"

[CmdletBinding()]
param(
    [string]$Package = $(if ($env:PRUMO_PACKAGE) { $env:PRUMO_PACKAGE } else { "" }),
    [string]$Repository = $(if ($env:PRUMO_REPOSITORY) { $env:PRUMO_REPOSITORY } else { "raillen/prumo" }),
    [string]$Version = $(if ($env:PRUMO_VERSION) { $env:PRUMO_VERSION } else { "v0.6.1" }),
    [string]$InstallDir = $(if ($env:PRUMO_INSTALL_DIR) { $env:PRUMO_INSTALL_DIR } else { "$env:LOCALAPPDATA\Programs\prumo" })
)

$ErrorActionPreference = "Stop"

# Detect architecture
$arch = "amd64"
if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64" -or $env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
    $arch = "arm64"
} elseif ($env:PROCESSOR_ARCHITECTURE -ne "AMD64" -and $env:PROCESSOR_ARCHITEW6432 -ne "AMD64") {
    Write-Error "Unsupported processor architecture: $env:PROCESSOR_ARCHITECTURE"
    exit 1
}

if (-not $Package) {
    if ([Environment]::UserInteractive -and -not [Console]::IsInputRedirected) {
        Write-Host "=================================================" -ForegroundColor Cyan
        Write-Host "       Prumo Package Selection ($Version)" -ForegroundColor Cyan
        Write-Host "=================================================" -ForegroundColor Cyan
        Write-Host "1) Prumo CLI        (Core runtime, daemon & tools) [default]"
        Write-Host "2) Prumo IDE        (Desktop GUI pair programmer & editor)"
        Write-Host "3) Prumo Code Agent (Interactive terminal client)"
        Write-Host "4) All packages      (CLI + IDE + Code Agent)"
        $choice = Read-Host "Select package [1-4, default: 1]"
        switch ($choice) {
            "2" { $Package = "ide" }
            "3" { $Package = "tui" }
            "4" { $Package = "all" }
            default { $Package = "cli" }
        }
    } else {
        $Package = "cli"
    }
}

$installCli = $false
$installIde = $false
$installTui = $false

switch ($Package.ToLower()) {
    "cli"     { $installCli = $true }
    "harness" { $installCli = $true }
    "ide"     { $installIde = $true }
    "viewer"  { $installIde = $true }
    "tui"     { $installTui = $true }
    "all"     { $installCli = $true; $installIde = $true; $installTui = $true }
    "full"    { $installCli = $true; $installIde = $true; $installTui = $true }
    default {
        Write-Error "Unknown package: $Package. Valid options: cli, ide, tui, all."
        exit 1
    }
}

Write-Host "Installing Prumo $Version ($Package) for Windows..." -ForegroundColor Cyan

$baseUrl = "https://github.com/$Repository/releases/download/$Version"
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    $checksumUrl = "$baseUrl/checksums.txt"
    Write-Host "Downloading checksums from $checksumUrl..."
    Invoke-WebRequest -Uri $checksumUrl -OutFile (Join-Path $tempDir "checksums.txt") -UseBasicParsing
    $checksumContent = Get-Content (Join-Path $tempDir "checksums.txt")

    function Install-Artifact([string]$Asset, [string]$DestName) {
        $expectedHash = ""
        foreach ($line in $checksumContent) {
            $parts = -split $line
            if ($parts.Length -ge 2 -and $parts[1] -eq $Asset) {
                $expectedHash = $parts[0].Trim().ToLower()
                break
            }
        }

        if (-not $expectedHash) {
            Write-Error "No checksum found for $Asset in checksums.txt"
            exit 1
        }

        $assetUrl = "$baseUrl/$Asset"
        Write-Host "Downloading $Asset from $assetUrl..."
        Invoke-WebRequest -Uri $assetUrl -OutFile (Join-Path $tempDir $Asset) -UseBasicParsing

        $actualHash = (Get-FileHash -Path (Join-Path $tempDir $Asset) -Algorithm SHA256).Hash.ToLower()
        if ($expectedHash -ne $actualHash) {
            Write-Error "Checksum mismatch for $Asset. Expected: $expectedHash, Got: $actualHash"
            exit 1
        }
        Write-Host "Verified checksum for $Asset." -ForegroundColor Green

        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        $targetExe = Join-Path $InstallDir "$DestName.exe"
        Copy-Item -Path (Join-Path $tempDir $Asset) -Destination $targetExe -Force
        Write-Host "Installed $DestName to $targetExe" -ForegroundColor Green
        return $targetExe
    }

    if ($installCli) {
        $cliAsset = "prumo-harness-cli-windows-$arch.exe"
        $hasCliAsset = $false
        foreach ($line in $checksumContent) {
            if ($line -match "\s$cliAsset$") { $hasCliAsset = $true; break }
        }
        if (-not $hasCliAsset) { $cliAsset = "prumo-windows-$arch.exe" }
        $targetCli = Install-Artifact -Asset $cliAsset -DestName "prumo"
        & $targetCli setup | Out-Null
    }

    if ($installTui) {
        $tuiAsset = "prumo-agent-tui-windows-$arch.exe"
        $targetTui = Install-Artifact -Asset $tuiAsset -DestName "prumo-tui"
        Copy-Item -Path $targetTui -Destination (Join-Path $InstallDir "pa.exe") -Force
    }

    if ($installIde) {
        $ideAsset = "prumo-agent-ide-windows-$arch.exe"
        $targetIde = Install-Artifact -Asset $ideAsset -DestName "prumo-ide"
        Copy-Item -Path $targetIde -Destination (Join-Path $InstallDir "prumo-viewer.exe") -Force
    }

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
    if ($installCli) { Write-Host "  prumo version    - Check CLI version" -ForegroundColor Yellow }
    if ($installTui) { Write-Host "  prumo-tui (pa)   - Launch interactive terminal harness" -ForegroundColor Yellow }
    if ($installIde) { Write-Host "  prumo-ide        - Launch desktop Agent IDE" -ForegroundColor Yellow }
}
finally {
    Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
}
