package parser

import (
	"fmt"
	"strconv"

	"gat/pkg/ast"
	"gat/pkg/lexer"
	"gat/pkg/token"
)

// Precedence order
const (
	_ int = iota
	LOWEST
	OR          // ||
	AND         // &&
	EQUALS      // == !=
	LESSGREATER // > < >= <=
	SUM         // + -
	PRODUCT     // * / %
	PREFIX      // -X or !X
	CALL        // fn(X)
	INDEX       // X[Y]
	MEMBER      // X.Y
)

var precedences = map[token.TokenType]int{
	token.OR:       OR,
	token.AND:      AND,
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.LTE:      LESSGREATER,
	token.GT:       LESSGREATER,
	token.GTE:      LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.STAR:     PRODUCT,
	token.PERCENT:  PRODUCT,
	token.LPAREN:   CALL,
	token.LBRACKET: INDEX,
	token.DOT:      MEMBER,
}

type (
	prefixParseFn func() ast.Expr
	infixParseFn  func(ast.Expr) ast.Expr
)

type Parser struct {
	l *lexer.Lexer

	curToken  token.Token
	peekToken token.Token

	errors []string

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
	structNames    map[string]bool
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:           l,
		errors:      []string{},
		structNames: make(map[string]bool),
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentOrStructLitExpr)
	p.registerPrefix(token.INT, p.parseIntLitExpr)
	p.registerPrefix(token.STRING, p.parseStringLitExpr)
	p.registerPrefix(token.TRUE, p.parseBoolLitExpr)
	p.registerPrefix(token.FALSE, p.parseBoolLitExpr)
	p.registerPrefix(token.NIL, p.parseNilLitExpr)
	p.registerPrefix(token.BANG, p.parsePrefixExpr)
	p.registerPrefix(token.MINUS, p.parsePrefixExpr)
	p.registerPrefix(token.RAW, p.parsePrefixExpr)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpr)
	p.registerPrefix(token.NEW, p.parseNewExpr)
	p.registerPrefix(token.PRINT, p.parsePrintExpr)
	p.registerPrefix(token.LBRACKET, p.parseArrayLitExpr)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpr)
	p.registerInfix(token.MINUS, p.parseInfixExpr)
	p.registerInfix(token.SLASH, p.parseInfixExpr)
	p.registerInfix(token.STAR, p.parseInfixExpr)
	p.registerInfix(token.PERCENT, p.parseInfixExpr)
	p.registerInfix(token.EQ, p.parseInfixExpr)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpr)
	p.registerInfix(token.LT, p.parseInfixExpr)
	p.registerInfix(token.LTE, p.parseInfixExpr)
	p.registerInfix(token.GT, p.parseInfixExpr)
	p.registerInfix(token.GTE, p.parseInfixExpr)
	p.registerInfix(token.AND, p.parseInfixExpr)
	p.registerInfix(token.OR, p.parseInfixExpr)
	p.registerInfix(token.LPAREN, p.parseCallExpr)
	p.registerInfix(token.LBRACKET, p.parseIndexExpr)
	p.registerInfix(token.DOT, p.parseMemberExpr)

	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("[%d:%d] expected next token to be %s, got %s ('%s')",
		p.peekToken.Pos.Line, p.peekToken.Pos.Col, t, p.peekToken.Type, p.peekToken.Literal)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}
	return LOWEST
}

func (p *Parser) ParseProgram() *ast.Program {
	prog := &ast.Program{
		Decls: []ast.Decl{},
	}

	for !p.curTokenIs(token.EOF) {
		decl := p.parseDecl()
		if decl != nil {
			prog.Decls = append(prog.Decls, decl)
		}
		p.nextToken()
	}

	return prog
}

func (p *Parser) parseDecl() ast.Decl {
	switch p.curToken.Type {
	case token.STRUCT:
		return p.parseStructDecl()
	case token.CLASS:
		return p.parseClassDecl()
	case token.ENUM:
		return p.parseEnumDecl()
	case token.FN:
		return p.parseFnDecl()
	case token.LET, token.VAR:
		return p.parseLetStmt()
	default:
		p.errors = append(p.errors, fmt.Sprintf("[%d:%d] unexpected token %s at top level", p.curToken.Pos.Line, p.curToken.Pos.Col, p.curToken.Literal))
		return nil
	}
}

