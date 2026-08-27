# Gat Standard Library Reference (`std/`)

Comprehensive documentation of all built-in modules in the Gat Standard Library.

---

## Table of Contents
1. [`std/str.gat` - String & StringBuilder](#1-stdstrgat---string--stringbuilder)
2. [`std/io.gat` - Input & Output](#2-stdiogat---input--output)
3. [`std/fs.gat` - File System Operations](#3-stdfsgat---file-system-operations)
4. [`std/math.gat` - Mathematical Functions](#4-stdmathgat---mathematical-functions)
5. [`std/process.gat` - Process & OS Utilities](#5-stdprocessgat---process--os-utilities)

---

## 1. `std/str.gat` - String & StringBuilder

Import: `import "std/str.gat";`

### Functions

#### `str_sub(s: string, start: i64, len: i64) -> string`
Extracts a substring starting at index `start` with length `len`.

#### `str_split(s: string, sep: string) -> StringList`
Splits a string into dynamic linked segments separated by `sep`.

#### `str_trim(s: string) -> string`
Removes leading and trailing ASCII whitespace characters.

#### `str_index_of(s: string, sub: string) -> i64`
Returns 0-based index of first occurrence of `sub`, or `-1` if not found.

#### `str_contains(s: string, sub: string) -> bool`
Returns `true` if `sub` is present in `s`.

#### `str_replace(s: string, old_s: string, new_s: string) -> string`
Replaces all occurrences of `old_s` with `new_s`.

### StringBuilder

#### `sb_new(capacity: i64) -> StringBuilder`
Initializes a new growable string buffer.

#### `sb_append(sb: StringBuilder, s: string)`
Appends string `s` to the builder buffer.

#### `sb_to_string(sb: StringBuilder) -> string`
Constructs and returns an immutable `string` from the buffer contents.

---

## 2. `std/io.gat` - Input & Output

Import: `import "std/io.gat";`

### Intrinsic: `print(...)`
Variadic print statement accepting strings, integers, and boolean values directly to standard output.
```gat
print("Hello ", name, ", your ID is ", id, "\n");
```

---

## 3. `std/fs.gat` - File System Operations

Import: `import "std/fs.gat";`

### Functions

#### `fs_exists(path: string) -> bool`
Checks whether a file exists at the given path.

#### `fs_read_all(path: string) -> string`
Reads entire file contents into a string. Returns empty string on failure.

#### `fs_write_all(path: string, content: string) -> bool`
Writes text content to destination file. Returns `true` if successful.

#### `fs_file_size(path: string) -> i64`
Returns the file size in bytes.

#### `fs_delete_file(path: string) -> bool`
Deletes specified file from disk with automatic retry and sharing lock release handling.

#### `fs_temp_dir() -> string`
Returns system temporary directory path with trailing directory separator (e.g. `C:\Users\...\AppData\Local\Temp\`).

---

## 4. `std/math.gat` - Mathematical Functions

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

## 5. `std/process.gat` - Process & OS Utilities

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
