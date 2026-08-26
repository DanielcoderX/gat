package ast

import (
	"gat/pkg/token"
)

type Node interface {
	Pos() token.Position
}

type Decl interface {
	Node
	declNode()
}

type Stmt interface {
	Node
	stmtNode()
}

type Expr interface {
	Node
	exprNode()
}

// TypeNode represents a type expression in the AST
type TypeNode interface {
	Node
	typeNode()
	String() string
}

type NamedType struct {
	Position token.Position
	Name     string
}

func (t *NamedType) Pos() token.Position { return t.Position }
func (t *NamedType) typeNode()          {}
func (t *NamedType) String() string     { return t.Name }

type RawType struct {
	Position token.Position
	BaseType TypeNode
}

func (t *RawType) Pos() token.Position { return t.Position }
func (t *RawType) typeNode()          {}
func (t *RawType) String() string     { return "raw " + t.BaseType.String() }

type ArrayType struct {
	Position token.Position
	ElemType TypeNode
	Length   int64
}

func (t *ArrayType) Pos() token.Position { return t.Position }
func (t *ArrayType) typeNode()          {}
func (t *ArrayType) String() string     { return "[" + t.ElemType.String() + "]" }

type SliceType struct {
	Position token.Position
	ElemType TypeNode
}

func (t *SliceType) Pos() token.Position { return t.Position }
func (t *SliceType) typeNode()          {}
func (t *SliceType) String() string     { return "[]" + t.ElemType.String() }

// EnumVariant represents a single variant of an enum
type EnumVariant struct {
	Position token.Position
	Name     string
	Tag      int64
	Types    []TypeNode
}

// EnumDecl represents an enum definition
type EnumDecl struct {
	Position token.Position
	Name     string
	Variants []EnumVariant
}

func (d *EnumDecl) Pos() token.Position { return d.Position }
func (d *EnumDecl) declNode()          {}

// Program is the root AST node
type Program struct {
	Decls []Decl
}

func (p *Program) Pos() token.Position {
	if len(p.Decls) > 0 {
		return p.Decls[0].Pos()
	}
	return token.Position{Line: 1, Col: 1}
}

// Field represents a field in struct or class
type Field struct {
	Name string
	Type TypeNode
	Pos  token.Position
}

// StructDecl: value type
type StructDecl struct {
	Position token.Position
	Name     string
	Fields   []Field
}

func (d *StructDecl) Pos() token.Position { return d.Position }
func (d *StructDecl) declNode()          {}

// ClassDecl: reference type with optional deinit
type ClassDecl struct {
	Position token.Position
	Name     string
	Fields   []Field
	Deinit   *BlockStmt // optional user deinit block
}

func (d *ClassDecl) Pos() token.Position { return d.Position }
func (d *ClassDecl) declNode()          {}

// Param represents a function parameter
type Param struct {
	Name string
	Type TypeNode
	Pos  token.Position
}

// FnDecl represents a function definition
type FnDecl struct {
	Position   token.Position
	Name       string
	Params     []Param
	ReturnType TypeNode
	Body       *BlockStmt
}

func (d *FnDecl) Pos() token.Position { return d.Position }
func (d *FnDecl) declNode()          {}

// BlockStmt
type BlockStmt struct {
	Position token.Position
	Stmts    []Stmt
}

func (s *BlockStmt) Pos() token.Position { return s.Position }
func (s *BlockStmt) stmtNode()          {}

// LetStmt
type LetStmt struct {
	Position token.Position
	Name     string
	Type     TypeNode // optional
	Value    Expr
}

func (s *LetStmt) Pos() token.Position { return s.Position }
func (s *LetStmt) stmtNode()          {}
func (s *LetStmt) declNode()          {}

// AssignStmt
type AssignStmt struct {
	Position token.Position
	Target   Expr // IdentExpr or MemberExpr
	Value    Expr
}

func (s *AssignStmt) Pos() token.Position { return s.Position }
func (s *AssignStmt) stmtNode()          {}

// ReturnStmt
type ReturnStmt struct {
	Position token.Position
	Value    Expr // optional
}

func (s *ReturnStmt) Pos() token.Position { return s.Position }
func (s *ReturnStmt) stmtNode()          {}