func (p *Parser) parseStructDecl() *ast.StructDecl {
	pos := p.curToken.Pos
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	p.structNames[name] = true

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	fields := p.parseFields()

	return &ast.StructDecl{
		Position: pos,
		Name:     name,
		Fields:   fields,
	}
}

func (p *Parser) parseClassDecl() *ast.ClassDecl {
	pos := p.curToken.Pos
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	p.structNames[name] = true

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	classDecl := &ast.ClassDecl{
		Position: pos,
		Name:     name,
		Fields:   []ast.Field{},
	}

	p.nextToken() // move inside {
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.DEINIT) {
			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			classDecl.Deinit = p.parseBlockStmt()
		} else if p.curTokenIs(token.IDENT) {
			fieldName := p.curToken.Literal
			if !p.expectPeek(token.COLON) {
				return nil
			}
			p.nextToken()
			fieldType := p.parseType()
			classDecl.Fields = append(classDecl.Fields, ast.Field{
				Name: fieldName,
				Type: fieldType,
				Pos:  p.curToken.Pos,
			})
			if p.peekTokenIs(token.SEMICOLON) {
				p.nextToken()
			}
		}
		p.nextToken()
	}

	return classDecl
}

func (p *Parser) parseFields() []ast.Field {
	var fields []ast.Field
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) {
			fieldName := p.curToken.Literal
			if !p.expectPeek(token.COLON) {
				return nil
			}
			p.nextToken()
			fieldType := p.parseType()
			fields = append(fields, ast.Field{
				Name: fieldName,
				Type: fieldType,
				Pos:  p.curToken.Pos,
			})
			if p.peekTokenIs(token.SEMICOLON) {
				p.nextToken()
			}
		}
		p.nextToken()
	}
	return fields
}

func (p *Parser) parseFnDecl() *ast.FnDecl {
	pos := p.curToken.Pos
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	params := p.parseParams()

	var retType ast.TypeNode = &ast.NamedType{Position: p.curToken.Pos, Name: "void"}
	if p.peekTokenIs(token.ARROW) || p.peekTokenIs(token.COLON) {
		p.nextToken() // skip -> or :
		p.nextToken()
		retType = p.parseType()
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	body := p.parseBlockStmt()

	return &ast.FnDecl{
		Position:   pos,
		Name:       name,
		Params:     params,
		ReturnType: retType,
		Body:       body,
	}
}

func (p *Parser) parseParams() []ast.Param {
	var params []ast.Param
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
	paramName := p.curToken.Literal
	if !p.expectPeek(token.COLON) {
		return nil
	}
	p.nextToken()
	paramType := p.parseType()
	params = append(params, ast.Param{
		Name: paramName,
		Type: paramType,
		Pos:  p.curToken.Pos,
	})

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		pName := p.curToken.Literal
		if !p.expectPeek(token.COLON) {
			return nil
		}
		p.nextToken()
		pType := p.parseType()
		params = append(params, ast.Param{
			Name: pName,
			Type: pType,
			Pos:  p.curToken.Pos,
		})
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return params
}

func (p *Parser) parseType() ast.TypeNode {
	pos := p.curToken.Pos
	if p.curTokenIs(token.RAW) {
		p.nextToken()
		base := p.parseType()
		return &ast.RawType{Position: pos, BaseType: base}
	}
	if p.curTokenIs(token.LBRACKET) {
		p.nextToken()
		elem := p.parseType()
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken() // skip ;
			p.nextToken() // number
			lenVal, _ := strconv.ParseInt(p.curToken.Literal, 10, 64)
			if !p.expectPeek(token.RBRACKET) {
				return nil
			}
			return &ast.ArrayType{Position: pos, ElemType: elem, Length: lenVal}
		}
		if !p.expectPeek(token.RBRACKET) {
			return nil
		}
		return &ast.SliceType{Position: pos, ElemType: elem}
	}
	return &ast.NamedType{Position: pos, Name: p.curToken.Literal}
}

