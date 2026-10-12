# ============================================================================
# tools/package_release.ps1 - Gat Release Packaging Script
# ============================================================================
param (
    [string]$Version = "v1.1.0"
)

$ErrorActionPreference = "Stop"

Write-Host "=== Packaging Gat Toolchain Release $Version ===" -ForegroundColor Cyan

$distDir = "dist"
if (Test-Path $distDir) {
    Remove-Item -Recurse -Force $distDir
}
New-Item -ItemType Directory -Path $distDir | Out-Null

# 1. Compile All Target Binaries
Write-Host "[1/5] Compiling native binaries for all release targets..." -ForegroundColor Yellow

# Windows x64
& .\bin\gatc.exe src\compiler.gat -o bin\gatc_rel.exe
Move-Item -Force bin\gatc_rel.exe bin\gatc.exe
& .\bin\gatc.exe cli\gat.gat -o bin\gat_rel.exe
Move-Item -Force bin\gat_rel.exe bin\gat.exe

# Linux x64
& .\bin\gat.exe build src\compiler.gat -o bin\gatc-linux-x64 --target=linux
& .\bin\gat.exe build cli\gat.gat -o bin\gat-linux-x64 --target=linux

# Linux ARM64
& .\bin\gat.exe build src\compiler.gat -o bin\gatc-linux-arm64 --target=linux-arm64
& .\bin\gat.exe build cli\gat.gat -o bin\gat-linux-arm64 --target=linux-arm64

# macOS ARM64
& .\bin\gat.exe build src\compiler.gat -o bin\gatc-macos.o --target=macos-arm64
& .\bin\gat.exe build cli\gat.gat -o bin\gat-macos.o --target=macos-arm64

# 2. Package Windows x64 Archive (.zip)
Write-Host "[2/5] Packaging Windows x64 release..." -ForegroundColor Yellow
$winPkg = "gat-$Version-windows-x64"
$winPkgDir = Join-Path $distDir $winPkg
New-Item -ItemType Directory -Path (Join-Path $winPkgDir "bin") | Out-Null
Copy-Item "bin\gat.exe", "bin\gatc.exe" (Join-Path $winPkgDir "bin")
Copy-Item -Recurse "std", "docs", "examples", "README.md", "LICENSE" $winPkgDir
Compress-Archive -Path "$winPkgDir\*" -DestinationPath "$distDir\$winPkg.zip" -Force
Remove-Item -Recurse -Force $winPkgDir

# 3. Package Linux x64 Archive (.zip/.tar.gz)
Write-Host "[3/5] Packaging Linux x64 release..." -ForegroundColor Yellow
$linuxPkg = "gat-$Version-linux-x64"
$linuxPkgDir = Join-Path $distDir $linuxPkg
New-Item -ItemType Directory -Path (Join-Path $linuxPkgDir "bin") | Out-Null
Copy-Item "bin\gat-linux-x64" (Join-Path $linuxPkgDir "bin\gat")
Copy-Item "bin\gatc-linux-x64" (Join-Path $linuxPkgDir "bin\gatc")
Copy-Item -Recurse "std", "docs", "examples", "README.md", "LICENSE" $linuxPkgDir
Compress-Archive -Path "$linuxPkgDir\*" -DestinationPath "$distDir\$linuxPkg.zip" -Force
Remove-Item -Recurse -Force $linuxPkgDir

# 4. Package Linux ARM64 Archive (.zip/.tar.gz)
Write-Host "[4/5] Packaging Linux ARM64 release..." -ForegroundColor Yellow
$armPkg = "gat-$Version-linux-arm64"
$armPkgDir = Join-Path $distDir $armPkg
New-Item -ItemType Directory -Path (Join-Path $armPkgDir "bin") | Out-Null
Copy-Item "bin\gat-linux-arm64" (Join-Path $armPkgDir "bin\gat")
Copy-Item "bin\gatc-linux-arm64" (Join-Path $armPkgDir "bin\gatc")
Copy-Item -Recurse "std", "docs", "examples", "README.md", "LICENSE" $armPkgDir
Compress-Archive -Path "$armPkgDir\*" -DestinationPath "$distDir\$armPkg.zip" -Force
Remove-Item -Recurse -Force $armPkgDir

# 5. Generate Checksums
Write-Host "[5/5] Generating SHA256 checksums..." -ForegroundColor Yellow
$checksums = @()
Get-ChildItem -Path $distDir -Filter "*.zip" | ForEach-Object {
    $hash = (Get-FileHash -Path $_.FullName -Algorithm SHA256).Hash.ToLower()
    $checksums += "$hash  $($_.Name)"
}
$checksums | Out-File -FilePath "$distDir\SHA256SUMS.txt" -Encoding ascii

Write-Host "=== Packaging Complete! ===" -ForegroundColor Green
Get-ChildItem $distDir | Select-Object Name, Length