// IfStmt
type IfStmt struct {
	Position   token.Position
	Condition  Expr
	ThenBranch *BlockStmt
	ElseBranch Stmt // *BlockStmt or *IfStmt
}

func (s *IfStmt) Pos() token.Position { return s.Position }
func (s *IfStmt) stmtNode()          {}

// WhileStmt
type WhileStmt struct {
	Position  token.Position
	Condition Expr
	Body      *BlockStmt
}

func (s *WhileStmt) Pos() token.Position { return s.Position }
func (s *WhileStmt) stmtNode()          {}

// MatchArm represents a single pattern arm in a match statement
type MatchArm struct {
	Position   token.Position
	EnumName   string
	Variant    string
	Bindings   []string // variable names bound to payload fields
	Body       *BlockStmt
	IsWildcard bool
}

// MatchStmt represents a match statement
type MatchStmt struct {
	Position token.Position
	Expr     Expr
	Arms     []MatchArm
}

func (s *MatchStmt) Pos() token.Position { return s.Position }
func (s *MatchStmt) stmtNode()          {}

// ArrayLitExpr represents [e1, e2, ...]
type ArrayLitExpr struct {
	Position token.Position
	Elements []Expr
}

func (e *ArrayLitExpr) Pos() token.Position { return e.Position }
func (e *ArrayLitExpr) exprNode()          {}

// ExprStmt
type ExprStmt struct {
	Position token.Position
	Expr     Expr
}

func (s *ExprStmt) Pos() token.Position { return s.Position }
func (s *ExprStmt) stmtNode()          {}

// Expressions

type IdentExpr struct {
	Position token.Position
	Name     string
}

func (e *IdentExpr) Pos() token.Position { return e.Position }
func (e *IdentExpr) exprNode()          {}

type IntLitExpr struct {
	Position token.Position
	Value    int64
}

func (e *IntLitExpr) Pos() token.Position { return e.Position }
func (e *IntLitExpr) exprNode()          {}

type StringLitExpr struct {
	Position token.Position
	Value    string
}

func (e *StringLitExpr) Pos() token.Position { return e.Position }
func (e *StringLitExpr) exprNode()          {}

type BoolLitExpr struct {
	Position token.Position
	Value    bool
}

func (e *BoolLitExpr) Pos() token.Position { return e.Position }
func (e *BoolLitExpr) exprNode()          {}

type NilLitExpr struct {
	Position token.Position
}

func (e *NilLitExpr) Pos() token.Position { return e.Position }
func (e *NilLitExpr) exprNode()          {}

type BinaryExpr struct {
	Position token.Position
	Left     Expr
	Op       string
	Right    Expr
}

func (e *BinaryExpr) Pos() token.Position { return e.Position }
func (e *BinaryExpr) exprNode()          {}

type UnaryExpr struct {
	Position token.Position
	Op       string
	Right    Expr
}

func (e *UnaryExpr) Pos() token.Position { return e.Position }
func (e *UnaryExpr) exprNode()          {}

type CallExpr struct {
	Position token.Position
	Callee   string
	Args     []Expr
}

func (e *CallExpr) Pos() token.Position { return e.Position }
func (e *CallExpr) exprNode()          {}

type MemberExpr struct {
	Position token.Position
	Target   Expr
	Member   string
}

func (e *MemberExpr) Pos() token.Position { return e.Position }
func (e *MemberExpr) exprNode()          {}

type NewExpr struct {
	Position  token.Position
	TypeName  string
	FieldInits []FieldInit
}

type FieldInit struct {
	Name  string
	Value Expr
}

func (e *NewExpr) Pos() token.Position { return e.Position }
func (e *NewExpr) exprNode()          {}

type PrintExpr struct {
	Position token.Position
	Args     []Expr
}

func (e *PrintExpr) Pos() token.Position { return e.Position }
func (e *PrintExpr) exprNode()          {}

type IndexExpr struct {
	Position token.Position
	Target   Expr
	Index    Expr
}

func (e *IndexExpr) Pos() token.Position { return e.Position }
func (e *IndexExpr) exprNode()          {}
