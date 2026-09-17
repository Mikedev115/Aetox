param(
    [Parameter(Mandatory = $true)]
    [ValidateSet("aetox", "codex")]
    [string]$Harness,

    [string]$Task = "sqlite-email-migration",

    [ValidateRange(1, 999)]
    [int]$Run = 1,

    [ValidateRange(1, 120)]
    [int]$TimeoutMinutes = 20,

    [string]$ResultsRoot = "",

    [switch]$UseExistingAetoxSession
)

$ErrorActionPreference = "Stop"
$repoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot ".."))
$taskRoot = Join-Path $repoRoot "benchmarks\harness\tasks\$Task"
$baseRoot = Join-Path $taskRoot "base"
$promptPath = Join-Path $taskRoot "PROMPT.md"
$hiddenPath = Join-Path $taskRoot "hidden_tests.py"

foreach ($required in @($baseRoot, $promptPath, $hiddenPath)) {
    if (-not (Test-Path -LiteralPath $required)) {
        throw "benchmark fixture is incomplete: $required"
    }
}

if ([string]::IsNullOrWhiteSpace($ResultsRoot)) {
    $ResultsRoot = Join-Path $repoRoot "output\harness-bench"
}
$ResultsRoot = [IO.Path]::GetFullPath($ResultsRoot)
$runID = "$Harness-$Task-r$Run"
$runRoot = Join-Path $ResultsRoot $runID
if (Test-Path -LiteralPath $runRoot) {
    throw "run directory already exists: $runRoot"
}

New-Item -ItemType Directory -Path $runRoot -Force | Out-Null
$workspace = Join-Path $runRoot "workspace"
Copy-Item -LiteralPath $baseRoot -Destination $workspace -Recurse

function Invoke-CapturedProcess {
    param(
        [Parameter(Mandatory = $true)][string]$File,
        [Parameter(Mandatory = $true)][string[]]$Arguments,
        [Parameter(Mandatory = $true)][string]$WorkingDirectory,
        [hashtable]$Environment = @{},
        [int]$TimeoutMilliseconds = 1200000
    )

    $start = New-Object System.Diagnostics.ProcessStartInfo
    $start.FileName = $File
    $start.WorkingDirectory = $WorkingDirectory
    $start.UseShellExecute = $false
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    foreach ($argument in $Arguments) {
        [void]$start.ArgumentList.Add($argument)
    }
    foreach ($entry in $Environment.GetEnumerator()) {
        $start.Environment[$entry.Key] = [string]$entry.Value
    }

    $process = New-Object System.Diagnostics.Process
    $process.StartInfo = $start
    [void]$process.Start()
    $stdout = $process.StandardOutput.ReadToEndAsync()
    $stderr = $process.StandardError.ReadToEndAsync()
    $completed = $process.WaitForExit($TimeoutMilliseconds)
    if (-not $completed) {
        $process.Kill($true)
        $process.WaitForExit()
    }
    return [pscustomobject]@{
        ExitCode = if ($completed) { $process.ExitCode } else { 124 }
        TimedOut = -not $completed
        Stdout = $stdout.Result
        Stderr = $stderr.Result
    }
}

$git = (Get-Command git -ErrorAction Stop).Source
$python = (Get-Command python -ErrorAction Stop).Source
$go = (Get-Command go -ErrorAction Stop).Source
$codex = (Get-Command codex -ErrorAction Stop).Source

& $git -C $workspace init -q
& $git -C $workspace config user.email "benchmark@aetox.local"
& $git -C $workspace config user.name "Aetox Harness Benchmark"
& $git -C $workspace add -- .
& $git -C $workspace commit -qm "benchmark baseline"

$sourceCommit = (& $git -C $repoRoot rev-parse HEAD).Trim()
$aetoxVersion = (& (Join-Path $repoRoot "aetox.exe") --version | Select-Object -First 1)
$codexVersion = (& $codex --version).Trim()
$prompt = Get-Content -LiteralPath $promptPath -Raw -Encoding UTF8
$lastMessage = Join-Path $runRoot "last-message.txt"
$reportPath = Join-Path $runRoot "aetox-report.jsonl"
$temporaryDataRoot = $null

