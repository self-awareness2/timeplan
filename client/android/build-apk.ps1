param(
    [string]$WebUrl = "http://10.0.2.2:8765/"
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command gradle -ErrorAction SilentlyContinue) -and -not (Test-Path ".\gradlew.bat")) {
    throw "Gradle not found. Install Android Studio or Gradle, then run this script again."
}

if (-not $env:ANDROID_HOME -and -not $env:ANDROID_SDK_ROOT) {
    throw "Android SDK not found. Set ANDROID_HOME or ANDROID_SDK_ROOT."
}

$gradleFile = ".\app\build.gradle"
$content = Get-Content $gradleFile -Raw
$escaped = $WebUrl.Replace("\", "\\").Replace('"', '\"')
$content = $content -replace 'buildConfigField "String", "DEFAULT_WEB_URL", "\\".*?\\""', "buildConfigField `"String`", `"DEFAULT_WEB_URL`", `"\`"$escaped\`"`""
Set-Content -Path $gradleFile -Value $content -Encoding UTF8

if (Test-Path ".\gradlew.bat") {
    .\gradlew.bat assembleDebug
} else {
    gradle assembleDebug
}

Write-Host "APK: app\build\outputs\apk\debug\app-debug.apk"
