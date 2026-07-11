# TimePlanner

TimePlanner is organized as a unified client/server app. The Windows desktop app and the web app both use the same HTTP backend, so account login and schedule data stay synchronized across platforms.

## Structure

```text
time/
├─ server/          Go + Gin + SQLite backend
├─ client/
│  ├─ web/          Shared Vite + TypeScript UI
│  └─ windows/      Windows WebView shell
├─ data/            Runtime data, ignored by git
└─ CMakeLists.txt   Builds the Windows desktop shell
```

## Architecture

```text
Windows .exe
Web browser
Future mobile app
    -> client/web UI
    -> server HTTP API
    -> data/server/timeplanner.sqlite
```

The desktop executable is now a native Windows shell around the same web client used by browsers. Business logic, accounts, and persistence live in `server/`.

## Server

Technology:

- Go
- Gin
- SQLite
- JWT login tokens
- bcrypt password hashing

Server layout:

```text
server/
├─ cmd/server/main.go
├─ internal/app/
├─ internal/auth/
├─ internal/db/
├─ internal/schedules/
└─ go.mod
```

Install Go first, then run:

```powershell
cd C:\code\ai\time\server
go mod tidy
go run .\cmd\server
```

The server listens on:

```text
http://localhost:8765
```

For production, set:

```powershell
$env:TIME_PLANNER_SECRET = "replace-with-a-long-random-secret"
```

Data is stored at:

```text
data/server/timeplanner.sqlite
```

## Web Client

```powershell
cd C:\code\ai\time\client\web
npm install
npm run build
```

The Go server serves `client/web/dist`.

## Windows Client

Start the server first, then run:

```powershell
.\build\Release\time_planner.exe
```

By default the Windows client opens:

```text
http://localhost:8765
```

To point it at a deployed server:

```powershell
$env:TIME_PLANNER_URL = "https://your-domain.example"
.\build\Release\time_planner.exe
```

Build the Windows shell:

```powershell
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release
```

## Mobile Path

The mobile app should reuse `client/web` through Capacitor or a similar WebView wrapper. Point the mobile client at the same server URL to share accounts and data.
