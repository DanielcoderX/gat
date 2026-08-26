package typecheck

import (
	"testing"

	"gat/pkg/lexer"
	"gat/pkg/parser"
)

func TestTypeCheckBasic(t *testing.T) {
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
    let res = inspect(n, raw p);
    return res;
}
`
	l := lexer.New(input)
	p := parser.New(l)
	prog := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("parser errors: %v", p.Errors())
	}

	tc := New()
	ok := tc.CheckProgram(prog)
	if !ok || len(tc.Errors()) > 0 {
		t.Fatalf("typecheck errors: %v", tc.Errors())
	}
}