func (p *Parser) parseEnumDecl() *ast.EnumDecl {
	pos := p.curToken.Pos
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal
	p.structNames[name] = true

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	enumDecl := &ast.EnumDecl{
		Position: pos,
		Name:     name,
		Variants: []ast.EnumVariant{},
	}

	p.nextToken() // move inside {
	tag := int64(0)
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) {
			vName := p.curToken.Literal
			var vTypes []ast.TypeNode
			if p.peekTokenIs(token.LPAREN) {
				p.nextToken() // consume '('
				if !p.peekTokenIs(token.RPAREN) {
					p.nextToken()
					vTypes = append(vTypes, p.parseType())
					for p.peekTokenIs(token.COMMA) {
						p.nextToken() // skip ,
						p.nextToken()
						vTypes = append(vTypes, p.parseType())
					}
				}
				if !p.expectPeek(token.RPAREN) {
					return nil
				}
			}
			enumDecl.Variants = append(enumDecl.Variants, ast.EnumVariant{
				Position: p.curToken.Pos,
				Name:     vName,
				Tag:      tag,
				Types:    vTypes,
			})
			tag++
			if p.peekTokenIs(token.COMMA) || p.peekTokenIs(token.SEMICOLON) {
				p.nextToken()
			}
		}
		p.nextToken()
	}
	return enumDecl
}

func (p *Parser) parseBlockStmt() *ast.BlockStmt {
	block := &ast.BlockStmt{
		Position: p.curToken.Pos,
		Stmts:    []ast.Stmt{},
	}

	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStmt()
		if stmt != nil {
			block.Stmts = append(block.Stmts, stmt)
		}
		p.nextToken()
	}

	return block
}

func (p *Parser) parseStmt() ast.Stmt {
	switch p.curToken.Type {
	case token.LET, token.VAR:
		return p.parseLetStmt()
	case token.RETURN:
		return p.parseReturnStmt()
	case token.IF:
		return p.parseIfStmt()
	case token.MATCH:
		return p.parseMatchStmt()
	case token.WHILE:
		return p.parseWhileStmt()
	case token.LBRACE:
		return p.parseBlockStmt()
	default:
		return p.parseAssignOrExprStmt()
	}
}

func (p *Parser) parseLetStmt() *ast.LetStmt {
	pos := p.curToken.Pos
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	name := p.curToken.Literal

	var typeNode ast.TypeNode
	if p.peekTokenIs(token.COLON) {
		p.nextToken()
		p.nextToken()
		typeNode = p.parseType()
	}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()
	val := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return &ast.LetStmt{
		Position: pos,
		Name:     name,
		Type:     typeNode,
		Value:    val,
	}
}

func (p *Parser) parseReturnStmt() *ast.ReturnStmt {
	pos := p.curToken.Pos
	p.nextToken()

	var val ast.Expr
	if !p.curTokenIs(token.SEMICOLON) && !p.curTokenIs(token.RBRACE) {
		val = p.parseExpression(LOWEST)
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return &ast.ReturnStmt{
		Position: pos,
		Value:    val,
	}
}

func (p *Parser) parseIfStmt() *ast.IfStmt {
	pos := p.curToken.Pos
	p.nextToken()
	cond := p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	thenBranch := p.parseBlockStmt()
	var elseBranch ast.Stmt

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()
		if p.peekTokenIs(token.IF) {
			p.nextToken()
			elseBranch = p.parseIfStmt()
		} else if p.peekTokenIs(token.LBRACE) {
			p.nextToken()
			elseBranch = p.parseBlockStmt()
		}
	}

	return &ast.IfStmt{
		Position:   pos,
		Condition:  cond,
		ThenBranch: thenBranch,
		ElseBranch: elseBranch,
	}
}

func (p *Parser) parseWhileStmt() *ast.WhileStmt {
	pos := p.curToken.Pos
	p.nextToken()
	cond := p.parseExpression(LOWEST)

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	body := p.parseBlockStmt()

	return &ast.WhileStmt{
		Position:  pos,
		Condition: cond,
		Body:      body,
	}
}

func (p *Parser) parseAssignOrExprStmt() ast.Stmt {
	pos := p.curToken.Pos
	expr := p.parseExpression(LOWEST)

	if p.peekTokenIs(token.ASSIGN) {
		p.nextToken() // skip to '='
		p.nextToken() // skip '='
		val := p.parseExpression(LOWEST)
		if p.peekTokenIs(token.SEMICOLON) {
			p.nextToken()
		}
		return &ast.AssignStmt{
			Position: pos,
			Target:   expr,
			Value:    val,
		}
	}

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return &ast.ExprStmt{
		Position: pos,
		Expr:     expr,
	}
}

