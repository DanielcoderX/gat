# Gat Language Specification

**Version:** 0.3.0  
**Target:** x86-64 Native (Windows PE32+) *[Linux ELF64: Planned Future Target]*  
**Memory Model:** Automatic Reference Counting (ARC)

---

## Table of Contents
1. [Overview](#1-overview)
2. [Lexical Structure](#2-lexical-structure)
3. [Type System & Generics](#3-type-system--generics)
4. [Memory Model & ARC](#4-memory-model--arc)
5. [Declarations](#5-declarations)
6. [Statements & Control Flow](#6-statements--control-flow)
7. [Expressions & Operators](#7-expressions--operators)
8. [Module & Import System](#8-module--import-system)
9. [Grammar (EBNF)](#9-grammar-ebnf)

---

## 1. Overview
Gat is a statically-typed, memory-safe, self-hosting systems language designed for high performance, direct native code emission, and deterministic automatic reference counted memory management without a bulky runtime.

---

## 2. Lexical Structure

### 2.1 Identifiers
Identifiers start with an ASCII letter or underscore, followed by letters, digits, or underscores:
`[a-zA-Z_][a-zA-Z0-9_]*`

### 2.2 Keywords
`fn`, `let`, `if`, `else`, `while`, `for`, `in`, `return`, `match`, `struct`, `class`, `enum`, `import`, `new`, `nil`, `true`, `false`

### 2.3 Literals
- **Integer**: Decimal literals (`42`, `0`, `1000000`). Stored as 64-bit signed integers (`i64`).
- **Boolean**: `true`, `false`.
- **String**: Enclosed in double quotes (`"hello\n"`). Supports escape sequences `\n`, `\r`, `\t`, `\\`, `\"`.
- **String Interpolation**: Expressions enclosed in `{}` within double quotes (`"count: {x + 1}"`).
- **Array Literals**: Bracketed comma-separated items (`[10, 20, 30]`).
- **Nil**: `nil` represents null pointers for structs, classes, and references.

### 2.4 Comments
- Line comments: `// ...` (continues to end of line).

---

## 3. Type System & Generics

### 3.1 Primitive Types
- `i64`: 64-bit signed integer.
- `bool`: Boolean value (`true` or `false`).
- `string`: Immutable reference-counted byte buffer.
- `void`: Return type for procedures with no return value.
- `array`: Dynamic array of reference-counted elements.

### 3.2 Composite Types
- **Struct**: Value aggregate defined with `struct Name { field: Type; ... }`.
- **Class**: Reference-counted heap object defined with `class Name { field: Type; ... }`.
- **Enum**: Tagged algebraic sum type with optional payload:
  ```gat
  enum Result<T, E> {
      Ok(T),
      Err(E)
  }
  ```

### 3.3 Generics Model (Uniform Word-Sized Type Erasure)
Gat implements **Uniform 64-bit Word-Sized Generics** via compile-time type erasure:
- Generic parameters `<T, U>` represent any 64-bit word-sized type:
  - Primitives: `i64`, `bool`
  - References & Handles: `string`, `class` instances, `array`, raw pointers
- **Representation**: Every generic type parameter occupies exactly 1 machine word (8 bytes).
- **Value Type Restriction**: Multi-word value types (multi-field `struct` value types) cannot be used directly as generic type arguments; they must be wrapped in a `class` (heap reference) or passed via pointers.
- **Benefits**: Eliminates template code-bloat, enables fast single-pass compilation, and provides 100% type soundness across all 64-bit word-sized types.

```gat
class Pair<T, U> {
    first: T;
    second: U;
}

fn make_pair<T, U>(a: T, b: U) -> Pair<T, U> {
    return new Pair<T, U> {
        first: a,
        second: b
    };
}
```

---

## 4. Memory Model & ARC

### 4.1 Header Layout
All heap-allocated objects (classes, strings, arrays) are prefixed with a 16-byte metadata header:
```
Offset -16: [ ref_count : i64 ]   (64-bit reference counter)
Offset  -8: [ type_tag  : i64 ]   (64-bit type discriminator / size)
Offset   0: [ User Payload    ]   <-- Object pointer returned to user
```

### 4.2 Retain & Release Semantics
- **Allocation**: `new Class { ... }` or `alloc_mem(sz)` initializes `ref_count = 1`.
- **Retain**: Passing or assigning heap references invokes `__gat_retain` (`inc qword ptr [ptr-16]`).
- **Release**: Exiting lexical scopes or reassignment invokes `__gat_release` (`dec qword ptr [ptr-16]`). When `ref_count == 0`, memory is freed via the OS heap manager.

---

## 5. Declarations

### 5.1 Functions
```gat
fn add(a: i64, b: i64) -> i64 {
    return a + b;
}
```

### 5.2 Structs & Classes
```gat
struct Point {
    x: i64;
    y: i64;
}

class LinkedListNode<T> {
    data: T;
    next: LinkedListNode<T>;
}
```

### 5.3 Enums
```gat
enum Result<T, E> {
    Ok(T),
    Err(E)
}
```

---

## 6. Statements & Control Flow

### 6.1 Variable Binding (`let`)
```gat
let x = 10;
let y: i64 = 20;
```

### 6.2 Conditionals (`if` / `else`)
```gat
if x > 10 {
    print("Greater\n");
} else if x == 10 {
    print("Equal\n");
} else {
    print("Lesser\n");
}
```

### 6.3 Loops (`while` and `for`)
- **While Loop**:
  ```gat
  let i = 0;
  while i < 10 {
      i = i + 1;
  }
  ```
- **Range For Loop**:
  ```gat
  for i in 0..10 {
      print("i: {i}\n");
  }
  ```
- **Iterator For Loop**:
  ```gat
  for item in items {
      print("item: {item}\n");
  }
  ```

### 6.4 Pattern Matching (`match`)
```gat
match opt {
    Option::Some(v) => {
        print("Value: {v}\n");
    }
    Option::None => {
        print("None\n");
    }
}
```

---

## 7. Expressions & Operators

- **Binary Arithmetic**: `+`, `-`, `*`, `/`, `%`
- **Comparison**: `==`, `!=`, `<`, `<=`, `>`, `>=`
- **Logical**: `&&`, `||`, `!`
- **Member Access**: `obj.field`
- **Instantiation**:
  ```gat
  let pt = new Point { x: 10, y: 20 };
  ```
- **String Interpolation**:
  ```gat
  let msg = "Hello {name}, score: {score * 2}";
  ```

---

## 8. Module & Import System

Source files import other modules using relative paths:
```gat
import "std/str.gat";
import "std/fs.gat";
import "std/math.gat";
```
Imports are resolved transitively and deduplicated at compile-time.

---

## 9. Grammar (EBNF)

```ebnf
Program         ::= ( ImportDecl | TopLevelDecl )*
ImportDecl      ::= 'import' StringLit ';'
TopLevelDecl    ::= FnDecl | StructDecl | ClassDecl | EnumDecl

FnDecl          ::= 'fn' Ident [ TypeParams ] '(' [ ParamList ] ')' [ '->' Type ] Block
StructDecl      ::= 'struct' Ident [ TypeParams ] '{' ( Ident ':' Type ';' )* '}'
ClassDecl       ::= 'class' Ident [ TypeParams ] '{' ( Ident ':' Type ';' )* '}'
EnumDecl        ::= 'enum' Ident [ TypeParams ] '{' ( EnumVariant ( ',' EnumVariant )* )? '}'
EnumVariant     ::= Ident [ '(' Type ')' ]

Block           ::= '{' Stmt* '}'
Stmt            ::= LetStmt | IfStmt | WhileStmt | ForStmt | MatchStmt | ReturnStmt | ExprStmt
LetStmt         ::= 'let' Ident [ ':' Type ] '=' Expr ';'
IfStmt          ::= 'if' Expr Block [ 'else' ( IfStmt | Block ) ]
WhileStmt       ::= 'while' Expr Block
ForStmt         ::= 'for' Ident 'in' ( Expr '..' Expr | Expr ) Block
MatchStmt       ::= 'match' Expr '{' MatchArm* '}'
MatchArm        ::= Ident '::' Ident [ '(' Ident ')' ] '=>' Block
ReturnStmt      ::= 'return' [ Expr ] ';'
ExprStmt        ::= Expr ';'

Expr            ::= BinaryExpr
BinaryExpr      ::= UnaryExpr ( BinaryOp UnaryExpr )*
UnaryExpr       ::= ( '-' | '!' ) UnaryExpr | PrimaryExpr
PrimaryExpr     ::= IntLit | StrLit | BoolLit | 'nil' | Ident | MemberExpr | CallExpr | NewExpr | '(' Expr ')'
NewExpr         ::= 'new' Ident [ TypeParams ] '{' ( Ident ':' Expr ( ',' Ident ':' Expr )* )? '}'
```
