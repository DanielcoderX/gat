# Gat Standard Library Reference (`std/`)

Comprehensive documentation of all built-in modules in the Gat Standard Library.

---

## Table of Contents
1. [`std/option.gat` - Option Type & Combinators](#1-stdoptiongat---option-type--combinators)
2. [`std/result.gat` - Result Type & Error Handling](#2-stdresultgat---result-type--error-handling)
3. [`std/str.gat` - String, Formatted Output & StringBuilder](#3-stdstrgat---string-formatted-output--stringbuilder)
4. [`std/io.gat` - Input & Output](#4-stdiogat---input--output)
5. [`std/fs.gat` - File System Operations](#5-stdfsgat---file-system-operations)
6. [`std/math.gat` - Mathematical Functions](#6-stdmathgat---mathematical-functions)
7. [`std/process.gat` - Process & OS Utilities](#7-stdprocessgat---process--os-utilities)

---

## 1. `std/option.gat` - Option Type & Combinators

Import: `import "std/option.gat";`

```gat
enum Option<T> {
    None,
    Some(T)
}
```

### Functions

#### `option_some<T>(val: T) -> Option<T>`
Wraps `val` into `Option.Some(val)`.

#### `option_none<T>() -> Option<T>`
Returns `Option.None`.

#### `option_is_some<T>(opt: Option<T>) -> bool`
Returns `true` if `opt` is `Option.Some`.

#### `option_is_none<T>(opt: Option<T>) -> bool`
Returns `true` if `opt` is `Option.None`.

#### `option_unwrap<T>(opt: Option<T>) -> T`
Returns the wrapped value if `Some`, panics and aborts process if `None`.

#### `option_unwrap_or<T>(opt: Option<T>, default_val: T) -> T`
Returns the wrapped value if `Some`, otherwise returns `default_val`.

---

## 2. `std/result.gat` - Result Type & Error Handling

Import: `import "std/result.gat";`

```gat
enum Result<T, E> {
    Ok(T),
    Err(E)
}
```

### Functions

#### `result_ok<T, E>(val: T) -> Result<T, E>`
Constructs a successful `Result.Ok(val)`.

#### `result_err<T, E>(err: E) -> Result<T, E>`
Constructs an erroneous `Result.Err(err)`.

#### `result_is_ok<T, E>(res: Result<T, E>) -> bool`
Returns `true` if `res` is `Result.Ok`.

#### `result_is_err<T, E>(res: Result<T, E>) -> bool`
Returns `true` if `res` is `Result.Err`.

#### `result_unwrap<T, E>(res: Result<T, E>) -> T`
Unwraps the `Ok` value; panics and aborts process if `Err`.

#### `result_unwrap_or<T, E>(res: Result<T, E>, default_val: T) -> T`
Unwraps the `Ok` value, or returns `default_val` on `Err`.

#### `result_unwrap_err<T, E>(res: Result<T, E>) -> E`
Unwraps the `Err` payload; panics if called on `Result.Ok`.

---

## 3. `std/str.gat` - String, Formatted Output & StringBuilder

Import: `import "std/str.gat";`

### Functions

#### `str_parse_int(s: string) -> Result<i64, string>`
Parses a decimal integer from string `s`. Returns `Result.Ok(i64)` on success, or `Result.Err("description")` on non-digit / invalid characters.

#### `format1(fmt: string, a: string) -> string`
#### `format2(fmt: string, a: string, b: string) -> string`
#### `format3(fmt: string, a: string, b: string, c: string) -> string`
#### `format4(fmt: string, a: string, b: string, c: string, d: string) -> string`
Positional `{}` placeholder substitution into `fmt`.

#### `str_sub(s: string, start: i64, len: i64) -> string`
Extracts a substring starting at index `start` with length `len`.

#### `str_trim(s: string) -> string`
Removes leading and trailing ASCII whitespace characters.

#### `str_index_of(s: string, sub: string) -> i64`
Returns 0-based index of first occurrence of `sub`, or `-1` if not found.

#### `str_starts_with(s: string, prefix: string) -> bool`
Returns `true` if `s` begins with `prefix`.

#### `str_ends_with(s: string, suffix: string) -> bool`
Returns `true` if `s` ends with `suffix`.

### StringBuilder

#### `sb_new(capacity: i64) -> StringBuilder`
Initializes a new growable string buffer.

#### `sb_append(sb: StringBuilder, s: string)`
Appends string `s` to the builder buffer.

#### `sb_append_char(sb: StringBuilder, ch: i64)`
Appends ASCII character `ch` to the builder buffer.

#### `sb_to_string(sb: StringBuilder) -> string`
Constructs and returns an immutable `string` from the buffer contents.

---

## 4. `std/io.gat` - Input & Output

Import: `import "std/io.gat";`

### Functions

#### `io_read_text(path: string) -> Result<string, string>`
Reads full file contents into a `Result.Ok(string)` or `Result.Err(string)`.

#### `io_write_text(path: string, content: string) -> Result<i64, string>`
Writes string `content` to file `path`. Returns bytes written on success.

#### `io_append_text(path: string, content: string) -> Result<i64, string>`
Appends string `content` to file `path`.

#### `io_path_combine(dir: string, file: string) -> string`
#### `io_path_ext(path: string) -> string`
#### `io_path_filename(path: string) -> string`

---

## 5. `std/fs.gat` - File System Operations

Import: `import "std/fs.gat";`

### Functions

#### `fs_exists(path: string) -> bool`
Checks whether a file exists at the given path.

#### `fs_read_all(path: string) -> Result<string, string>`
Reads entire file contents into `Result.Ok(string)` or `Result.Err(string)` on failure.

#### `fs_write_all(path: string, content: string) -> Result<i64, string>`
Writes text content to destination file. Returns `Result.Ok(bytes_written)` or `Result.Err(string)`.

#### `fs_file_size(path: string) -> i64`
Returns the file size in bytes.

#### `fs_delete_file(path: string) -> bool`
Deletes specified file from disk.

#### `fs_temp_dir() -> string`
Returns system temporary directory path.

---

## 6. `std/math.gat` - Mathematical Functions

Import: `import "std/math.gat";`

### Functions

#### `math_abs(x: i64) -> i64`
Computes absolute value of `x`.

#### `math_min(a: i64, b: i64) -> i64`
Returns the smaller of two integers.

#### `math_max(a: i64, b: i64) -> i64`
Returns the larger of two integers.

#### `math_clamp(x: i64, min_v: i64, max_v: i64) -> i64`
Clamps `x` within inclusive range `[min_v, max_v]`.

#### `math_pow(base: i64, exp: i64) -> i64`
Computes integer exponentiation (`base ^ exp`) in $O(\log n)$ using binary exponentiation.

#### `math_sqrt(x: i64) -> i64`
Computes integer square root ($\lfloor\sqrt{x}\rfloor$) using Newton-Raphson method.

---

## 7. `std/process.gat` - Process & OS Utilities

Import: `import "std/process.gat";`

### Functions

#### `proc_exec(cmd: string) -> i64`
Spawns and executes a command line synchronously via the operating system process manager and returns the child exit code.

#### `proc_get_pid() -> i64`
Returns current process identifier (PID).

#### `proc_sleep(ms: i64)`
Suspends process execution for `ms` milliseconds.

#### `proc_exit(code: i64)`
Terminates the calling process with exit code `code`.