func (p *Parser) parseExpression(precedence int) ast.Expr {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.errors = append(p.errors, fmt.Sprintf("[%d:%d] no prefix parse function for %s ('%s')",
			p.curToken.Pos.Line, p.curToken.Pos.Col, p.curToken.Type, p.curToken.Literal))
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil || leftExp == nil {
			return leftExp
		}
		p.nextToken()
		leftExp = infix(leftExp)
	}

	return leftExp
}

func (p *Parser) parseIdentOrStructLitExpr() ast.Expr {
	pos := p.curToken.Pos
	name := p.curToken.Literal

	if p.structNames[name] && p.peekTokenIs(token.LBRACE) {
		p.nextToken() // move to {
		var inits []ast.FieldInit
		p.nextToken()
		for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
			if p.curTokenIs(token.IDENT) {
				fName := p.curToken.Literal
				if !p.expectPeek(token.COLON) {
					return nil
				}
				p.nextToken()
				fVal := p.parseExpression(LOWEST)
				inits = append(inits, ast.FieldInit{Name: fName, Value: fVal})
				if p.peekTokenIs(token.COMMA) {
					p.nextToken()
				}
			}
			p.nextToken()
		}

		return &ast.NewExpr{
			Position:   pos,
			TypeName:   name,
			FieldInits: inits,
		}
	}

	return &ast.IdentExpr{Position: pos, Name: name}
}

func (p *Parser) parseIntLitExpr() ast.Expr {
	val, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("[%d:%d] could not parse %q as integer",
			p.curToken.Pos.Line, p.curToken.Pos.Col, p.curToken.Literal))
		return nil
	}
	return &ast.IntLitExpr{Position: p.curToken.Pos, Value: val}
}

func (p *Parser) parseStringLitExpr() ast.Expr {
	return &ast.StringLitExpr{Position: p.curToken.Pos, Value: p.curToken.Literal}
}

func (p *Parser) parseBoolLitExpr() ast.Expr {
	return &ast.BoolLitExpr{Position: p.curToken.Pos, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseNilLitExpr() ast.Expr {
	return &ast.NilLitExpr{Position: p.curToken.Pos}
}

func (p *Parser) parsePrefixExpr() ast.Expr {
	pos := p.curToken.Pos
	op := p.curToken.Literal
	p.nextToken()
	right := p.parseExpression(PREFIX)
	return &ast.UnaryExpr{
		Position: pos,
		Op:       op,
		Right:    right,
	}
}

func (p *Parser) parseGroupedExpr() ast.Expr {
	p.nextToken()
	exp := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}
	return exp
}

func (p *Parser) parseInfixExpr(left ast.Expr) ast.Expr {
	pos := p.curToken.Pos
	op := p.curToken.Literal
	precedence := p.curPrecedence()
	p.nextToken()
	right := p.parseExpression(precedence)
	return &ast.BinaryExpr{
		Position: pos,
		Left:     left,
		Op:       op,
		Right:    right,
	}
}

func (p *Parser) parseCallExpr(left ast.Expr) ast.Expr {
	if left == nil {
		return nil
	}
	callee := ""
	if ident, ok := left.(*ast.IdentExpr); ok {
		callee = ident.Name
	} else if mem, ok := left.(*ast.MemberExpr); ok {
		if targetIdent, ok := mem.Target.(*ast.IdentExpr); ok {
			callee = fmt.Sprintf("%s.%s", targetIdent.Name, mem.Member)
		}
	}

	if callee == "" {
		p.errors = append(p.errors, fmt.Sprintf("[%d:%d] only direct identifier or member calls supported currently", left.Pos().Line, left.Pos().Col))
		return nil
	}

	args := p.parseCallArgs()
	return &ast.CallExpr{
		Position: left.Pos(),
		Callee:   callee,
		Args:     args,
	}
}

func (p *Parser) parseCallArgs() []ast.Expr {
	var args []ast.Expr
	if p.peekTokenIs(token.RPAREN) {
		p.nextToken()
		return args
	}

	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))

	for p.peekTokenIs(token.COMMA) {
		p.nextToken()
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return args
}

