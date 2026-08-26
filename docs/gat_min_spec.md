# `gat-min` Language Specification (Bootstrap Subset)

`gat-min` is the minimal frozen subset of the `gat` programming language used to implement the self-hosted compiler `gatc-v1`.

---

## 1. Features Included in `gat-min`

### 1.1 Types
- **Primitives**: `i64`, `bool`, `void`, `string`
- **Value Structs**: `struct Name { f1: T; f2: U; }` (stack allocated, passed/copied by value)
- **Reference Classes**: `class Name { f1: T; ... deinit { ... } }` (heap allocated via `new`, non-atomic ARC managed, optional RAII `deinit`)
- **Unsafe Raw Pointers**: `raw T` (e.g. `raw i8`, `raw i64`, `raw Point`), supporting index access `ptr[idx]` and address-of `raw expr`

### 1.2 Declarations & Functions
- Top-level `struct`, `class`, `fn`
- Functions with typed parameters and explicit return types (`fn name(p1: T, p2: U) -> Ret`)
- Calling convention: **borrowed by default** for reference class parameters

### 1.3 Statements & Control Flow
- Variable declarations: `let name = expr;` / `let name: Type = expr;`
- Variable & field assignments: `x = expr;`, `obj.field = expr;`, `buf[idx] = expr;`
- Conditional branching: `if (cond) { ... } else { ... }`
- Iteration: `while (cond) { ... }`
- Function returns: `return;` / `return expr;`
- Expressions as statements: function calls, `print(...)`

### 1.4 Operators & Expressions
- Arithmetic: `+`, `-`, `*`, `/`, `%`, unary `-`
- Relational & Equality: `==`, `!=`, `<`, `<=`, `>`, `>=`
- Logical: `&&`, `||`, `!`
- Member Access: `obj.field`
- Index Access: `buf[index]` (loads byte if `raw i8`/`string`, 8 bytes if `raw i64`/general pointer)
- Instantiation: `new ClassName { field: val, ... }` and `StructName { field: val, ... }`
- Literals: decimal integers (`123`), hex integers (`0x7F`), character literals (`'a'`, `'\n'`), strings (`"..."`), booleans (`true`, `false`), `nil`

### 1.5 Builtin Runtime Intrinsics
- `print(args...)`: Output strings, numbers, booleans
- `read_file(path: string) -> string`: Read whole file from disk
- `write_file(path: string, data: string, len: i64) -> i64`: Write bytes to disk
- `alloc_mem(bytes: i64) -> raw i8`: Allocate raw heap buffer
- `free_mem(ptr: raw i8)`: Free raw heap buffer
- `str_len(s: string) -> i64`: Get string length
- `str_eq(a: string, b: string) -> bool`: Compare two strings
- `str_char(s: string, idx: i64) -> i64`: Read ASCII byte at index
- `str_sub(s: string, start: i64, len: i64) -> string`: Substring extraction
- `str_concat(a: string, b: string) -> string`: Concatenate two strings
- `str_from_int(n: i64) -> string`: Convert integer to string
- `get_cmd_arg(idx: i64) -> string`: Retrieve command line argument

---

## 2. Deliberately Excluded Features (Deferred Post-Bootstrap)

- Generics / Type parameters
- Closures / First-class anonymous functions
- Pattern matching (`match` / `switch`)
- Operator overloading
- Trait / Interface inheritance
- Multithreading & Concurrency primitives
- Automatic cycle collection
