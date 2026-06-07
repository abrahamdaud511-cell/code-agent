#!/usr/bin/env pwsh
$Repo = "anomalyco/codeagent"
$Version = if ($env:VERSION) { $env:VERSION } else { "latest" }
$InstallDir = if ($env:INSTALL_DIR) { $env:INSTALL_DIR } else { "$env:LOCALAPPDATA\CodeAgent\bin" }

if ($Version -eq "latest") {
    $Release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name
}

$Arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$Binary = "codeagent_${Version}_windows_${Arch}.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Binary"

Write-Host "Downloading CodeAgent $Version for Windows/$Arch..."
Write-Host "  $Url"

$TmpDir = Join-Path $env:TEMP "codeagent-install"
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null
$ZipFile = Join-Path $TmpDir $Binary

Invoke-WebRequest -Uri $Url -OutFile $ZipFile
Expand-Archive -Path $ZipFile -DestinationPath $TmpDir -Force

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Copy-Item (Join-Path $TmpDir "codeagent.exe") (Join-Path $InstallDir "codeagent.exe") -Force

Remove-Item -Recurse -Force $TmpDir

Write-Host ""
Write-Host "✓ CodeAgent $Version installed to $InstallDir"
Write-Host ""
Write-Host "Add to your PATH:"
Write-Host "  `$env:PATH = `"$InstallDir;`$env:PATH`""
Write-Host ""
Write-Host "Quick start:"
Write-Host "  codeagent auth login --provider openai --key sk-..."
Write-Host "  codeagent"