if ($Harness -eq "aetox") {
    $binaryRoot = Join-Path $runRoot "bin"
    New-Item -ItemType Directory -Path $binaryRoot -Force | Out-Null
    $aetox = Join-Path $binaryRoot "aetox.exe"
    $build = Invoke-CapturedProcess -File $go -Arguments @("build", "-o", $aetox, "./cmd/aetox") -WorkingDirectory $repoRoot -TimeoutMilliseconds 300000
    Set-Content -LiteralPath (Join-Path $runRoot "build.stdout.txt") -Value $build.Stdout -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $runRoot "build.stderr.txt") -Value $build.Stderr -Encoding UTF8
    if ($build.ExitCode -ne 0) {
        throw "building Aetox failed with exit $($build.ExitCode)"
    }

    $temporaryDataRoot = Join-Path ([IO.Path]::GetTempPath()) ("aetox-harness-" + [guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Path $temporaryDataRoot -Force | Out-Null
    if (-not $UseExistingAetoxSession) {
        throw "Aetox benchmark requires -UseExistingAetoxSession so the temporary profile can use the existing ChatGPT account"
    }
    $sourceOAuth = Join-Path $env:APPDATA "aetox\oauth.json"
    if (-not (Test-Path -LiteralPath $sourceOAuth)) {
        throw "the existing Aetox OAuth store was not found at $sourceOAuth"
    }
    Copy-Item -LiteralPath $sourceOAuth -Destination (Join-Path $temporaryDataRoot "oauth.json")
    $auth = Invoke-CapturedProcess -File $aetox -Arguments @("auth") -WorkingDirectory $repoRoot -Environment @{ AETOX_DATA_ROOT = $temporaryDataRoot } -TimeoutMilliseconds 60000
    Set-Content -LiteralPath (Join-Path $runRoot "auth-setup.txt") -Value ($auth.Stdout + $auth.Stderr) -Encoding UTF8
    if ($auth.ExitCode -ne 0 -or $auth.Stdout -notmatch "codex\s+ChatGPT\s+signed in") {
        throw "the copied Aetox session is not signed in to Codex"
    }
    $emptyProfile = Join-Path $runRoot "empty-user-profile"
    New-Item -ItemType Directory -Path $emptyProfile -Force | Out-Null
    $command = $aetox
    $arguments = @(
        "--model-provider", "codex",
        "--model-name", "gpt-5.6-luna",
        "--think", "low",
        "--approval", "full-access",
        "--root", $workspace,
        "--report", $reportPath,
        "chat", $prompt
    )
    $environment = @{
        AETOX_DATA_ROOT = $temporaryDataRoot
        USERPROFILE = $emptyProfile
    }
} else {
    $command = $codex
    $arguments = @(
        "exec",
        "--ephemeral",
        "--ignore-user-config",
        "--ignore-rules",
        "--model", "gpt-5.6-luna",
        "-c", 'model_reasoning_effort="low"',
        "--sandbox", "workspace-write",
        "--ask-for-approval", "never",
        "--cd", $workspace,
        "--skip-git-repo-check",
        "--color", "never",
        "--output-last-message", $lastMessage,
        $prompt
    )
    $environment = @{}
}

$started = [DateTimeOffset]::UtcNow
$timer = [Diagnostics.Stopwatch]::StartNew()
try {
    $execution = Invoke-CapturedProcess -File $command -Arguments $arguments -WorkingDirectory $workspace -Environment $environment -TimeoutMilliseconds ($TimeoutMinutes * 60 * 1000)
} finally {
    $timer.Stop()
    if ($temporaryDataRoot) {
        $tempRoot = [IO.Path]::GetFullPath([IO.Path]::GetTempPath())
        $resolvedDataRoot = [IO.Path]::GetFullPath($temporaryDataRoot)
        if (-not $resolvedDataRoot.StartsWith($tempRoot, [StringComparison]::OrdinalIgnoreCase)) {
            throw "refusing to remove benchmark data outside the temporary directory: $resolvedDataRoot"
        }
        Remove-Item -LiteralPath $resolvedDataRoot -Recurse -Force
    }
}

Set-Content -LiteralPath (Join-Path $runRoot "stdout.txt") -Value $execution.Stdout -Encoding UTF8
Set-Content -LiteralPath (Join-Path $runRoot "stderr.txt") -Value $execution.Stderr -Encoding UTF8
if (-not (Test-Path -LiteralPath $lastMessage)) {
    Set-Content -LiteralPath $lastMessage -Value $execution.Stdout -Encoding UTF8
}

(& $git -C $workspace diff --binary HEAD) | Set-Content -LiteralPath (Join-Path $runRoot "changes.patch") -Encoding UTF8
$changedFiles = @(& $git -C $workspace diff --name-only HEAD)
$changedFiles | Set-Content -LiteralPath (Join-Path $runRoot "changed-files.txt") -Encoding UTF8

$scorer = Join-Path $workspace ".bench-hidden-tests.py"
Copy-Item -LiteralPath $hiddenPath -Destination $scorer
$score = Invoke-CapturedProcess -File $python -Arguments @($scorer) -WorkingDirectory $workspace -TimeoutMilliseconds 120000
Set-Content -LiteralPath (Join-Path $runRoot "hidden-tests.txt") -Value ($score.Stderr + $score.Stdout) -Encoding UTF8
$scoreLine = ($score.Stdout -split "`r?`n" | Where-Object { $_.Trim().StartsWith("{") } | Select-Object -Last 1)
$hidden = if ($scoreLine) { $scoreLine | ConvertFrom-Json } else { [pscustomobject]@{ tests = 0; passed = 0; failures = 0; errors = 1; successful = $false } }
Remove-Item -LiteralPath $scorer -Force

$sourceFiles = Get-ChildItem -LiteralPath $workspace -Recurse -File | Where-Object {
    $_.Extension -in @(".py", ".sql") -and $_.FullName -notmatch "[\\/]\.git[\\/]"
}
$lineCounts = @{}
$longestLine = 0
foreach ($file in $sourceFiles) {
    $relative = [IO.Path]::GetRelativePath($workspace, $file.FullName).Replace("\", "/")
    $lines = @(Get-Content -LiteralPath $file.FullName)
    $lineCounts[$relative] = $lines.Count
    foreach ($line in $lines) {
        if ($line.Length -gt $longestLine) {
            $longestLine = $line.Length
        }
    }
}

$result = [ordered]@{
    schema = 1
    run_id = $runID
    harness = $Harness
    task = $Task
    run = $Run
    source_commit = $sourceCommit
    model = "gpt-5.6-luna"
    reasoning = "low"
    started_at = $started.ToString("o")
    seconds = [Math]::Round($timer.Elapsed.TotalSeconds, 3)
    timeout_minutes = $TimeoutMinutes
    timed_out = $execution.TimedOut
    harness_exit_code = $execution.ExitCode
    scorer_exit_code = $score.ExitCode
    hidden_tests = $hidden
    changed_files = $changedFiles
    longest_line = $longestLine
    lines_by_file = $lineCounts
    versions = [ordered]@{
        aetox = $aetoxVersion
        codex = $codexVersion
        python = (& $python --version 2>&1).Trim()
        go = (& $go version).Trim()
    }
}
$result | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $runRoot "result.json") -Encoding UTF8
$result | ConvertTo-Json -Depth 8
