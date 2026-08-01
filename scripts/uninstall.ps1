# Breez Windows PowerShell Uninstall Script
$ErrorActionPreference = 'Stop'

$installDir = "$env:LOCALAPPDATA\Programs\Breez"
$binaryPath = Join-Path $installDir "breez.exe"

if (Test-Path -Path $binaryPath) {
    Remove-Item -Path $binaryPath -Force
    Write-Host "Removed binary at $binaryPath"
}

if (Test-Path -Path $installDir) {
    Remove-Item -Path $installDir -Recurse -Force
}

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -like "*$installDir*") {
    $newPath = ($userPath -split ';' | Where-Object { $_ -ne $installDir }) -join ';'
    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    Write-Host "Removed $installDir from User PATH."
}

Write-Host "✔ Breez has been successfully uninstalled from your system."
