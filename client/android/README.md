# TimePlanner Android

This is the smallest Android client for TimePlanner: a native Java WebView shell.

It does not bundle the web assets. The APK opens the TimePlanner web server URL,
so account login and data sync still use the same Go backend.

## Build

Requirements:

- Android SDK
- Gradle or Android Studio
- JDK

From this folder:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\build-apk.ps1 -WebUrl "https://your-domain.example/"
```

For Android emulator testing against the local Go server:

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\build-apk.ps1 -WebUrl "http://10.0.2.2:8765/"
```

Output:

```text
app\build\outputs\apk\debug\app-debug.apk
```

For a real phone, do not use `localhost` or `10.0.2.2`; use your Cloudflare Tunnel
URL, public domain, or another reachable server address.
