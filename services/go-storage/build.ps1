# build.ps1 — cross-compile the storage service from Windows PowerShell.
# Equivalent to `make cross` for environments without GNU Make.
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
    $Version = (git describe --tags --always --dirty 2>$null) ?? "dev"
}

$LdFlags = "-s -w -X main.version=$Version"

$Targets = @(
    @{ GOOS = "linux";   GOARCH = "amd64"; Ext = "" }
    @{ GOOS = "linux";   GOARCH = "arm64"; Ext = "" }
    @{ GOOS = "darwin";  GOARCH = "amd64"; Ext = "" }
    @{ GOOS = "windows"; GOARCH = "amd64"; Ext = ".exe" }
)

New-Item -ItemType Directory -Force -Path $BinDir | Out-Null

foreach ($t in $Targets) {
    $name = "$Binary-$($t.GOOS)-$($t.GOARCH)$($t.Ext)"

    if ($Target -ne "all" -and $Target -ne "$($t.GOOS)-$($t.GOARCH)") {
        continue
    }

    Write-Host "Building $name ..."
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

Write-Host "Done. Binaries in ./$BinDir/"
