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
$tempDir = Join-Path $env:TEMP "breez-install-$version"

Write-Host "Downloading $downloadUrl..."
Invoke-WebRequest -Uri $downloadUrl -OutFile $tempZip

if (!(Test-Path -Path $installDir)) {
    New-Item -ItemType Directory -Path $installDir | Out-Null
}

if (Test-Path -Path $tempDir) {
    Remove-Item -Path $tempDir -Recurse -Force
}
New-Item -ItemType Directory -Path $tempDir | Out-Null

Expand-Archive -Path $tempZip -DestinationPath $tempDir -Force
Copy-Item -Path (Join-Path $tempDir $binaryName) -Destination $installDir -Force

Remove-Item -Path $tempZip -Force
Remove-Item -Path $tempDir -Recurse -Force

# Add to User PATH if missing
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installDir", "User")
    Write-Host "Added $installDir to your User PATH variable."
}

# Update PATH for the current active session as well
if ($env:Path -notlike "*$installDir*") {
    $env:Path = "$env:Path;$installDir"
}

Write-Host "✔ Breez successfully installed to $installDir\$binaryName"
Write-Host "You can now run 'breez version' directly!"

