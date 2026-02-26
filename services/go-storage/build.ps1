# build.ps1 — cross-compile the storage service from Windows PowerShell.
# Equivalent to `make cross` for environments without GNU Make.
#
# Compatible with PowerShell 5.1+ (ships with Windows 10/11).
# The ?? null-coalescing operator requires PS7+; use if/else here for PS5.1.
#
# Usage:
#   .\build.ps1              # all targets
#   .\build.ps1 -Target linux-amd64

param(
    [string]$Target = "all",
    [string]$Version = ""
)

$Binary = "storage-service"
$CmdPath = "./cmd/server"
$BinDir = "bin"

if (-not $Version) {
    $gitVersion = git describe --tags --always --dirty 2>$null
    if ($gitVersion) {
        $Version = $gitVersion
    } else {
        $Version = "dev"
    }
}

$LdFlags = "-s -w -X main.version=$Version"

$Targets = @(
    @{ GOOS = "linux";   GOARCH = "amd64"; Ext = "" }
    @{ GOOS = "linux";   GOARCH = "arm64"; Ext = "" }
    @{ GOOS = "darwin";  GOARCH = "amd64"; Ext = "" }
    @{ GOOS = "darwin";  GOARCH = "arm64"; Ext = "" }
    @{ GOOS = "windows"; GOARCH = "amd64"; Ext = ".exe" }
)

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

foreach ($t in $Targets) {
    $name = "$Binary-$($t.GOOS)-$($t.GOARCH)$($t.Ext)"

    if ($Target -ne "all" -and $Target -ne "$($t.GOOS)-$($t.GOARCH)") {
        continue
    }

    Write-Output "Building $name ..."
    $env:CGO_ENABLED = "0"
    $env:GOOS        = $t.GOOS
    $env:GOARCH      = $t.GOARCH

    go build -trimpath -ldflags $LdFlags -o "$BinDir/$name" $CmdPath
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Build failed for $name"
        exit 1
    }
}

# Restore env so subsequent go commands in the same shell behave normally.
Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
Remove-Item Env:GOOS        -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH      -ErrorAction SilentlyContinue

Write-Output "Done. Binaries in ./$BinDir/"
