package main

import (
	"fmt"
	"os"
	"strings"

	"gat/pkg/arc"
	"gat/pkg/codegen"
	"gat/pkg/ir"
	"gat/pkg/lexer"
	"gat/pkg/parser"
	"gat/pkg/pe"
	"gat/pkg/typecheck"
)

func main() {
	var srcFile string
	var outPath string
	dumpIR := false

	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-dump-ir" {
			dumpIR = true
		} else if arg == "-o" && i+1 < len(args) {
			outPath = args[i+1]
			i++
		} else if strings.HasPrefix(arg, "-o=") {
			outPath = strings.TrimPrefix(arg, "-o=")
		} else if !strings.HasPrefix(arg, "-") && srcFile == "" {
			srcFile = arg
		}
	}

	if srcFile == "" {
		fmt.Println("Usage: gatc [options] <source.gat>")
		fmt.Println("  -o <path>    Output executable path")
		fmt.Println("  -dump-ir     Dump intermediate representation")
		os.Exit(1)
	}

	if outPath == "" {
		if strings.HasSuffix(srcFile, ".gat") {
			outPath = strings.TrimSuffix(srcFile, ".gat") + ".exe"
		} else {
			outPath = srcFile + ".exe"
		}
	}

	srcBytes, err := os.ReadFile(srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading source file: %v\n", err)
		os.Exit(1)
	}

	// 1. Lexing & Parsing
	l := lexer.New(string(srcBytes))
	p := parser.New(l)
	astProg := p.ParseProgram()
	if len(p.Errors()) > 0 {
		fmt.Fprintf(os.Stderr, "Parser error(s) (first 5 of %d):\n", len(p.Errors()))
		for i, e := range p.Errors() {
			if i >= 5 {
				break
			}
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
		os.Exit(1)
	}

	// 2. Type Checking
	tc := typecheck.New()
	if !tc.CheckProgram(astProg) {
		fmt.Fprintf(os.Stderr, "Typecheck error(s):\n")
		for _, e := range tc.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
		os.Exit(1)
	}

	// 3. IR Construction
	builder := ir.NewBuilder(tc, astProg)
	irProg := builder.Build()

	// 4. ARC Insertion Pass
	arcPass := arc.NewPass(irProg)
	arcPass.Run()

	if dumpIR {
		fmt.Println("--- IR Program (ARC annotated) ---")
		for _, fn := range irProg.Functions {
			fmt.Printf("function %s():\n", fn.Name)
			for _, inst := range fn.Instructions {
				fmt.Println(inst.String())
			}
		}
	}

	// 5. Machine Code Generation (x86-64)
	cg := codegen.New(irProg)
	code, relocs, symOffsets := cg.Generate()

	// 6. Direct PE Emission
	peBuilder := pe.NewBuilder(irProg, code, relocs, symOffsets)
	if err := peBuilder.BuildExecutable(outPath); err != nil {
		fmt.Fprintf(os.Stderr, "PE generation error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiled %s -> %s successfully.\n", srcFile, outPath)
}
