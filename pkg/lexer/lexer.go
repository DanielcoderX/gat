package lexer

import (
	"fmt"
	"gat/pkg/token"
)

type Lexer struct {
	input        string
	position     int  // current char position
	readPosition int  // current reading position (after current char)
	ch           byte // current char under examination
	line         int
	col          int
}

func New(input string) *Lexer {
	l := &Lexer{
		input: input,
		line:  1,
		col:   0,
	}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.col++
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespaceAndComments()

	pos := token.Position{Line: l.line, Col: l.col}

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Pos: pos}
		} else if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.FAT_ARROW, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = token.Token{Type: token.ASSIGN, Literal: string(l.ch), Pos: pos}
		}
	case '+':
		tok = token.Token{Type: token.PLUS, Literal: string(l.ch), Pos: pos}
	case '-':
		if l.peekChar() == '>' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.ARROW, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = token.Token{Type: token.MINUS, Literal: string(l.ch), Pos: pos}
		}
	case '*':
		tok = token.Token{Type: token.STAR, Literal: string(l.ch), Pos: pos}
	case '/':
		tok = token.Token{Type: token.SLASH, Literal: string(l.ch), Pos: pos}
	case '%':
		tok = token.Token{Type: token.PERCENT, Literal: string(l.ch), Pos: pos}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = token.Token{Type: token.BANG, Literal: string(l.ch), Pos: pos}
		}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = token.Token{Type: token.LT, Literal: string(l.ch), Pos: pos}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = token.Token{Type: token.GT, Literal: string(l.ch), Pos: pos}
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Pos: pos}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch), Pos: pos}
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Pos: pos}
		}
	case '.':
		tok = token.Token{Type: token.DOT, Literal: string(l.ch), Pos: pos}
	case ',':
		tok = token.Token{Type: token.COMMA, Literal: string(l.ch), Pos: pos}
	case ':':
		tok = token.Token{Type: token.COLON, Literal: string(l.ch), Pos: pos}
	case ';':
		tok = token.Token{Type: token.SEMICOLON, Literal: string(l.ch), Pos: pos}
	case '(':
		tok = token.Token{Type: token.LPAREN, Literal: string(l.ch), Pos: pos}
	case ')':
		tok = token.Token{Type: token.RPAREN, Literal: string(l.ch), Pos: pos}
	case '{':
		tok = token.Token{Type: token.LBRACE, Literal: string(l.ch), Pos: pos}
	case '}':
		tok = token.Token{Type: token.RBRACE, Literal: string(l.ch), Pos: pos}
	case '[':
		tok = token.Token{Type: token.LBRACKET, Literal: string(l.ch), Pos: pos}
	case ']':
		tok = token.Token{Type: token.RBRACKET, Literal: string(l.ch), Pos: pos}
	case '\'':
		tok.Type = token.INT
		tok.Literal = l.readCharLit()
		tok.Pos = pos
		return tok
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
		tok.Pos = pos
		return tok
	case 0:
		tok = token.Token{Type: token.EOF, Literal: "", Pos: pos}
	default:
		if isLetter(l.ch) {
			tok.Pos = pos
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		} else if isDigit(l.ch) {
			tok.Pos = pos
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			return tok
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Pos: pos}
		}
	}

	l.readChar()
	return tok
}

func (l *Lexer) skipWhitespaceAndComments() {
	for {
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
			if l.ch == '\n' {
				l.line++
				l.col = 0
			}
			l.readChar()
		}

		// Check for line comment //
		if l.ch == '/' && l.peekChar() == '/' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
			if l.ch == '\n' {
				l.line++
				l.col = 0
				l.readChar()
			}
		} else if l.ch == '/' && l.peekChar() == '*' {
			// Block comment /* ... */
			l.readChar()
			l.readChar()
			for {
				if l.ch == 0 {
					break
				}
				if l.ch == '\n' {
					l.line++
					l.col = 0
				}
				if l.ch == '*' && l.peekChar() == '/' {
					l.readChar()
					l.readChar()
					break
				}
				l.readChar()
			}
		} else {
			break
		}
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	if l.ch == '0' && (l.peekChar() == 'x' || l.peekChar() == 'X') {
		l.readChar() // '0'
		l.readChar() // 'x'
		for isHexDigit(l.ch) {
			l.readChar()
		}
		return l.input[position:l.position]
	}
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readCharLit() string {
	l.readChar() // skip initial '
	var chVal byte
	if l.ch == '\\' {
		l.readChar()
		switch l.ch {
		case 'n':
			chVal = '\n'
		case 'r':
			chVal = '\r'
		case 't':
			chVal = '\t'
		case '0':
			chVal = 0
		case '\'':
			chVal = '\''
		case '\\':
			chVal = '\\'
		default:
			chVal = l.ch
		}
	} else {
		chVal = l.ch
	}
	l.readChar() // consume char
	if l.ch == '\'' {
		l.readChar() // consume ending '
	}
	return fmt.Sprintf("%d", chVal)
}

func isHexDigit(ch byte) bool {
	return ('0' <= ch && ch <= '9') || ('a' <= ch && ch <= 'f') || ('A' <= ch && ch <= 'F')
}

func (l *Lexer) readString() string {
	l.readChar() // skip initial "
	start := l.position
	var result []byte
	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar()
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 'r':
				result = append(result, '\r')
			case 't':
				result = append(result, '\t')
			case '0':
				result = append(result, 0)
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			default:
				result = append(result, l.ch)
			}
		} else {
			if l.ch == '\n' {
				l.line++
				l.col = 0
			}
			result = append(result, l.ch)
		}
		l.readChar()
	}
	if l.ch == '"' {
		l.readChar() // consume ending "
	}
	if len(result) == 0 && start == l.position-1 {
		return ""
	}
	return string(result)
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
