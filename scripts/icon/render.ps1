# Renders icon.svg into the app icon files (committed, so this only runs when
# the icon changes). Uses Microsoft Edge in headless mode; no other tools.
#
#   .\scripts\icon\render.ps1
#
# Writes:
#   backend\internal\desktop\icon.ico   tray icon, and exe icon at release build
#   frontend\app\favicon.ico            browser tab icon

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$svg = Get-Content -Raw "$PSScriptRoot\icon.svg"
$edge = "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
if (-not (Test-Path $edge)) { $edge = "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe" }

$tmp = Join-Path $env:TEMP "vam-icon"
if (Test-Path $tmp) { Remove-Item -Recurse -Force $tmp }
New-Item -ItemType Directory $tmp | Out-Null

# Render each size from the SVG (sharper than downscaling one bitmap).
$sizes = 16, 24, 32, 48, 64, 256
$pngs = @()
foreach ($n in $sizes) {
    $svg -replace 'width="256" height="256"', "width=""$n"" height=""$n""" | Set-Content -Encoding utf8 "$tmp\i$n.svg"
    "<html><body style=""margin:0;background:transparent""><img src=""i$n.svg"" style=""display:block""></body></html>" |
        Set-Content -Encoding utf8 "$tmp\i$n.html"
    $png = "$tmp\i$n.png"
    $html = "file:///" + ("$tmp\i$n.html" -replace '\\', '/')
    $edgeArgs = "--headless --disable-gpu --hide-scrollbars --default-background-color=00000000 " +
        "--force-device-scale-factor=1 --user-data-dir=""$tmp\edge"" --window-size=$n,$n --screenshot=""$png"" ""$html"""
    Start-Process -FilePath $edge -ArgumentList $edgeArgs -Wait -WindowStyle Hidden
    if (-not (Test-Path $png)) { throw "Edge did not render $n px" }
    $pngs += , [System.IO.File]::ReadAllBytes($png)
}

# ICO container with PNG-compressed images (supported since Windows Vista).
function Write-Ico([string]$path) {
    $stream = [System.IO.File]::Create($path)
    $w = New-Object System.IO.BinaryWriter($stream)
    $w.Write([UInt16]0); $w.Write([UInt16]1); $w.Write([UInt16]$sizes.Count)
    $offset = 6 + 16 * $sizes.Count
    for ($i = 0; $i -lt $sizes.Count; $i++) {
        $dim = if ($sizes[$i] -ge 256) { 0 } else { $sizes[$i] } # 0 means 256
        $w.Write([Byte]$dim); $w.Write([Byte]$dim); $w.Write([Byte]0); $w.Write([Byte]0)
        $w.Write([UInt16]1); $w.Write([UInt16]32)
        $w.Write([UInt32]$pngs[$i].Length); $w.Write([UInt32]$offset)
        $offset += $pngs[$i].Length
    }
    foreach ($p in $pngs) { $w.Write($p) }
    $w.Close()
}

Write-Ico "$root\backend\internal\desktop\icon.ico"
Copy-Item "$root\backend\internal\desktop\icon.ico" "$root\frontend\app\favicon.ico"
Remove-Item -Recurse -Force $tmp
Write-Host "Icon written to backend\internal\desktop\icon.ico and frontend\app\favicon.ico"
