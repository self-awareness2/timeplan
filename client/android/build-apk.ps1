param(
    [string]$ServerUrl = "https://glitzy-parole-ambulance.ngrok-free.dev/",
    [ValidateSet("Debug", "Release")]
    [string]$Configuration = "Debug"
)

$ErrorActionPreference = "Stop"
$sdk = if ($env:ANDROID_SDK_ROOT) { $env:ANDROID_SDK_ROOT } elseif ($env:ANDROID_HOME) { $env:ANDROID_HOME } else { "$env:LOCALAPPDATA\Android\Sdk" }
if (-not (Test-Path $sdk)) { throw "Android SDK not found. Set ANDROID_SDK_ROOT." }
$env:ANDROID_SDK_ROOT = $sdk
$env:ANDROID_HOME = $sdk

$gradle = if (Test-Path ".\gradlew.bat") {
    ".\gradlew.bat"
} else {
    $cached = Get-ChildItem "$env:USERPROFILE\.gradle\wrapper\dists" -Recurse -Filter gradle.bat -ErrorAction SilentlyContinue |
        Sort-Object FullName -Descending | Select-Object -First 1
    if (-not $cached) { throw "Gradle not found. Install Android Studio or add a Gradle wrapper." }
    $cached.FullName
}

& $gradle "assemble$Configuration" "-PserverUrl=$ServerUrl"
if ($LASTEXITCODE -ne 0) { throw "Android build failed with exit code $LASTEXITCODE." }

$variant = $Configuration.ToLowerInvariant()
$apk = Resolve-Path ".\app\build\outputs\apk\$variant\app-$variant.apk"
Write-Host "APK: $apk"
