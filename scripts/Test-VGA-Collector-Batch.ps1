$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Path $PSScriptRoot -Parent
$batPath = Join-Path $repoRoot 'Collect_VGA_Report.bat'
$tempRoot = Join-Path $repoRoot '.local\audit-temp'
$testRoot = Join-Path $tempRoot ('vga-batch-' + [guid]::NewGuid())
$utf8 = New-Object Text.UTF8Encoding($false)
$passed = 0

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) { throw "Assertion failed: $Message" }
}

function Invoke-BatchScenario {
    param([string]$Name, [string]$Action)
    $scenarioDir = Join-Path $testRoot ($Name + " Báo cáo & thử (x16) 'vga' !")
    [void][IO.Directory]::CreateDirectory($scenarioDir)
    $copyPath = Join-Path $scenarioDir 'Collect_VGA_Report.bat'
    # Test the real entrypoint with public deployment constants forced offline.
    $copyText = [IO.File]::ReadAllText($batPath, $utf8)
    $keyPattern = '(?m)^\$script:SigningKey = [^\r\n]*'
    Assert-True ([regex]::Matches($copyText, $keyPattern).Count -eq 1) 'exactly one signing key assignment'
    $copyText = [regex]::Replace($copyText, $keyPattern, { '$script:SigningKey = ''REPLACE_WITH_SMOKE_TEST_SIGNING_KEY''' })
    [IO.File]::WriteAllText($copyPath, $copyText, $utf8)
    $start = New-Object Diagnostics.ProcessStartInfo
    $start.FileName = $env:ComSpec
    $start.Arguments = '/d /s /c ""' + $copyPath + '""'
    $start.WorkingDirectory = $scenarioDir
    $start.UseShellExecute = $false
    $start.CreateNoWindow = $true
    $start.RedirectStandardInput = $true
    $start.RedirectStandardOutput = $true
    $start.RedirectStandardError = $true
    $start.StandardOutputEncoding = $utf8
    $start.StandardErrorEncoding = $utf8
    $start.EnvironmentVariables['VGA_TARGET_BDF'] = 'auto'
    $proc = New-Object Diagnostics.Process
    $proc.StartInfo = $start
    $watch = [Diagnostics.Stopwatch]::StartNew()
    try {
        [void]$proc.Start()
        $stdoutTask = $proc.StandardOutput.ReadToEndAsync()
        $stderrTask = $proc.StandardError.ReadToEndAsync()
        foreach ($answer in @('3', '8', '3', '4', '4', '', $Action, 'K')) {
            $proc.StandardInput.WriteLine($answer)
        }
        $proc.StandardInput.Close()
        if (-not $proc.WaitForExit(90000)) {
            # Terminate only this test process and its child PowerShell process.
            & taskkill.exe /PID $proc.Id /T /F | Out-Null
            throw "BAT timed out in scenario $Name"
        }
        $stdout = $stdoutTask.GetAwaiter().GetResult()
        $stderr = $stderrTask.GetAwaiter().GetResult()
        [IO.File]::WriteAllText((Join-Path $scenarioDir 'stdout.txt'), $stdout, $utf8)
        [IO.File]::WriteAllText((Join-Path $scenarioDir 'stderr.txt'), $stderr, $utf8)
        Assert-True ($proc.ExitCode -eq 0) "BAT exit=$($proc.ExitCode); stderr=$stderr"
        Assert-True ([string]::IsNullOrWhiteSpace($stderr)) "unexpected stderr: $stderr"
        Assert-True ($stdout.Contains('THU THẬP BÁO CÁO CHẨN ĐOÁN VGA')) 'accented UTF-8 menu is displayed'
        Assert-True ($stdout -notmatch 'is not recognized|not recognized as an internal|CommandNotFoundException') 'PowerShell tail is never parsed as batch commands'
        $watch.Stop()
        Write-Host ("BAT {0}: {1:N2}s" -f $Name, $watch.Elapsed.TotalSeconds)
        return [pscustomobject]@{ Output = $stdout; Dir = $scenarioDir; OutDir = (Join-Path $scenarioDir 'vga-reports-out') }
    } finally {
        $proc.Dispose()
    }
}

