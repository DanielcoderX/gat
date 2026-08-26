# Native Gat Test Runner
$ErrorActionPreference = "Stop"

Write-Host "=== Building & Verifying Gat Self-Hosting Compiler ===" -ForegroundColor Cyan

# 1. Self-Host Verification (Stage 1 -> Stage 2 bitwise equality)
Write-Host "[1/3] Compiling src/compiler.gat with bin/gatc.exe -> bin/gatc-stage2.exe..."
& .\bin\gatc.exe src\compiler.gat -o bin\gatc-stage2.exe
if ($LASTEXITCODE -ne 0) { Write-Error "Failed to compile stage2"; exit 1 }

Write-Host "[2/3] Compiling src/compiler.gat with bin/gatc-stage2.exe -> bin/gatc-stage3.exe..."
& .\bin\gatc-stage2.exe src\compiler.gat -o bin\gatc-stage3.exe
if ($LASTEXITCODE -ne 0) { Write-Error "Failed to compile stage3"; exit 1 }

$fc = (fc.exe /b bin\gatc-stage2.exe bin\gatc-stage3.exe | Out-String)
if (-not $fc.Contains("no differences encountered")) {
    Write-Error "Bitwise identity mismatch between stage2 and stage3!`n$fc"
    exit 1
}
Write-Host "  -> Self-hosting bitwise identity verified: 100% exact match!" -ForegroundColor Green

# 2. Example Test Suite
Write-Host "[3/3] Running Language Feature Tests..." -ForegroundColor Cyan

$testCases = @(
    @{ Name = "Hello World"; File = "examples/hello.gat"; Args = @(); Expect = "hello bootstrap!"; ExpectedCode = 0 },
    @{ Name = "Exit Code 42"; File = "examples/ret42.gat"; Args = @(); Expect = $null; ExpectedCode = 42 },
    @{ Name = "Str From Int"; File = "examples/test_features.gat"; Args = @(); Expect = "Formatted: 12345"; ExpectedCode = 0 },
    @{ Name = "ARC Lifecycle"; File = "examples/e2e_arc.gat"; Args = @(); Expect = "Exiting main"; ExpectedCode = 0 },
    @{ Name = "Node Chaining"; File = "examples/test_node.gat"; Args = @(); Expect = "Call str_val: str_concat"; ExpectedCode = 0 },
    @{ Name = "Enum & Match & Array"; File = "examples/test_enum_match.gat"; Args = @(); Expect = "Array[0]: 10, Array[2]: 30"; ExpectedCode = 0 },
    @{ Name = "GatMin Suite"; File = "examples/test_gatmin.gat"; Args = @("alpha", "beta"); Expect = "Buf char 0: 71, char 1: 65, char 2: 84"; ExpectedCode = 0 },
    @{ Name = "Multi-Module Import"; File = "examples/test_import.gat"; Args = @(); Expect = "Sum: 13, Prod: 42"; ExpectedCode = 0 },
    @{ Name = "Standard Library (std/)"; File = "examples/test_std.gat"; Args = @(); Expect = "All standard library tests passed!"; ExpectedCode = 0 },
    @{ Name = "For Loops"; File = "examples/test_for.gat"; Args = @(); Expect = "For loop test passed!"; ExpectedCode = 0 },
    @{ Name = "String Interpolation"; File = "examples/test_interp.gat"; Args = @(); Expect = "String interpolation test passed!"; ExpectedCode = 0 },
    @{ Name = "Generics & Parametric Types"; File = "examples/test_generics.gat"; Args = @(); Expect = "Generics test passed!"; ExpectedCode = 0 }
)

$passed = 0
foreach ($t in $testCases) {
    $outBin = "bin\test_out.exe"
    if (Test-Path $outBin) { Remove-Item -Force $outBin }
    
    & .\bin\gatc.exe $t.File -o $outBin
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [FAIL] $($t.Name): Compilation error" -ForegroundColor Red
        exit 1
    }

    $pinfo = New-Object System.Diagnostics.ProcessStartInfo
    $pinfo.FileName = (Resolve-Path $outBin).Path
    $pinfo.Arguments = ($t.Args -join " ")
    $pinfo.RedirectStandardOutput = $true
    $pinfo.UseShellExecute = $false
    $pinfo.CreateNoWindow = $true

    $proc = [System.Diagnostics.Process]::Start($pinfo)
    $output = $proc.StandardOutput.ReadToEnd()
    $proc.WaitForExit()
    $exitCode = $proc.ExitCode

    if ($exitCode -ne $t.ExpectedCode) {
        Write-Host "  [FAIL] $($t.Name): Expected exit code $($t.ExpectedCode), got $exitCode" -ForegroundColor Red
        exit 1
    }

    if ($null -ne $t.Expect -and -not $output.Contains($t.Expect)) {
        Write-Host "  [FAIL] $($t.Name): Expected substring '$($t.Expect)' not found in output:`n$output" -ForegroundColor Red
        exit 1
    }

    Write-Host "  [PASS] $($t.Name)" -ForegroundColor Green
    $passed++
}

Write-Host "`nAll $passed/$($testCases.Count) tests passed successfully on pure Gat compiler!" -ForegroundColor Green
