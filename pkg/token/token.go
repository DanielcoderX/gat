package token

type TokenType string

const (
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	// Identifiers & Literals
	IDENT  TokenType = "IDENT"
	INT    TokenType = "INT"
	STRING TokenType = "STRING"
	BOOL   TokenType = "BOOL"

	// Operators
	ASSIGN   TokenType = "="
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	STAR     TokenType = "*"
	SLASH    TokenType = "/"
	PERCENT  TokenType = "%"
	BANG     TokenType = "!"
	EQ       TokenType = "=="
	NOT_EQ   TokenType = "!="
	LT       TokenType = "<"
	LTE      TokenType = "<="
	GT       TokenType = ">"
	GTE      TokenType = ">="
	AND      TokenType = "&&"
	OR       TokenType = "||"
	DOT       TokenType = "."
	ARROW     TokenType = "->"
	FAT_ARROW TokenType = "=>"

	// Delimiters
	COMMA     TokenType = ","
	COLON     TokenType = ":"
	SEMICOLON TokenType = ";"
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"
	LBRACKET  TokenType = "["
	RBRACKET  TokenType = "]"

	// Keywords
	FN     TokenType = "fn"
	LET    TokenType = "let"
	VAR    TokenType = "var"
	STRUCT TokenType = "struct"
	CLASS  TokenType = "class"
	ENUM   TokenType = "enum"
	MATCH  TokenType = "match"
	DEINIT TokenType = "deinit"
	RETURN TokenType = "return"
	IF     TokenType = "if"
	ELSE   TokenType = "else"
	WHILE  TokenType = "while"
	NEW    TokenType = "new"
	RAW    TokenType = "raw"
	PRINT  TokenType = "print"
	TRUE   TokenType = "true"
	FALSE  TokenType = "false"
	NIL    TokenType = "nil"
)

var keywords = map[string]TokenType{
	"fn":     FN,
	"let":    LET,
	"var":    VAR,
	"struct": STRUCT,
	"class":  CLASS,
	"enum":   ENUM,
	"match":  MATCH,
	"deinit": DEINIT,
	"return": RETURN,
	"if":     IF,
	"else":   ELSE,
	"while":  WHILE,
	"new":    NEW,
	"raw":    RAW,
	"print":  PRINT,
	"true":   TRUE,
	"false":  FALSE,
	"nil":    NIL,
}

type Position struct {
	Line int
	Col  int
}

type Token struct {
	Type    TokenType
	Literal string
	Pos     Position
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
