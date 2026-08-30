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

### 3.4 Function Types & First-Class Functions
Gat supports **first-class function values (function pointers)**:
- **Type Syntax**: `fn(ParamType1, ParamType2, ...) -> ReturnType`
- **Function Values**: Top-level function declarations can be referenced as values without call parentheses (`let f = add;`).
- **Indirect Calls**: Functions stored in variables, passed as arguments, or stored as struct/class fields are called indirectly via machine code `call reg` (Windows fastcall ABI compliant).
- **Zero Overhead**: Function values are represented as 64-bit code addresses requiring no runtime allocations or ARC tracking.

```gat
fn apply<T, U>(f: fn(T) -> U, val: T) -> U {
    return f(val);
}

fn double_num(x: i64) -> i64 {
    return x * 2;
}

let result = apply(double_num, 21); // 42
```

---

## 4. Memory Model & ARC

### 4.1 Header Layout
All heap-allocated objects (classes, strings, arrays) are prefixed with a 24-byte metadata header:
```
Offset -24: [ strong_count : i64 ]   (64-bit strong reference counter)
Offset -16: [ weak_count   : i64 ]   (64-bit weak reference counter)
Offset  -8: [ type_tag     : i64 ]   (64-bit type discriminator / size)
Offset   0: [ User Payload       ]   <-- Object pointer returned to user
```

### 4.2 Retain & Release Semantics
- **Allocation**: `new Class { ... }` or `alloc_mem(sz)` initializes `strong_count = 1, weak_count = 0`.
- **Strong Retain**: Passing or assigning heap references invokes `__gat_retain` (`inc qword ptr [ptr-24]`).
- **Strong Release**: Reassignment or explicit release invokes `__gat_release` (`dec qword ptr [ptr-24]`). When `strong_count == 0`, deterministically runs `deinit` and releases child references. If `weak_count == 0`, immediately deallocates backing memory.
- **Weak Retain / Release**: Creating a `weak T` reference increments `weak_count` (`[ptr-16]`). Dropping a `weak T` reference decrements `weak_count`. If both `strong_count == 0` and `weak_count == 0`, backing memory is freed.

### 4.3 Weak References & Cycle Breaking (`weak T`)
Reference cycles (such as parent-child tree links or doubly linked lists) are resolved using `weak T` back-references:
- **Modifier**: `weak T` represents a non-owning reference to a class `T`.
- **Creation**: `weak_from(obj)` creates a `weak T` and increments `weak_count`.
- **Upgrade**: `weak_upgrade(w: weak T) -> Option<T>` atomically inspects `strong_count`. If alive (`> 0`), retains and returns `Option.Some(obj)`. If dead (`== 0`), returns `Option.None`.

```gat
class Node {
    next: Node;
    prev: weak Node; // Back-pointer does not keep parent alive, preventing cycle leak
}
```

### 4.4 Concurrency & Thread-Boundary Isolation
Gat provides native OS threading via `std/thread.gat` while preserving non-atomic ARC performance and memory soundness through compile-time thread-boundary isolation:
- **Thread-Local Heaps**: Each thread manages its own reference-counted heap. Reference counts (`strong_count`, `weak_count`) remain non-atomic and fast.
- **Thread-Boundary Safety**: The compiler enforces at compile time that reference-counted types (`class`, `string`, `weak T`, or structs containing them) cannot be passed across thread boundaries in `thread_spawn`.
- **Value & Raw Data Sharing**: Threads can receive primitive data (`i64`, `bool`), value `struct`s containing only plain data, and explicit `raw T` pointers.
- **Synchronization (`Mutex`)**: `std/sync.gat` provides `Mutex` (wrapping Win32 Critical Sections) for safe, serialized mutation of shared `raw T` state across threads.

```gat
import "std/thread.gat";
import "std/sync.gat";

struct Task {
    id: i64;
    out_ptr: raw i64;
}

fn worker(task: raw Task) {
    task.out_ptr[0] = task.id * 10;
}

let out: raw i64 = alloc_mem(8);
let t = new Task { id: 5, out_ptr: out };
let h = thread_spawn(worker, raw t);
thread_join(h);
// out[0] == 50
```

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

### 5.4 Modules & Namespaces (`import ... as ...`)
```gat
import "std/str.gat";                 // Spliced relative import
import "pkg/math.gat" as math;        // Namespaced module import
import "dep_json" as json;            // Cached dependency in .gat/deps/

let res = math.add(1, 2);
let pt = new math.Point { x: 10, y: 20 };
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
