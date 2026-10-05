# Builds the portable release: one VRChatAssetManager.exe with the UI embedded,
# zipped together with a short readme for end users.
#
#   .\scripts\build-release.ps1              # version from `git describe`
#   .\scripts\build-release.ps1 -Version 1.0.0
#
# Needs Node.js and Go on the build machine only; users need neither.
# Output: dist\VRChatAssetManager-<version>.zip

param([string]$Version)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot

if (-not $Version) {
    $Version = git -C $root describe --tags --always --dirty
    if (-not $Version) { $Version = "dev" }
}
Write-Host "Building VRChat Asset Manager $Version" -ForegroundColor Cyan

# 1. Frontend -> static export (frontend\out), API on the same origin.
Push-Location "$root\frontend"
try {
    if (-not (Test-Path node_modules)) {
        npm ci
        if ($LASTEXITCODE) { throw "npm ci failed" }
    }
    # A stale .next cache can reference routes that no longer exist.
    foreach ($dir in ".next", "out") {
        if (Test-Path $dir) { Remove-Item -Recurse -Force $dir }
    }
    $env:NEXT_PUBLIC_SAME_ORIGIN_API = "1"
    npm run build
    if ($LASTEXITCODE) { throw "frontend build failed" }
}
finally {
    Remove-Item Env:NEXT_PUBLIC_SAME_ORIGIN_API -ErrorAction SilentlyContinue
    Pop-Location
}

# 2. Copy the export where go:embed picks it up (git-ignored).
$embedDir = "$root\backend\internal\web\dist"
if (Test-Path $embedDir) { Remove-Item -Recurse -Force $embedDir }
Copy-Item -Recurse "$root\frontend\out" $embedDir

# 3. Go binary with the release tag.
$outDir = "$root\dist\VRChatAssetManager"
if (Test-Path $outDir) { Remove-Item -Recurse -Force $outDir }
New-Item -ItemType Directory -Force $outDir | Out-Null

Push-Location "$root\backend"
try {
    # Exe icon and version details (Explorer > Properties), as a .syso file
    # that go build links in. Windows wants a numeric version here.
    $numeric = if ($Version -match '^v?(\d+(\.\d+){0,3})$') { $Matches[1] } else { "0.0.0" }
    go run github.com/tc-hib/go-winres@v0.3.3 simply --arch amd64 --manifest gui `
        --icon internal\desktop\icon.ico --product-name "VRChat Asset Manager" `
        --file-description "VRChat Asset Manager" --original-filename VRChatAssetManager.exe `
        --product-version $numeric --file-version $numeric
    if ($LASTEXITCODE) { throw "go-winres failed" }

    # -H windowsgui: no console window; the app lives in the tray.
    go build -tags release -trimpath -ldflags "-s -w -H windowsgui -X main.version=$Version" -o "$outDir\VRChatAssetManager.exe" .
    if ($LASTEXITCODE) { throw "go build failed" }
}
finally {
    Remove-Item rsrc_windows_*.syso -ErrorAction SilentlyContinue
    Pop-Location
}

# 4. Readme + zip.
Copy-Item "$root\scripts\release-readme.txt" "$outDir\README.txt"
$zip = "$root\dist\VRChatAssetManager-$Version.zip"
if (Test-Path $zip) { Remove-Item -Force $zip }
# tar.exe (built into Windows 10+) rather than Compress-Archive, which on
# Windows PowerShell 5.1 writes "\" path separators other unzip tools reject.
tar.exe -a -c -f $zip -C "$root\dist" VRChatAssetManager
if ($LASTEXITCODE) { throw "zip failed" }

$size = [math]::Round((Get-Item "$outDir\VRChatAssetManager.exe").Length / 1MB, 1)
Write-Host "Done: $zip (exe $size MB)" -ForegroundColor Green
