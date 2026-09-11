@echo off
cd /d "%~dp0"
rem Keeps ALL of Aetox's own data (preferences, sessions, WebView2 profiles,
rem the downloaded rtk binary, ...) off the C: drive during dev — see
rem internal/config.DataRoot. Production builds never set this and use the
rem normal %AppData%\aetox default.
set AETOX_DATA_ROOT=%~dp0.aetox-data
rem The engine is a process beside the app (§248 phase 2). Built here so the
rem dev binary finds it next to itself; without it the app falls back to
rem `go run ./cmd/aetox-engine`, which works but compiles on every start.
go build -o build\bin\aetox-engine.exe ..\cmd\aetox-engine
wails dev
pause
