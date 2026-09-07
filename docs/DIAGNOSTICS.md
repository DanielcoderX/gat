---
layout: default
title: Diagnostics & Error Catalog
nav_order: 10
description: "Comprehensive catalog of Gat compiler diagnostic error codes, root causes, and corrective examples."
permalink: /DIAGNOSTICS.html
---

# Gat Compiler Diagnostics & Error Catalog

**Version:** 1.0.0  
**Tooling:** `gatc --explain <code|list>` / `gat explain <code|list>`  

This reference catalog documents all standard diagnostic error codes emitted by the `gat` parser and typechecker. Each entry explains the invariant being violated, the compiler's rationale, and practical before/after code examples.

---

## Index of Error Codes

| Error Code | Category | Name | Diagnostic Summary |
|---|---|---|---|
| [E0001](#error-e0001-syntax-error) | Syntax | Syntax Error | Unexpected token or missing delimiter |
| [E0002](#error-e0002-undeclared-identifier) | Semantic | Undeclared Identifier | Variable or constant used without declaration |
| [E0003](#error-e0003-undeclared-function) | Semantic | Undeclared Function | Call to unknown symbol or unimported function |
| [E0004](#error-e0004-function-arity-mismatch) | Semantic | Function Arity Mismatch | Argument count does not match function signature |
| [E0005](#error-e0005-invalid-member-access) | Semantic | Invalid Member Access | Field name not found on struct or class |
| [E0006](#error-e0006-thread-boundary-safety-violation) | Safety | Thread Boundary Safety Violation | Passing reference-counted type across thread boundary |
| [E0007](#error-e0007-void-return-value-mismatch) | Semantic | Void Return Value Mismatch | Void procedure returning an expression |
| [E0008](#error-e0008-duplicate-symbol-across-imports) | Semantic | Duplicate Symbol Across Imports | Flat import symbol collision |
| [E0009](#error-e0009-unknown-type-in-instantiation) | Semantic | Unknown Type in Instantiation | `new` used with undeclared type name |
| [E0010](#error-e0010-assignment-to-undeclared-identifier) | Semantic | Assignment to Undeclared Identifier | Variable assigned before `let` declaration |

---

## Error [E0001]: Syntax Error

### Description
The parser encountered an unexpected token that violates Gat grammar. This occurs when an expected punctuation mark (such as a colon, semicolon, parenthesis, or brace) is missing or misplaced, or when an expression is expected but an operator or delimiter was found instead.

### Panic-Mode Error Recovery
In `gat` v1.0, the parser implements **panic-mode multi-error recovery**. Rather than aborting parsing on the first syntax error, the compiler records the error and advances tokens until reaching the next statement boundary (`;`, `}` or statement keyword) or top-level declaration (`fn`, `struct`, `class`, `enum`, `import`). This allows reporting multiple independent syntax mistakes in a single build invocation.

### Common Causes
1. Missing colon in a parameter definition: `fn f(a i64)` instead of `fn f(a: i64)`
2. Missing assignment value: `let x = ;`
3. Mismatched parentheses in conditionals or loops.

### Examples

#### Invalid
```gat
fn compute(val i64) -> i64 {
    let result = ;
    return result;
}
```

#### Corrected
```gat
fn compute(val: i64) -> i64 {
    let result = val * 2;
    return result;
}
```

---

## Error [E0002]: Undeclared Identifier

### Description
An identifier was referenced in an expression, but no local variable, global constant, or type with that name has been registered in the current scope.

### Common Causes
1. Typo in a variable name.
2. Referencing a variable before its `let` binding.
3. Accessing a variable defined inside an inner block scope that has already terminated.

### Examples

#### Invalid
```gat
fn process_items() -> i64 {
    let total = count + 10; // 'count' was never declared
    return total;
}
```

#### Corrected
```gat
fn process_items() -> i64 {
    let count = 5;
    let total = count + 10;
    return total;
}
```

---

## Error [E0003]: Undeclared Function

### Description
A function call was invoked on a symbol that does not match any declared top-level function, local function pointer, standard library builtin, or imported module alias.

### Common Causes
1. Missing `import` statement for the module providing the function.
2. Calling a function without its module namespace alias (e.g. calling `str_len` instead of `str.str_len`).
3. Typo in function name.

### Examples

#### Invalid
```gat
import "std/str.gat" as str;

fn main() -> i64 {
    let len = str_len("hello"); // 'str_len' is in module 'str'
    return len;
}
```

#### Corrected
```gat
import "std/str.gat" as str;

fn main() -> i64 {
    let len = str.str_len("hello");
    return len;
}
```

---

## Error [E0004]: Function Arity Mismatch

### Description
A function was invoked with an argument count that differs from its parameter list definition.

### Common Causes
1. Passing extra or missing arguments to a function.
2. Modifying a function signature in a library without updating all call sites.

### Examples

#### Invalid
```gat
fn calculate_area(width: i64, height: i64) -> i64 {
    return width * height;
}

let area = calculate_area(10, 20, 30); // Error: expects 2 arguments, got 3
```

#### Corrected
```gat
let area = calculate_area(10, 20);
```

---

## Error [E0005]: Invalid Member Access

### Description
A member access expression (`object.field`) references a field name that does not exist in the definition of the struct or class.

### Common Causes
1. Typo in field name.
2. Accessing a field of a nested object without dereferencing the intermediate member.

### Examples

#### Invalid
```gat
struct Point {
    x: i64;
    y: i64;
}

let p = new Point { x: 10, y: 20 };
let z_val = p.z; // Error: type 'Point' has no field named 'z'
```

#### Corrected
```gat
let x_val = p.x;
```

---

## Error [E0006]: Thread Boundary Safety Violation

### Description
An attempt was made to pass a reference-counted type (`class`, `string`, `weak T`, or a `struct` containing any of them) across a thread boundary in `thread_spawn`.

### Safety Rationale
To guarantee maximum single-threaded performance without lock contention, Gat's ARC counters (`strong_count` and `weak_count`) are non-atomic 64-bit integers. If reference-counted objects were shared across OS threads, concurrent increments and decrements would introduce data races and memory leaks/corruptions.

### Allowed Thread Arguments
- Primitives: `i64`, `bool`
- Plain structs containing only primitives or raw pointers
- Explicit raw pointers (`raw T`) synchronized with `std/sync.gat` `Mutex`

### Examples

#### Invalid
```gat
import "std/thread.gat";

class TaskData {
    name: string;
}

fn worker(data: TaskData) {
    // ...
}

let d = new TaskData { name: "job" };
thread_spawn(worker, d); // REJECTED: cannot pass reference-counted class across threads
```

#### Corrected
```gat
import "std/thread.gat";

struct WorkerParams {
    job_id: i64;
    out_ptr: raw i64;
}

fn worker(params: raw WorkerParams) {
    params.out_ptr[0] = params.job_id * 10;
}

let out: raw i64 = alloc_mem(8);
let p = new WorkerParams { job_id: 42, out_ptr: out };
let t = thread_spawn(worker, raw p);
thread_join(t);
```

---

## Error [E0007]: Void Return Value Mismatch

### Description
A function explicitly declared with return type `void` attempted to return an expression (`return <expr>;`).

### Common Causes
1. Function intended to return a value, but return type annotation was omitted (defaults to `void`).
2. Accidentally writing `return val;` in a procedural cleanup function.

### Examples

#### Invalid
```gat
fn log_status(code: i64) -> void {
    return code; // Error: void function cannot return a value
}
```

#### Corrected
```gat
fn log_status(code: i64) -> i64 {
    return code;
}
```

---

## Error [E0008]: Duplicate Symbol Across Imports

### Description
Two or more imported source files declare a top-level function or type with the identical identifier in the root flat namespace.

### Resolution
Use **namespaced module imports** (`import "path.gat" as alias;`) to isolate module symbols into distinct namespaces.

### Examples

#### Invalid
```gat
// Both a.gat and b.gat define 'fn helper()'
import "a.gat";
import "b.gat"; // Error: duplicate declaration of function 'helper' across flat imports
```

#### Corrected
```gat
import "a.gat" as a;
import "b.gat" as b;

a.helper();
b.helper();
```

---

## Error [E0009]: Unknown Type in Instantiation

### Description
An instantiation expression `new TypeName { ... }` specifies a type name that has not been declared as a `struct`, `class`, or `enum`.

### Examples

#### Invalid
```gat
let obj = new NonExistentType { x: 1 }; // Error: unknown type in instantiation
```

#### Corrected
```gat
struct ExistingType { x: i64; }
let obj = new ExistingType { x: 1 };
```

---

## Error [E0010]: Assignment to Undeclared Identifier

### Description
An assignment statement `identifier = value;` references an identifier that has not been declared with `let` in the current scope.

### Resolution
Declare the variable with `let` prior to assigning to it.

### Examples

#### Invalid
```gat
fn main() -> i64 {
    target_var = 42; // Error: assignment to undeclared identifier
    return target_var;
}
```

#### Corrected
```gat
fn main() -> i64 {
    let target_var = 42;
    return target_var;
}
```
