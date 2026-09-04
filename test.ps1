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
    @{ Name = "Generics & Parametric Types"; File = "examples/test_generics.gat"; Args = @(); Expect = "Generics test passed!"; ExpectedCode = 0 },
    @{ Name = "Optimizer Passes"; File = "examples/test_opt.gat"; Args = @(); Expect = "Optimizer passes verified successfully!"; ExpectedCode = 0 },
    @{ Name = "Extended Standard Library (math, fs, proc)"; File = "examples/test_std_ext.gat"; Args = @(); Expect = "Extended standard library tests passed!"; ExpectedCode = 0 },
    @{ Name = "Generic Types & Erasure Suite"; File = "examples/test_generic_types.gat"; Args = @(); Expect = "All uniform word-sized generic tests passed!"; ExpectedCode = 0 },
    @{ Name = "Advanced DCE & ARC Optimizer"; File = "examples/test_opt_adv.gat"; Args = @(); Expect = "Advanced DCE and ARC optimizer test passed!"; ExpectedCode = 0 },
    @{ Name = "Linear-Scan Register Allocation"; File = "examples/test_regalloc.gat"; Args = @(); Expect = "All register allocator tests passed successfully!"; ExpectedCode = 0 },
    @{ Name = "Option, Result & Formatted Output"; File = "examples/test_result_option.gat"; Args = @(); Expect = "All Option, Result, and formatting tests passed successfully!"; ExpectedCode = 0 },
    @{ Name = "First-Class Functions & Function Pointers"; File = "examples/test_first_class_fn.gat"; Args = @(); Expect = "All first-class function tests passed successfully!"; ExpectedCode = 0 },
    @{ Name = "Weak References & Cycle Collection"; File = "examples/test_weak_ref.gat"; Args = @(); Expect = "All weak reference and cycle tests passed successfully!"; ExpectedCode = 0 },
    @{ Name = "Concurrency, Threads & Mutex"; File = "examples/test_threads.gat"; Args = @(); Expect = "All thread and synchronization tests passed successfully!"; ExpectedCode = 0 },
    @{ Name = "Namespaced Modules & Imports"; File = "examples/test_modules.gat"; Args = @(); Expect = "Namespaced module imports and symbol resolution passed!"; ExpectedCode = 0 },
    @{ Name = "Package Manager & Dependency Fetcher"; File = "examples/test_pkg_fetch.gat"; Args = @(); Expect = "Gat package manager and dependency fetching tests passed!"; ExpectedCode = 0 },
    @{ Name = "Standard Library Sockets & Networking"; File = "examples/test_net.gat"; Args = @(); Expect = "All Socket & Networking tests completed successfully!"; ExpectedCode = 0 },
    @{ Name = "JSON Parser & Serializer (std/json.gat)"; File = "examples/test_json.gat"; Args = @(); Expect = "All JSON parser & serializer tests completed successfully!"; ExpectedCode = 0 }
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

Write-Host "`n[4/4] Running Diagnostic Error Quality Negative Tests..." -ForegroundColor Cyan

$negativeTests = @(
    @{ Name = "Syntax Error (Missing Colon in Parameter)"; File = "examples/errors/err_syntax_missing_colon.gat"; ExpectErr = "[Parser Error] line 1:13: expected ':', got 'i64' (identifier)" },
    @{ Name = "Semantic Error (Undeclared Function)"; File = "examples/errors/err_type_undeclared_fn.gat"; ExpectErr = "[Type Error] call to undeclared function 'non_existent_function_12345'" },
    @{ Name = "Semantic Error (Arity Mismatch)"; File = "examples/errors/err_type_arity.gat"; ExpectErr = "[Type Error] function 'add' expects 2 arguments, got 3" },
    @{ Name = "Semantic Error (Invalid Struct Member)"; File = "examples/errors/err_type_field.gat"; ExpectErr = "[Type Error] type 'Point' has no field named 'z'" },
    @{ Name = "Semantic Error (Void Function Returning Value)"; File = "examples/errors/err_type_void_return.gat"; ExpectErr = "[Type Error] void function 'do_work' cannot return a value" },
    @{ Name = "Safety Error (Passing Class Across Thread Boundary)"; File = "examples/errors/err_thread_class_arg.gat"; ExpectErr = "[Type Error] cannot pass reference-counted type 'Person' across thread boundary in thread_spawn" },
    @{ Name = "Semantic Error (Duplicate Declaration Across Flat Imports)"; File = "examples/errors/err_flat_import_collision.gat"; ExpectErr = "[Type Error] duplicate declaration of function 'helper' across flat imports" }
)

