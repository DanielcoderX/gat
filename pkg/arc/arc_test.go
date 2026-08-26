package arc

import (
	"testing"

	"gat/pkg/ir"
	"gat/pkg/lexer"
	"gat/pkg/parser"
	"gat/pkg/typecheck"
)

func TestArcInsertionPass(t *testing.T) {
	input := `
class Node {
    val: i64;
    deinit {
        print("Node deinit\n");
    }
}

fn test_arc() {
    let a = new Node { val: 10 };
    let b = a;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	astProg := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser err: %v", p.Errors())
	}

	tc := typecheck.New()
	if !tc.CheckProgram(astProg) {
		t.Fatalf("typecheck err: %v", tc.Errors())
	}

	b := ir.NewBuilder(tc, astProg)
	irProg := b.Build()

	pass := NewPass(irProg)
	pass.Run()

	var foundRetain, foundRelease bool
	for _, fn := range irProg.Functions {
		if fn.Name == "test_arc" {
			for _, inst := range fn.Instructions {
				if inst.Op == ir.OpRetain {
					foundRetain = true
				}
				if inst.Op == ir.OpRelease {
					foundRelease = true
				}
			}
		}
	}

	if !foundRetain {
		t.Errorf("expected OpRetain in test_arc")
	}
	if !foundRelease {
		t.Errorf("expected OpRelease in test_arc")
	}
}
