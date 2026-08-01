# Breez Windows PowerShell Install Script
$ErrorActionPreference = 'Stop'

$repo = "devlopersabbir/breez"
$installDir = "$env:LOCALAPPDATA\Programs\Breez"
$binaryName = "breez.exe"

Write-Host "Fetching latest release of Breez..."
$releaseUrl = "https://api.github.com/repos/$repo/releases/latest"
$release = Invoke-RestMethod -Uri $releaseUrl
$tag = $release.tag_name
$version = $tag.TrimStart('v')

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "32bit" }
$tarName = "breez_${version}_windows_${arch}.zip"
$downloadUrl = "https://github.com/$repo/releases/download/$tag/$tarName"

$tempZip = Join-Path $env:TEMP $tarName

Write-Host "Downloading $downloadUrl..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip

if (!(Test-Path -Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

Expand-Archive -Path $tempZip -DestinationPath $tempDir -Force
Copy-Item -Path (Join-Path $tempDir $binaryName) -Destination $installDir -Force

Remove-Item -Path $tempZip -Force

# Add to User PATH if missing
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your User PATH variable."
}

Write-Host "✔ Breez successfully installed to $installDir\$binaryName"
Write-Host "Please restart your terminal session and run 'breez version'!"
