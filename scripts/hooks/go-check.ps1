# go-check.ps1 — a PostToolUse hook for Aetox (internal/hook).
#
# After the agent writes or edits a .go file, vet that file's package and run
# its tests, and print the verdict. Whatever this prints reaches the model in
# the tool receipt's `after_hook` field, so a failing test lands in front of
# the model on the very call that broke it, and it keeps working instead of
# reporting the edit as done.
#
# The hook runs under a 10s limit that is deliberately not configurable, so
# this script budgets itself: `go vet` always (it compiles the package, and a
# compile error is the highest-value signal there is), then `go test` for
# whatever time is left, killed and reported honestly when it does not fit.
# The model is told to run the tests itself in that case, which the coding
# desk's direction already asks of it.
#
# Wire it in hooks.json (beside permissions.json in the Aetox data folder):
#   {"hooks":[
#     {"event":"PostToolUse","matcher":"write","command":"& 'D:/Aetox/Aetox/scripts/hooks/go-check.ps1'"},
#     {"event":"PostToolUse","matcher":"edit", "command":"& 'D:/Aetox/Aetox/scripts/hooks/go-check.ps1'"}
#   ]}
#
# Files that are not Go, or not inside a Go module, exit silently: silence
# costs no tokens, and this hook is global to every project the desk opens.

$ErrorActionPreference = 'Continue'

$raw = $env:AETOX_TOOL_ARGS
if (-not $raw) { exit 0 }
try { $call = $raw | ConvertFrom-Json } catch { exit 0 }
$path = $call.args.path
if (-not $path) { exit 0 }
if ($path -notmatch '\.go$') { exit 0 }
if (-not (Test-Path -LiteralPath $path)) { exit 0 }

$dir = Split-Path -Parent (Resolve-Path -LiteralPath $path)
$budgetMs = 8000
$sw = [Diagnostics.Stopwatch]::StartNew()

Push-Location $dir
try {
    $gomod = & go env GOMOD 2>$null
    if (-not $gomod -or $gomod -eq 'NUL' -or $gomod -eq '/dev/null') { exit 0 }

    $pkg = & go list -f '{{.ImportPath}}' . 2>$null
    if (-not $pkg) { $pkg = $dir }

    $vet = (& go vet . 2>&1 | Out-String).Trim()
    if ($LASTEXITCODE -ne 0) {
        Write-Output "go vet ${pkg}: FAILED"
        Write-Output $vet
        exit 1
    }

    $left = $budgetMs - $sw.ElapsedMilliseconds
    if ($left -lt 1500) {
        Write-Output "go vet ${pkg}: ok. Tests were not run here (the hook's time budget is spent); run `go test $pkg` yourself before calling this done."
        exit 0
    }

    $outFile = [IO.Path]::GetTempFileName()
    $errFile = $outFile + '.err'
    $proc = Start-Process -FilePath 'go' -ArgumentList 'test', '-count=1', '.' `
        -WorkingDirectory $dir -NoNewWindow -PassThru `
        -RedirectStandardOutput $outFile -RedirectStandardError $errFile
    if (-not $proc.WaitForExit([int]$left)) {
        try { $proc.Kill() } catch {}
        Remove-Item -LiteralPath $outFile, $errFile -ErrorAction SilentlyContinue
        Write-Output "go vet ${pkg}: ok. `go test $pkg` did not finish inside the hook's time budget and was stopped; run it yourself before calling this done."
        exit 0
    }

    $text = ((Get-Content -LiteralPath $outFile -Raw) + (Get-Content -LiteralPath $errFile -Raw)).Trim()
    Remove-Item -LiteralPath $outFile, $errFile -ErrorAction SilentlyContinue

    if ($proc.ExitCode -ne 0) {
        Write-Output "go test ${pkg}: FAILED"
        # The tail is where the failures are; the head is the passing noise.
        $lines = $text -split "`r?`n"
        if ($lines.Count -gt 60) { $lines = $lines[-60..-1] }
        Write-Output ($lines -join "`n")
        exit 1
    }

    Write-Output "go vet + go test ${pkg}: ok"
    exit 0
}
finally {
    Pop-Location
}
