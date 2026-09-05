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
# One path for write/edit/append/delete; a list of them for a batch, which
# names each edit's own path and has no path of its own. Every file this call
# touched counts, and the package is checked once even when the batch touched
# several files in it.
$paths = @()
if ($call.args.path) { $paths += [string]$call.args.path }
if ($call.args.edits) { foreach ($e in $call.args.edits) { if ($e.path) { $paths += [string]$e.path } } }
$dirs = @()
foreach ($p in $paths) {
    if ($p -notmatch '\.go$') { continue }
    if (-not (Test-Path -LiteralPath $p)) { continue }
    $d = Split-Path -Parent (Resolve-Path -LiteralPath $p)
    if ($dirs -notcontains $d) { $dirs += $d }
}
if ($dirs.Count -eq 0) { exit 0 }
# One package per call is the overwhelmingly common case; a batch that spans
# packages gets the first checked and the rest named, rather than a second
# check that would not fit the budget anyway.
$dir = $dirs[0]
if ($dirs.Count -gt 1) {
    Write-Output ("Note: this call touched Go files in {0} packages; only the first is checked below. Run go test on the others yourself: {1}" -f $dirs.Count, (($dirs | Select-Object -Skip 1) -join ', '))
}
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