func (p *Parser) parseMemberExpr(left ast.Expr) ast.Expr {
	pos := p.curToken.Pos
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	memberName := p.curToken.Literal
	return &ast.MemberExpr{
		Position: pos,
		Target:   left,
		Member:   memberName,
	}
}

func (p *Parser) parseNewExpr() ast.Expr {
	pos := p.curToken.Pos
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	typeName := p.curToken.Literal

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	var inits []ast.FieldInit
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		if p.curTokenIs(token.IDENT) {
			fName := p.curToken.Literal
			if !p.expectPeek(token.COLON) {
				return nil
			}
			p.nextToken()
			fVal := p.parseExpression(LOWEST)
			inits = append(inits, ast.FieldInit{Name: fName, Value: fVal})
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
		}
		p.nextToken()
	}

	return &ast.NewExpr{
		Position:   pos,
		TypeName:   typeName,
		FieldInits: inits,
	}
}

func (p *Parser) parsePrintExpr() ast.Expr {
	pos := p.curToken.Pos
	if !p.expectPeek(token.LPAREN) {
		return nil
	}
	args := p.parseCallArgs()
	return &ast.PrintExpr{
		Position: pos,
		Args:     args,
	}
}

func (p *Parser) parseIndexExpr(left ast.Expr) ast.Expr {
	pos := p.curToken.Pos
	p.nextToken() // move inside [
	idx := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return &ast.IndexExpr{
		Position: pos,
		Target:   left,
		Index:    idx,
	}
}

func (p *Parser) parseArrayLitExpr() ast.Expr {
	pos := p.curToken.Pos
	p.nextToken() // move inside [
	var elems []ast.Expr
	if p.curTokenIs(token.RBRACKET) {
		return &ast.ArrayLitExpr{Position: pos, Elements: elems}
	}
	elems = append(elems, p.parseExpression(LOWEST))
	for p.peekTokenIs(token.COMMA) {
		p.nextToken() // skip comma
		p.nextToken()
		if p.curTokenIs(token.RBRACKET) {
			break
		}
		elems = append(elems, p.parseExpression(LOWEST))
	}
	if !p.curTokenIs(token.RBRACKET) && !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return &ast.ArrayLitExpr{Position: pos, Elements: elems}
}

func (p *Parser) parseMatchStmt() *ast.MatchStmt {
	pos := p.curToken.Pos
	p.nextToken() // skip match
	target := p.parseExpression(LOWEST)
	if !p.expectPeek(token.LBRACE) {
		return nil
	}
	matchStmt := &ast.MatchStmt{
		Position: pos,
		Expr:     target,
		Arms:     []ast.MatchArm{},
	}

	p.nextToken() // inside {
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		armPos := p.curToken.Pos
		isWildcard := false
		enumName := ""
		variantName := ""
		var bindings []string

		if p.curTokenIs(token.IDENT) {
			first := p.curToken.Literal
			if first == "_" {
				isWildcard = true
			} else if p.peekTokenIs(token.DOT) {
				p.nextToken() // skip dot
				if !p.expectPeek(token.IDENT) {
					return nil
				}
				enumName = first
				variantName = p.curToken.Literal
			} else {
				variantName = first
			}

			if !isWildcard && p.peekTokenIs(token.LPAREN) {
				p.nextToken() // consume '('
				if !p.peekTokenIs(token.RPAREN) {
					p.nextToken()
					bindings = append(bindings, p.curToken.Literal)
					for p.peekTokenIs(token.COMMA) {
						p.nextToken() // skip ,
						p.nextToken()
						bindings = append(bindings, p.curToken.Literal)
					}
				}
				if !p.expectPeek(token.RPAREN) {
					return nil
				}
			}

			if !p.expectPeek(token.FAT_ARROW) {
				return nil
			}

			if !p.expectPeek(token.LBRACE) {
				return nil
			}
			body := p.parseBlockStmt()
			matchStmt.Arms = append(matchStmt.Arms, ast.MatchArm{
				Position:   armPos,
				EnumName:   enumName,
				Variant:    variantName,
				Bindings:   bindings,
				Body:       body,
				IsWildcard: isWildcard,
			})
			if p.peekTokenIs(token.COMMA) {
				p.nextToken()
			}
		}
		p.nextToken()
	}
	return matchStmt
}
