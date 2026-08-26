package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type TestCase struct {
	Name         string
	SourceFile   string
	Args         []string
	ExpectedOut  []string
	ExpectedCode int
}

func TestEndToEndBootstrapAndExamples(t *testing.T) {
	rootDir, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("Failed to get root dir: %v", err)
	}

	binDir := filepath.Join(rootDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}

	gatcV0 := filepath.Join(binDir, "gatc-v0.exe")
	gatcV1 := filepath.Join(binDir, "gatc-v1.exe")
	gatcV2 := filepath.Join(binDir, "gatc-v2.exe")
	gatcV3 := filepath.Join(binDir, "gatc-v3.exe")

	// 1. Build Go bootstrap compiler (gatc-v0)
	t.Run("01_Build_Go_Bootstrap_Compiler", func(t *testing.T) {
		cmd := exec.Command("go", "build", "-o", gatcV0, "./cmd/gatc")
		cmd.Dir = rootDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Failed to build gatc-v0: %v\nOutput: %s", err, string(out))
		}
	})

	// 2. Stage 1: gatc-v0 compiles src/compiler.gat -> bin/gatc-v1.exe
	t.Run("02_Stage1_Bootstrap_V1", func(t *testing.T) {
		compilerSrc := filepath.Join(rootDir, "src", "compiler.gat")
		cmd := exec.Command(gatcV0, compilerSrc, "-o", gatcV1)
		cmd.Dir = rootDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Stage 1 compilation failed: %v\nOutput: %s", err, string(out))
		}
	})

	// 3. Stage 2: gatc-v1 compiles src/compiler.gat -> bin/gatc-v2.exe
	t.Run("03_Stage2_Bootstrap_V2", func(t *testing.T) {
		compilerSrc := filepath.Join(rootDir, "src", "compiler.gat")
		cmd := exec.Command(gatcV1, compilerSrc, "-o", gatcV2)
		cmd.Dir = rootDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Stage 2 compilation failed: %v\nOutput: %s", err, string(out))
		}
	})

	// 4. Stage 3: gatc-v2 compiles src/compiler.gat -> bin/gatc-v3.exe
	t.Run("04_Stage3_Bootstrap_V3", func(t *testing.T) {
		compilerSrc := filepath.Join(rootDir, "src", "compiler.gat")
		cmd := exec.Command(gatcV2, compilerSrc, "-o", gatcV3)
		cmd.Dir = rootDir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("Stage 3 compilation failed: %v\nOutput: %s", err, string(out))
		}
	})

	// 5. Verify bitwise reproduction (gatc-v2 == gatc-v3)
	t.Run("05_Bitwise_Identity_V2_vs_V3", func(t *testing.T) {
		b2, err := os.ReadFile(gatcV2)
		if err != nil {
			t.Fatalf("Failed to read gatc-v2: %v", err)
		}
		b3, err := os.ReadFile(gatcV3)
		if err != nil {
			t.Fatalf("Failed to read gatc-v3: %v", err)
		}
		if !bytes.Equal(b2, b3) {
			t.Fatalf("Bitwise identity mismatch: gatc-v2 size=%d, gatc-v3 size=%d", len(b2), len(b3))
		}
	})

	// Test cases across both compilers (gatc-v0 and self-hosted gatc-v2)
	testCases := []TestCase{
		{
			Name:         "Hello_World",
			SourceFile:   "examples/hello.gat",
			ExpectedOut:  []string{"hello bootstrap!"},
			ExpectedCode: 0,
		},
		{
			Name:         "Return_ExitCode_42",
			SourceFile:   "examples/ret42.gat",
			ExpectedCode: 42,
		},
		{
			Name:         "Features_Str_From_Int",
			SourceFile:   "examples/test_features.gat",
			ExpectedOut:  []string{"Testing str_from_int...", "Formatted: 12345"},
			ExpectedCode: 0,
		},
		{
			Name:       "GatMin_Full_Suite",
			SourceFile: "examples/test_gatmin.gat",
			Args:       []string{"alpha", "beta"},
			ExpectedOut: []string{
				"=== Gat-Min Feature Test ===",
				"Arg 1: alpha",
				"Arg 2: beta",
				"Concat: Hello World",
				"Len: 11",
				"Eq test: true",
				"Char 0: 72",
				"Sub: Hello",
				"NumStr: 12345",
				"Buf char 0: 71, char 1: 65, char 2: 84",
				"File read: Hello from Gat File IO!",
				"Done!",
			},
			ExpectedCode: 0,
		},
		{
			Name:       "ARC_Lifecycle_And_Deinit",
			SourceFile: "examples/e2e_arc.gat",
			ExpectedOut: []string{
				"=== Gat v0 Compiler E2E Test ===",
				"[borrowed call] Inspecting person id = 1, age = 25",
				"[raw pointer access] Point x = 10, y = 20",
				"Returned age: 25",
				"--- Scope 1: Allocating Person 101 ---",
				"Person 101 allocated.",
				"--- Rebinding p1 to Person 202 ---",
				"Rebound p1. Exiting create_and_rebind scope...",
				"Exiting main (Person 1 should be freed now)...",
			},
			ExpectedCode: 0,
		},
		{
			Name:       "Nested_Node_Chaining",
			SourceFile: "examples/test_node.gat",
			ExpectedOut: []string{
				"Call str_val: str_concat",
				"Child str_val: str_concat",
			},
			ExpectedCode: 0,
		},
		{
			Name:       "Enum_Match_And_Arrays",
			SourceFile: "examples/test_enum_match.gat",
			ExpectedOut: []string{
				"=== Enum & Match Test ===",
				"Option has value: 42",
				"Option is None",
				"Color: Green",
				"Array[0]: 10, Array[2]: 30",
				"All tests passed!",
			},
			ExpectedCode: 0,
		},
	}

	compilers := []struct {
		Label string
		Path  string
	}{
		{"Bootstrap_Compiler_V0", gatcV0},
		{"SelfHosted_Compiler_V2", gatcV2},
	}

	for _, comp := range compilers {
		t.Run("Compiler_"+comp.Label, func(t *testing.T) {
			for _, tc := range testCases {
				t.Run(tc.Name, func(t *testing.T) {
					srcPath := filepath.Join(rootDir, tc.SourceFile)
					outExe := filepath.Join(binDir, fmt.Sprintf("test_%s_%s.exe", comp.Label, tc.Name))

					// Compile
					compCmd := exec.Command(comp.Path, srcPath, "-o", outExe)
					compCmd.Dir = rootDir
					compOut, err := compCmd.CombinedOutput()
					if err != nil {
						t.Fatalf("Compilation failed with %s: %v\nOutput: %s", comp.Label, err, string(compOut))
					}

					// Run
					runCmd := exec.Command(outExe, tc.Args...)
					runCmd.Dir = rootDir
					runOut, runErr := runCmd.CombinedOutput()
					actualCode := 0
					if runErr != nil {
						if exitErr, ok := runErr.(*exec.ExitError); ok {
							actualCode = exitErr.ExitCode()
						} else {
							t.Fatalf("Execution failed unexpectedly: %v", runErr)
						}
					}

					if actualCode != tc.ExpectedCode {
						t.Fatalf("Exit code mismatch: expected %d, got %d. Output:\n%s", tc.ExpectedCode, actualCode, string(runOut))
					}

					outStr := string(runOut)
					for _, exp := range tc.ExpectedOut {
						if !strings.Contains(outStr, exp) {
							t.Fatalf("Expected output substring not found: %q\nActual output:\n%s", exp, outStr)
						}
					}
				})
			}
		})
	}
}
