package parser

import (
	"testing"

	"gat/pkg/lexer"
)

func TestParseClassStructFn(t *testing.T) {
	input := `
struct Point {
    x: i64;
    y: i64;
}

class Node {
    value: i64;
    next: Node;

    deinit {
        print("Node deinitialized\n");
    }
}

fn inspect(n: Node, p: raw Point) -> i64 {
    let copy = n;
    print("inspecting\n");
    return copy.value;
}

fn main() -> i64 {
    let p = Point { x: 10, y: 20 };
    let n = new Node { value: 42, next: nil };
    inspect(n, raw p);
    return 0;
}
`
	l := lexer.New(input)
	p := New(l)
	prog := p.ParseProgram()

	if len(p.Errors()) > 0 {
		for _, err := range p.Errors() {
			t.Errorf("parser error: %s", err)
		}
	}

	if len(prog.Decls) != 4 {
		t.Fatalf("expected 4 decls, got %d", len(prog.Decls))
	}
}
