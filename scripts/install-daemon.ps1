#Requires -Version 5.1
<#
.SYNOPSIS
    Install clankers-daemon binary from GitHub Releases.

.DESCRIPTION
    Downloads and installs the clankers-daemon binary for Windows.
    Verifies checksum before installation.

.EXAMPLE
    irm https://raw.githubusercontent.com/dxta-dev/clankers/main/scripts/install-daemon.ps1 | iex

.EXAMPLE
    $env:CLANKERS_VERSION = "v0.1.0"; irm ... | iex

.EXAMPLE
    .\install-daemon.ps1 -Version v0.1.0 -AddToPath

.NOTES
    Environment variables:
      CLANKERS_VERSION      - Version to install (default: latest)
      CLANKERS_INSTALL_DIR  - Override install directory
      GITHUB_TOKEN          - Optional, for higher API rate limits
#>

param(
    [string]$Version,
    [string]$InstallDir,
    [switch]$AddToPath
)

$ErrorActionPreference = "Stop"

$Repo = "dxta-dev/clankers"
$BinaryName = "clankers-daemon"
$Target = "windows-amd64"

function Write-Log {
    param([string]$Message)
    Write-Host "[clankers] $Message"
}

function Write-ErrorLog {
    param([string]$Message)
    Write-Host "[clankers] ERROR: $Message" -ForegroundColor Red
    exit 1
}

function Get-LatestVersion {
    $url = "https://api.github.com/repos/$Repo/releases/latest"
    
    $headers = @{}
    if ($env:GITHUB_TOKEN) {
        $headers["Authorization"] = "Bearer $env:GITHUB_TOKEN"
    }
    
    try {
        $response = Invoke-RestMethod -Uri $url -Headers $headers
        return $response.tag_name
    }
    catch {
        Write-ErrorLog "Failed to fetch latest version: $_"
    }
}

function Get-FileChecksum {
    param([string]$FilePath)
    
    $hash = Get-FileHash -Path $FilePath -Algorithm SHA256
    return $hash.Hash.ToLower()
}

function Get-TargetInstallDir {
    if ($InstallDir) {
        return $InstallDir
    }
    
    if ($env:CLANKERS_INSTALL_DIR) {
        return $env:CLANKERS_INSTALL_DIR
    }
    
    return Join-Path $env:LOCALAPPDATA "clankers\bin"
}

function Get-TargetVersion {
    if ($Version) {
        return $Version
    }
    
    if ($env:CLANKERS_VERSION) {
        return $env:CLANKERS_VERSION
    }
    
    return $null
}

function Test-InPath {
    param([string]$Directory)
    
    $paths = $env:PATH -split ";"
    return $paths -contains $Directory
}

function Add-ToUserPath {
    param([string]$Directory)
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    if ($currentPath -notlike "*$Directory*") {
        $newPath = "$Directory;$currentPath"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$Directory;$env:PATH"
        Write-Log "Added $Directory to user PATH"
    }
}

function Main {
    Write-Log "Detected platform: $Target"
    
    # Get version
    $targetVersion = Get-TargetVersion
    if (-not $targetVersion) {
        Write-Log "Fetching latest version..."
        $targetVersion = Get-LatestVersion
        if (-not $targetVersion) {
            Write-ErrorLog "Could not determine latest version. Set `$env:CLANKERS_VERSION = 'v0.1.0'"
        }
    }
    
    Write-Log "Installing version: $targetVersion"
    
    # Determine filenames
    $artifact = "$Target-$BinaryName.exe"
    $destName = "$BinaryName.exe"
    
    # URLs
    $baseUrl = "https://github.com/$Repo/releases/download/$targetVersion"
    $binaryUrl = "$baseUrl/$artifact"
    $checksumsUrl = "$baseUrl/checksums.txt"
    
    # Create temp directory
    $tmpDir = Join-Path $env:TEMP "clankers-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null
    
    try {
        # Download binary
        $binaryPath = Join-Path $tmpDir $artifact
        Write-Log "Downloading $binaryUrl"
        Invoke-WebRequest -Uri $binaryUrl -OutFile $binaryPath -UseBasicParsing
        
        # Download checksums
        $checksumsPath = Join-Path $tmpDir "checksums.txt"
        Write-Log "Downloading checksums..."
        Invoke-WebRequest -Uri $checksumsUrl -OutFile $checksumsPath -UseBasicParsing
        
        # Extract expected checksum
        $checksumLines = Get-Content $checksumsPath
        $expectedLine = $checksumLines | Where-Object { $_ -like "*$artifact*" }
        if (-not $expectedLine) {
            Write-ErrorLog "Could not find checksum for $artifact"
        }
        $expectedChecksum = ($expectedLine -split "\s+")[0].ToLower()
        
        # Verify checksum
        $actualChecksum = Get-FileChecksum -FilePath $binaryPath
        if ($actualChecksum -ne $expectedChecksum) {
            Write-ErrorLog "Checksum mismatch!`n  Expected: $expectedChecksum`n  Actual:   $actualChecksum"
        }
        Write-Log "Checksum verified"
        
        # Install
        $targetInstallDir = Get-TargetInstallDir
        if (-not (Test-Path $targetInstallDir)) {
            New-Item -ItemType Directory -Path $targetInstallDir -Force | Out-Null
        }
        
        $destPath = Join-Path $targetInstallDir $destName
        
        # Stop existing daemon if running
        $existingProcess = Get-Process -Name $BinaryName -ErrorAction SilentlyContinue
        if ($existingProcess) {
            Write-Log "Stopping existing daemon..."
            Stop-Process -Name $BinaryName -Force -ErrorAction SilentlyContinue
            Start-Sleep -Seconds 1
        }
        
        Move-Item -Path $binaryPath -Destination $destPath -Force
        
        Write-Log "Installed to $destPath"
        
        # Check PATH
        if (-not (Test-InPath -Directory $targetInstallDir)) {
            Write-Log ""
            Write-Log "Note: $targetInstallDir is not in your PATH."
            
            if ($AddToPath) {
                Add-ToUserPath -Directory $targetInstallDir
                Write-Log "Restart your terminal for PATH changes to take effect."
            }
            else {
                Write-Log "Run with -AddToPath to add automatically, or add manually:"
                Write-Log "  `$env:PATH = `"$targetInstallDir;`$env:PATH`""
            }
        }
        
        Write-Log ""
        Write-Log "Done! Run 'clankers-daemon --help' to get started."
    }
    finally {
        # Cleanup
        if (Test-Path $tmpDir) {
            Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

Main
