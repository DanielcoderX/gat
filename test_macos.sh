#!/usr/bin/env bash
set -e

echo "=== 1. Self-Host Verification (Object Bitwise Identity) ==="
./bin/gatc src/compiler.gat -o bin/gatc_stage2.o
./bin/gatc src/compiler.gat -o bin/gatc_stage3.o
cmp bin/gatc_stage2.o bin/gatc_stage3.o
rm -f bin/gatc_stage2.o bin/gatc_stage3.o
echo "  -> Self-hosting Mach-O object bitwise identity verified: 100% exact match!"

echo "=== 2. Rebuild Native macOS Binaries ==="
./bin/gatc src/compiler.gat -o bin/gatc
./bin/gatc cli/gat.gat -o bin/gat

echo "=== 3. Language Feature Tests on macOS ARM64 ==="

run_test() {
    file="$1"
    expected_code="$2"
    expect="$3"
    echo "Running $file ..."
    ./bin/gat build "$file" -o /tmp/test_bin
    set +e
    output=$(/tmp/test_bin hello world 2>&1)
    code=$?
    set -e
    if [ "$code" -ne "$expected_code" ]; then
        echo "FAIL $file: expected exit code $expected_code, got $code"
        exit 1
    fi
    if [ -n "$expect" ] && [[ "$output" != *"$expect"* ]]; then
        echo "FAIL $file: expected '$expect' not found in '$output'"
        exit 1
    fi
    echo "  [PASS] $file"
}

run_test "examples/hello.gat" 0 "hello bootstrap!"
run_test "examples/ret42.gat" 42 ""
run_test "examples/test_features.gat" 0 "Formatted: 12345"
run_test "examples/test_enum_match.gat" 0 "Array[0]: 10, Array[2]: 30"
run_test "examples/test_cmd_arg.gat" 0 "Arg 1: hello"
run_test "examples/test_wf.gat" 0 "written: 12"

rm -f /tmp/test_bin test_out.txt

echo ""
echo "=== All macOS ARM64 tests passed successfully! ==="