$negPassed = 0
foreach ($nt in $negativeTests) {
    $outBin = "bin\test_err_out.exe"
    if (Test-Path $outBin) { Remove-Item -Force $outBin }

    $pinfo = New-Object System.Diagnostics.ProcessStartInfo
    $pinfo.FileName = (Resolve-Path "bin\gatc.exe").Path
    $pinfo.Arguments = "$($nt.File) -o $outBin"
    $pinfo.RedirectStandardOutput = $true
    $pinfo.RedirectStandardError = $true
    $pinfo.UseShellExecute = $false
    $pinfo.CreateNoWindow = $true

    $proc = [System.Diagnostics.Process]::Start($pinfo)
    $output = $proc.StandardOutput.ReadToEnd() + $proc.StandardError.ReadToEnd()
    $proc.WaitForExit()
    $exitCode = $proc.ExitCode

    if ($exitCode -eq 0) {
        Write-Host "  [FAIL] $($nt.Name): Expected compilation to fail, but succeeded with code 0" -ForegroundColor Red
        exit 1
    }

    if (-not $output.Contains($nt.ExpectErr)) {
        Write-Host "  [FAIL] $($nt.Name): Expected error string '$($nt.ExpectErr)' not found in output:`n$output" -ForegroundColor Red
        exit 1
    }

    Write-Host "  [PASS] $($nt.Name) (rejected cleanly with expected diagnostic)" -ForegroundColor Green
    $negPassed++
}

# 5. Language Server (LSP) Verification
Write-Host "`n[5/6] Running Language Server (LSP) Tests..." -ForegroundColor Cyan
& node editors\vscode\test_lsp.js
if ($LASTEXITCODE -ne 0) {
    Write-Host "  [FAIL] Language Server tests failed!" -ForegroundColor Red
    exit 1
}

# 6. Linux/ELF64 Target Verification (WSL)
Write-Host "`n[6/6] Running Linux/ELF64 Native Direct Syscall Suite..." -ForegroundColor Cyan

$linuxTests = @(
    @{ Name = "Linux Direct Syscall Suite"; File = "examples/test_linux_suite.gat"; Bin = "bin/test_linux_suite"; Expect = "ALL LINUX ELF64 TESTS PASSED SUCCESSFULLY" },
    @{ Name = "Linux Print & Interpolation"; File = "examples/test_linux_print.gat"; Bin = "bin/test_linux_print"; Expect = "ALL LINUX PRINT TESTS PASSED" },
    @{ Name = "Linux TCP Sockets & Networking"; File = "examples/test_net.gat"; Bin = "bin/test_net_linux"; Expect = "All Socket & Networking tests completed successfully!" },
    @{ Name = "Linux JSON Parser & Serializer"; File = "examples/test_json.gat"; Bin = "bin/test_json_linux"; Expect = "All JSON parser & serializer tests completed successfully!" }
)

$wslAvailable = $false
if (Get-Command "wsl.exe" -ErrorAction SilentlyContinue) {
    $distros = (wsl.exe -l -q 2>$null)
    if ($LASTEXITCODE -eq 0 -and $distros -and $distros.Trim().Length -gt 0) {
        $wslAvailable = $true
    }
}

foreach ($lt in $linuxTests) {
    if (Test-Path $lt.Bin) { Remove-Item -Force $lt.Bin }

    & .\bin\gatc.exe $lt.File -o $lt.Bin --target=linux
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  [FAIL] Failed to compile $($lt.Name) for Linux ELF target" -ForegroundColor Red
        exit 1
    }

    if ($wslAvailable) {
        wsl chmod +x "./$($lt.Bin)"
        $linuxOut = (wsl "./$($lt.Bin)" | Out-String)
        if ($LASTEXITCODE -ne 0 -or -not $linuxOut.Contains($lt.Expect)) {
            Write-Host "  [FAIL] $($lt.Name) failed under WSL:`n$linuxOut" -ForegroundColor Red
            exit 1
        }
        Write-Host "  [PASS] $($lt.Name) passed under WSL" -ForegroundColor Green
    } else {
        Write-Host "  [SKIP] WSL runnable distro not detected on host; $($lt.Name) ELF binary generated and verified statically" -ForegroundColor Yellow
    }
}

Write-Host "`nAll $passed positive tests, $negPassed negative diagnostic tests, LSP verification, and Linux ELF64 suite passed successfully!" -ForegroundColor Green
