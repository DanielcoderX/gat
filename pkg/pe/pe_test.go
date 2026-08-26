package pe

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gat/pkg/arc"
	"gat/pkg/codegen"
	"gat/pkg/ir"
	"gat/pkg/lexer"
	"gat/pkg/parser"
	"gat/pkg/typecheck"
)

func TestBuildAndRunPE(t *testing.T) {
	src := `
fn main() -> i64 {
    print("PE Integration Test Success\n");
    return 7;
}
`
	l := lexer.New(src)
	p := parser.New(l)
	astProg := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parse err: %v", p.Errors())
	}

	tc := typecheck.New()
	if !tc.CheckProgram(astProg) {
		t.Fatalf("typecheck err: %v", tc.Errors())
	}

	builder := ir.NewBuilder(tc, astProg)
	irProg := builder.Build()

	arcPass := arc.NewPass(irProg)
	arcPass.Run()

	cg := codegen.New(irProg)
	code, relocs, symOffsets := cg.Generate()

	tmpExe := filepath.Join(os.TempDir(), "gat_test_integration.exe")
	defer os.Remove(tmpExe)

	peBuilder := NewBuilder(irProg, code, relocs, symOffsets)
	if err := peBuilder.BuildExecutable(tmpExe); err != nil {
		t.Fatalf("pe build err: %v", err)
	}

	cmd := exec.Command(tmpExe)
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("exec err: %v", err)
		}
	}

	if !strings.Contains(string(out), "PE Integration Test Success") {
		t.Errorf("expected output to contain 'PE Integration Test Success', got: %q", string(out))
	}
	if exitCode != 7 {
		t.Errorf("expected exit code 7, got %d", exitCode)
	}
}
