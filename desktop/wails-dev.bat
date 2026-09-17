@echo off
cd /d "%~dp0"
rem Keeps ALL of Aetox's own data (preferences, sessions, WebView2 profiles,
rem the downloaded rtk binary, ...) off the C: drive during dev — see
rem internal/config.DataRoot. Production builds never set this and use the
rem normal %AppData%\aetox default.
set AETOX_DATA_ROOT=%~dp0.aetox-data
rem The engine is a process beside the app (§248 phase 2). Built here so the
rem dev app can use this one explicit build. Plain `wails dev` intentionally
rem leaves AETOX_ENGINE unset and runs the engine from current source instead,
rem so a stale binary in build\bin can never shadow source changes.
go build -o build\bin\aetox-engine.exe ..\cmd\aetox-engine
set "AETOX_ENGINE=%~dp0build\bin\aetox-engine.exe"
wails dev
pause
