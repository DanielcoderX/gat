# The `gat` Programming Language

`gat` is a compiled, low-level systems programming language featuring automatic reference counting (ARC), direct Windows x86-64 Portable Executable (PE32+) machine code emission, and a 100% bitwise-reproducible **self-hosted compiler**.

---

## Key Highlights

- **100% Self-Hosting**: The compiler in [`src/compiler.gat`](file:///c:/Users/Daniel/Desktop/gat/src/compiler.gat) compiles itself, generating identical binaries across all bootstrap stages (`gatc-v2 == gatc-v3 == gatc-v4`).
- **Zero External Toolchain Dependencies**: Emits standalone Windows PE (`.exe`) binaries directly without linkers (like `link.exe` or `lld`), assemblers (like `nasm`), or C runtimes (`MSVCRT`).
- **Dual Memory Model**:
  - `struct`: Stack-allocated value types (zero heap or refcount overhead).
  - `class`: Heap-allocated reference types managed via non-atomic ARC.
  - `deinit`: RAII-style destructor hooks executed automatically when reference count reaches zero.
  - `raw T`: Unsafe raw pointer escape hatch for memory buffers and systems programming.
- **Borrowed-by-Default Calling Convention**: Function parameters are borrowed by default, eliminating redundant retain/release cycles on calls.
- **Windows Fastcall ABI**: Conforms strictly to Microsoft x64 Fastcall (`RCX`, `RDX`, `R8`, `R9`, 32-byte shadow space, 16-byte stack alignment).

---

## Quick Start

### 1. Build the Bootstrap Compiler (Go)
Requires **Go 1.22+** on Windows x64.

```powershell
go build -o bin/gatc-v0.exe ./cmd/gatc
```

### 2. Compile and Run a Program
```powershell
# Compile hello.gat to hello.exe
.\bin\gatc-v0.exe examples/hello.gat -o bin/hello.exe

# Run the emitted executable
.\bin\hello.exe
```

---

## Language Guide & Tutorial

### 1. Basic Structure & Hello World

Every `gat` program contains function declarations and a `main` entry point:

```gat
fn main() -> i64 {
    print("Hello from gat!\n");
    return 0;
}
```

---

### 2. Primitive Types & Literals

| Type | Description | Literals / Examples |
|---|---|---|
| `i64` | Signed 64-bit integer | `0`, `42`, `-100`, `0x1000` |
| `bool` | Boolean | `true`, `false` |
| `string` | Null-terminated byte string | `"hello\n"`, `"gat"` |
| `void` | Empty return type | Functions without return value |
| `raw T` | Raw pointer to type `T` | `raw i8`, `raw Point` |

---

### 3. Variables & Assignment

Variables are declared with `let`. Types can be explicitly annotated or inferred:

```gat
fn variables_demo() {
    let a = 10;                     // inferred i64
    let b: i64 = 20;                // explicitly typed
    let msg: string = "compiler";   // string variable
    let flag: bool = true;          // bool variable

    a = a + b;                      // reassignment
    print("a + b = ", a, "\n");
}
```

---

### 4. Operators & Precedence

`gat` supports standard arithmetic, relational, and logical operators:

- **Arithmetic**: `+`, `-`, `*`, `/`, `%`
- **Comparisons**: `==`, `!=`, `<`, `<=`, `>`, `>=`
- **Logical**: `&&`, `||`, `!`
- **Bitwise / Memory**: `raw`, `[]` indexing

```gat
fn math_ops(x: i64, y: i64) -> i64 {
    let sum = x + y;
    let diff = x - y;
    let prod = x * y;
    let quot = x / y;
    let rem = x % y;

    if (x > 0 && y > 0) || !(x == 0) {
        return sum * prod;
    }
    return 0;
}
```

---

### 5. Control Flow

#### If / Else
```gat
fn max(a: i64, b: i64) -> i64 {
    if a > b {
        return a;
    } else {
        return b;
    }
}
```

#### While Loops
```gat
fn countdown(start: i64) {
    let i = start;
    while i > 0 {
        print("T-minus: ", i, "\n");
        i = i - 1;
    }
    print("Liftoff!\n");
}
```

---

### 6. Structs (Value Types)

`struct` types are stack-allocated and copied by value with zero heap allocation:

```gat
struct Point {
    x: i64;
    y: i64;
}

fn print_point(p: Point) {
    print("Point(", p.x, ", ", p.y, ")\n");
}

fn main() -> i64 {
    let pt = Point { x: 10, y: 25 };
    pt.x = pt.x + 5;
    print_point(pt);
    return 0;
}
```

---

### 7. Classes (Reference Types & ARC)

`class` types are heap-allocated via `new` and tracked by Automatic Reference Counting. You can define an optional `deinit` block that executes when the instance is reclaimed:

```gat
class Node {
    value: i64;
    next: Node;

    deinit {
        print("[deinit] Node freed with value = ", self.value, "\n");
    }
}

fn create_list() -> Node {
    let head = new Node { value: 1, next: nil };
    let second = new Node { value: 2, next: nil };
    head.next = second;
    return head;
}

fn main() -> i64 {
    let list = create_list();
    print("Head value: ", list.value, "\n");
    // ARC releases `list` and `second` automatically at scope exit
    return 0;
}
```

---

### 8. Unsafe Raw Pointers & Memory Management (`raw T`)

For low-level byte manipulation, dynamic buffers, or binary formats:

```gat
struct ByteBuffer {
    data: raw i8;
    size: i64;
    cap: i64;
}

fn bb_new(initial_cap: i64) -> ByteBuffer {
    let p: raw i8 = alloc_mem(initial_cap);
    return ByteBuffer {
        data: p,
        size: 0,
        cap: initial_cap
    };
}

fn bb_write_byte(bb: raw ByteBuffer, val: i64) {
    bb.data[bb.size] = val;
    bb.size = bb.size + 1;
}

fn bb_free(bb: ByteBuffer) {
    free_mem(bb.data);
}
```

---

### 9. Enums, Tagged Unions & Pattern Matching

`gat` supports algebraic data types (`enum`) with optional payload variants, and structural pattern matching via `match`:

```gat
enum Option {
    None,
    Some(i64)
}

enum Color {
    Red,
    Green,
    Blue
}

fn handle_option(opt: Option) {
    match opt {
        Option.Some(val) => {
            print("Received value: ", val, "\n");
        }
        Option.None => {
            print("Nothing here.\n");
        }
    }
}

fn handle_color(c: Color) {
    match c {
        Color.Red => { print("Red\n"); }
        Color.Green => { print("Green\n"); }
        Color.Blue => { print("Blue\n"); }
        _ => { print("Unknown\n"); }
    }
}
```

---

### 10. Fixed-Size Arrays & Indexing

Array literals `[e1, e2, ...]` allocate contiguous elements on the stack:

```gat
fn array_demo() {
    let arr = [10, 20, 30, 40];
    print("arr[0] = ", arr[0], "\n");
    print("arr[2] = ", arr[2], "\n");
}
```

---

### 11. Built-in Runtime Library

| Function | Signature | Description |
|---|---|---|
| `print(...)` | Variadic | Prints strings, integers, and booleans to stdout |
| `alloc_mem` | `fn alloc_mem(size: i64) -> raw i8` | Allocates zero-initialized heap memory via Windows `HeapAlloc` |
| `free_mem` | `fn free_mem(ptr: raw i8)` | Frees allocated memory via Windows `HeapFree` |
| `read_file` | `fn read_file(path: string) -> string` | Reads entire file into a null-terminated string |
| `write_file` | `fn write_file(path: string, data: raw i8, size: i64) -> i64` | Writes buffer to file via Windows `CreateFileA`/`WriteFile` |
| `str_len` | `fn str_len(s: string) -> i64` | Returns byte length of string |
| `str_eq` | `fn str_eq(a: string, b: string) -> bool` | Compares two strings for equality |
| `str_char` | `fn str_char(s: string, idx: i64) -> i64` | Returns byte character at index |
| `str_sub` | `fn str_sub(s: string, start: i64, len: i64) -> string` | Substring extraction |
| `str_concat` | `fn str_concat(a: string, b: string) -> string` | Concatenates two strings |
| `str_from_int` | `fn str_from_int(n: i64) -> string` | Formats 64-bit integer to decimal string |
| `get_cmd_arg` | `fn get_cmd_arg(idx: i64) -> string` | Parses command-line arguments |

---

## Self-Hosting Bootstrap & Verification

The `gat` compiler is written entirely in `gat` ([`src/compiler.gat`](file:///c:/Users/Daniel/Desktop/gat/src/compiler.gat)).

### 3-Stage Bootstrap Process

```
+-------------------+
|  gatc-v0 (Go)     | ---> compiles src/compiler.gat ---> bin/gatc-v1.exe
+-------------------+
                                                              |
+-------------------+                                         v
|  bin/gatc-v1.exe  | <---------------------------------------+
+-------------------+ ---> compiles src/compiler.gat ---> bin/gatc-v2.exe
                                                              |
+-------------------+                                         v
|  bin/gatc-v2.exe  | <---------------------------------------+
+-------------------+ ---> compiles src/compiler.gat ---> bin/gatc-v3.exe
                                                              |
                                                              v
                                                   fc.exe /b gatc-v2.exe gatc-v3.exe
                                                   (100% BITWISE IDENTICAL)
```

### Reproducing the Full Bootstrap Pipeline

Run the following commands in PowerShell:

```powershell
# Stage 1: Go bootstrap compiler builds gatc-v1.exe
go run ./cmd/gatc src/compiler.gat -o bin/gatc-v1.exe

# Stage 2: gatc-v1 (native PE) builds gatc-v2.exe
.\bin\gatc-v1.exe src/compiler.gat -o bin/gatc-v2.exe

# Stage 3: gatc-v2 (native PE) builds gatc-v3.exe
.\bin\gatc-v2.exe src/compiler.gat -o bin/gatc-v3.exe

# Verify 100% bitwise binary reproduction
fc.exe /b bin\gatc-v2.exe bin\gatc-v3.exe
```

Expected output:
```
Comparing files BIN\gatc-v2.exe and BIN\GATC-V3.EXE
FC: no differences encountered
```

---

## Directory Layout

```
gat/
├── bin/                 # Generated native executables (gatc-v1..v3, demos)
├── cmd/
│   └── gatc/            # Go v0 bootstrap compiler CLI
├── examples/            # Example .gat source files
│   ├── hello.gat        # Minimal hello world test
│   └── e2e_arc.gat      # ARC lifecycle and destructor test
├── pkg/                 # Go bootstrap compiler internals
│   ├── ast/             # AST node structures
│   ├── parser/          # Pratt recursive descent parser
│   ├── lexer/           # Tokenizer
│   ├── typecheck/       # Semantic analysis & type table
│   ├── ir/              # 3-address linear IR
│   ├── arc/             # Static reference counting insertion
│   ├── codegen/         # x86-64 machine code generator
│   └── pe/              # Direct PE32+ header and section builder
├── src/
│   └── compiler.gat     # Self-hosted compiler implementation in gat (3.2k lines)
└── README.md            # Language and toolchain documentation
```