try {
    [void][IO.Directory]::CreateDirectory($testRoot)
    $bytes = [IO.File]::ReadAllBytes($batPath)
    Assert-True (-not ($bytes.Length -ge 3 -and $bytes[0] -eq 239 -and $bytes[1] -eq 187 -and $bytes[2] -eq 191)) 'BAT must be UTF-8 without BOM for cmd.exe'
    $sourceText = (New-Object Text.UTF8Encoding($false, $true)).GetString($bytes)
    Assert-True ($sourceText -notmatch '(?<!\r)\n') 'BAT must keep CRLF lines for cmd.exe with UTF-8'
    Write-Host 'PASS batch_encoding_utf8_without_bom'
    $passed += 1

    $saved = Invoke-BatchScenario -Name 'save' -Action '1'
    Assert-True ($saved.Output.Contains('Đã lưu báo cáo cục bộ tại:')) 'save confirmation'
    $jsonFiles = @(Get-ChildItem -LiteralPath $saved.OutDir -Filter '*.json' -File)
    $summaries = @(Get-ChildItem -LiteralPath $saved.OutDir -Filter '*-summary.txt' -File)
    $zips = @(Get-ChildItem -LiteralPath $saved.OutDir -Filter '*.zip' -File)
    Assert-True ($jsonFiles.Count -eq 1 -and $summaries.Count -eq 1 -and $zips.Count -eq 1) 'JSON, summary and ZIP saved'
    $report = [IO.File]::ReadAllText($jsonFiles[0].FullName, $utf8) | ConvertFrom-Json
    Assert-True ($report.schema_version -in @(1, 2) -and $report.report_id) 'JSON contains report metadata'
    Write-Host "Observed GPUs=$(@($report.gpus).Count), nvidia.complete=$($report.nvidia.complete) (not hardware acceptance)"
    Assert-True ([IO.File]::ReadAllText($summaries[0].FullName, $utf8).Contains('TÓM TẮT BÁO CÁO CHẨN ĐOÁN VGA')) 'summary retains accented UTF-8'
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [IO.Compression.ZipFile]::OpenRead($zips[0].FullName)
    try {
        $names = @($archive.Entries | ForEach-Object { $_.FullName })
        Assert-True ($names -contains 'report.json' -and $names -contains 'summary.txt') 'ZIP contains both files'
    } finally { $archive.Dispose() }
    Write-Host 'PASS batch_save_real_cmd_in_accented_shell_path'
    $passed += 1

    $send = Invoke-BatchScenario -Name 'placeholder' -Action '2'
    Assert-True ($send.Output.Contains('[CHƯA GỬI ĐƯỢC] Trạng thái: offline')) 'placeholder stays offline'
    Assert-True (-not $send.Output.Contains('Đã gửi báo cáo thành công')) 'no false send success'
    Assert-True (@(Get-ChildItem -LiteralPath $send.OutDir -Filter '*.pending.json' -File).Count -eq 1) 'pending JSON retained'
    Assert-True (@(Get-ChildItem -LiteralPath $send.OutDir -Filter '*.zip' -File).Count -eq 1) 'ZIP retained'
    Write-Host 'PASS batch_placeholder_keeps_pending_without_false_success'
    $passed += 1

    $exitResult = Invoke-BatchScenario -Name 'exit' -Action '3'
    Assert-True ($exitResult.Output.Contains('Đã thoát theo yêu cầu.')) 'explicit exit confirmation'
    Assert-True (@(Get-ChildItem -LiteralPath $exitResult.OutDir -Filter '*.zip' -File -ErrorAction SilentlyContinue).Count -eq 0) 'exit does not save report archive'
    Write-Host 'PASS batch_exit_does_not_execute_powershell_tail'
    $passed += 1
    Write-Host "Summary: Passed=$passed Failed=0 (real host read-only; CMP acceptance remains pending)"
} catch {
    Write-Host "FAIL batch_smoke: $($_.Exception.Message)"
    Write-Host "Logs retained in: $testRoot"
    exit 1
}

# Keep captured transcripts and artifacts under the workspace for review.
Write-Host "Logs: $testRoot"
exit 0
