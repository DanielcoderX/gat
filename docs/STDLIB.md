---
layout: default
title: Standard Library
nav_order: 5
description: "Reference documentation for the Gat Standard Library modules."
permalink: /STDLIB.html
---

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
8. [`std/net.gat` - Sockets & TCP Networking](#8-stdnetgat---sockets--tcp-networking)
9. [`std/json.gat` - JSON Parser, Serializer & Data Types](#9-stdjsongat---json-parser-serializer--data-types)

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

---

## 8. `std/net.gat` - Sockets & TCP Networking

Import: `import "std/net.gat";`

Provides cross-platform socket primitives and high-level TCP abstractions powered by direct kernel syscalls on Linux (x86-64 `sys_socket`, `sys_connect`, `sys_bind`, `sys_listen`, `sys_accept`, `sys_sendto`, `sys_recvfrom`, `sys_close`) and dynamic Winsock (`ws2_32.dll`) resolution on Windows.

### Classes

#### `class TcpStream`
Manages an active client or inbound connection socket. Automatically invokes `net_close` on deallocation (`deinit`).
* Field `fd: i64` - OS file descriptor or socket handle.

#### `class TcpListener`
Manages a listening TCP server socket. Automatically invokes `net_close` on deallocation (`deinit`).
* Field `fd: i64` - OS server socket handle.

### High-Level TCP Functions

#### `tcp_connect(ip: string, port: i64) -> TcpStream`
Connects to remote IPv4 host `ip` on `port`. Returns an owning `TcpStream` on success, or `nil` on failure.

#### `tcp_listen(ip: string, port: i64) -> TcpListener`
Binds to IPv4 host `ip` on `port` and starts listening for connections with a backlog queue of 128. If `ip` is `""` or `"0.0.0.0"`, binds to all interfaces (`INADDR_ANY`). Returns `TcpListener` on success, or `nil` on failure.

#### `tcp_accept(listener: TcpListener) -> TcpStream`
Accepts the next incoming connection from `listener`. Returns a new `TcpStream` for bidirectional communication, or `nil` on error.

#### `tcp_send(stream: TcpStream, text: string) -> i64`
Sends string `text` across `stream`. Returns number of bytes sent or negative error code.

#### `tcp_recv(stream: TcpStream, max_len: i64) -> string`
Receives up to `max_len` bytes from `stream` into a newly allocated string. Returns received string, or empty string `""` on EOF or failure.

#### `tcp_close(stream: TcpStream)`
Explicitly closes client stream before scope exit.

#### `tcp_listener_close(listener: TcpListener)`
Explicitly closes server listener socket.

### Low-Level Socket Intrinsics

* `net_af_inet() -> i64` - Returns IPv4 address family constant (`AF_INET = 2`).
* `net_sock_stream() -> i64` - Returns stream socket type (`SOCK_STREAM = 1`).
* `net_sock_dgram() -> i64` - Returns datagram socket type (`SOCK_DGRAM = 2`).
* `net_ipproto_tcp() -> i64` - Returns TCP protocol constant (`IPPROTO_TCP = 6`).
* `net_ipproto_udp() -> i64` - Returns UDP protocol constant (`IPPROTO_UDP = 17`).
* `net_socket(domain: i64, kind: i64, proto: i64) -> i64` - Allocates OS socket.
* `net_close(fd: i64) -> i64` - Closes OS socket.

---

## 9. `std/json.gat` - JSON Parser, Serializer & Data Types

Import: `import "std/json.gat";`

RFC 8259 compliant JSON parser, serializer, and data model. Provides dynamic DOM manipulation, compact and indented pretty-printing, and streaming struct/class serialization.

```gat
enum JsonValue {
    Null,
    Bool(bool),
    Number(i64),
    String(string),
    Array(Vector<JsonValue>),
    Object(JsonObject)
}
```

### Value Constructors

* `json_null() -> JsonValue` - Returns JSON `null`.
* `json_bool(b: bool) -> JsonValue` - Returns JSON boolean (`true` or `false`).
* `json_number(n: i64) -> JsonValue` - Returns JSON integer/number.
* `json_string(s: string) -> JsonValue` - Returns JSON string.
* `json_array() -> JsonValue` - Returns a new empty JSON array.
* `json_object() -> JsonValue` - Returns a new empty JSON key-value object.

### Type Query & Extraction

* `json_is_null(v: JsonValue) -> bool`
* `json_is_bool(v: JsonValue) -> bool`
* `json_is_number(v: JsonValue) -> bool`
* `json_is_string(v: JsonValue) -> bool`
* `json_is_array(v: JsonValue) -> bool`
* `json_is_object(v: JsonValue) -> bool`
* `json_as_bool(v: JsonValue) -> bool`
* `json_as_int(v: JsonValue) -> i64`
* `json_as_string(v: JsonValue) -> string`
* `json_as_array(v: JsonValue) -> Vector<JsonValue>`
* `json_as_object(v: JsonValue) -> JsonObject`

### Array Operations

* `json_arr_len(v: JsonValue) -> i64` - Returns number of elements in array.
* `json_arr_get(v: JsonValue, idx: i64) -> JsonValue` - Gets element at index or `nil`.
* `json_arr_set(v: JsonValue, idx: i64, item: JsonValue)` - Updates element at index.
* `json_arr_set_str(v: JsonValue, idx: i64, s: string)` - Sets string at index.
* `json_arr_set_int(v: JsonValue, idx: i64, n: i64)` - Sets integer at index.
* `json_arr_set_bool(v: JsonValue, idx: i64, b: bool)` - Sets boolean at index.
* `json_arr_push(v: JsonValue, item: JsonValue)` - Appends value.
* `json_arr_push_str(v: JsonValue, s: string)` - Appends string.
* `json_arr_push_int(v: JsonValue, n: i64)` - Appends number.
* `json_arr_push_bool(v: JsonValue, b: bool)` - Appends boolean.
* `json_arr_remove_at(v: JsonValue, idx: i64) -> JsonValue` - Removes and returns element at index.

### Object Operations

* `json_set(v: JsonValue, key: string, val: JsonValue)` - Sets key-value pair.
* `json_set_str(v: JsonValue, key: string, s: string)` - Sets string field.
* `json_set_int(v: JsonValue, key: string, n: i64)` - Sets integer field.
* `json_set_bool(v: JsonValue, key: string, b: bool)` - Sets boolean field.
* `json_set_null(v: JsonValue, key: string)` - Sets field to `null`.
* `json_remove(v: JsonValue, key: string) -> bool` - Removes field from object.
* `json_get(v: JsonValue, key: string) -> JsonValue` - Retrieves field or `nil`.
* `json_get_str(v: JsonValue, key: string, default_val: string) -> string`
* `json_get_int(v: JsonValue, key: string, default_val: i64) -> i64`
* `json_get_bool(v: JsonValue, key: string, default_val: bool) -> bool`
* `json_get_arr(v: JsonValue, key: string) -> Vector`
* `json_get_obj(v: JsonValue, key: string) -> JsonObject`
* `json_has(v: JsonValue, key: string) -> bool` - Checks if key exists.

### Hierarchical Path Navigation & Editing

Easily read and modify deeply nested structures using dot notation (e.g., `"server.database.port"` or `"items.0.name"`):

* `json_get_path(v: JsonValue, path: string) -> JsonValue`
* `json_get_path_str(v: JsonValue, path: string, default_val: string) -> string`
* `json_get_path_int(v: JsonValue, path: string, default_val: i64) -> i64`
* `json_get_path_bool(v: JsonValue, path: string, default_val: bool) -> bool`
* `json_set_path(root: JsonValue, path: string, val: JsonValue) -> bool`
* `json_set_path_str(root: JsonValue, path: string, s: string) -> bool`
* `json_set_path_int(root: JsonValue, path: string, n: i64) -> bool`
* `json_set_path_bool(root: JsonValue, path: string, b: bool) -> bool`

### File I/O & In-Place File Editing

Load, inspect, modify, and persist JSON documents directly on disk:

#### `json_read_file(path: string) -> Result<JsonValue, string>`
Reads a JSON file from disk and parses it. Returns `Result.Ok(val)` on success or `Result.Err(msg)` on file I/O or syntax error.

#### `json_write_file(path: string, v: JsonValue) -> Result<bool, string>`
Writes a `JsonValue` tree to a file in compact format.

#### `json_write_file_pretty(path: string, v: JsonValue, indent_size: i64) -> Result<bool, string>`
Writes a formatted, indented JSON file with a trailing newline.

#### Example: Read, Edit, and Save a Config File

```gat
import "std/json.gat";
import "std/result.gat";

fn update_server_config(config_path: string) -> bool {
    // 1. Read JSON file from disk
    let res = json_read_file(config_path);
    if result_is_err(res) {
        print("Failed to read config: ", result_unwrap_err(res), "\n");
        return false;
    }
    let config = result_unwrap(res);

    // 2. Modify values using direct mutators or dot paths
    json_set_int(config, "port", 8080);
    json_set_path_str(config, "database.host", "db.internal");
    json_set_path_int(config, "database.pool_size", 32);

    // 3. Save modified JSON back to disk
    let save_res = json_write_file_pretty(config_path, config, 2);
    if result_is_err(save_res) {
        print("Failed to save config: ", result_unwrap_err(save_res), "\n");
        return false;
    }

    return true;
}
```

### Parsing & Serialization

#### `json_parse(input: string) -> Result<JsonValue, string>`
Parses a JSON string into an owning `JsonValue` tree. Returns `Result.Ok(val)` on success, or `Result.Err(msg)` with exact line and column diagnostics on syntax errors.

#### `json_stringify(v: JsonValue) -> string`
Serializes a `JsonValue` tree into a compact JSON string without extra whitespace.

#### `json_stringify_pretty(v: JsonValue, indent_size: i64) -> string`
Serializes a `JsonValue` tree into formatted, indented, human-readable JSON.

### Streaming Struct Serializer (`JsonSerializer`)

For high-performance serialization of custom structs and classes:

```gat
struct JsonSerializer { ... }

fn user_to_json(u: User) -> string {
    let ser = json_ser_new();
    json_ser_begin_object(ser);
    json_ser_kv_int(ser, "id", u.id);
    json_ser_kv_string(ser, "name", u.name);
    json_ser_kv_bool(ser, "active", u.active);
    json_ser_end_object(ser);
    return json_ser_finish(ser);
}
```

* `json_ser_new() -> JsonSerializer`
* `json_ser_begin_object(ser: JsonSerializer)`
* `json_ser_end_object(ser: JsonSerializer)`
* `json_ser_begin_array(ser: JsonSerializer)`
* `json_ser_end_array(ser: JsonSerializer)`
* `json_ser_key(ser: JsonSerializer, key: string)`
* `json_ser_string(ser: JsonSerializer, val: string)`
* `json_ser_int(ser: JsonSerializer, val: i64)`
* `json_ser_bool(ser: JsonSerializer, val: bool)`
* `json_ser_null(ser: JsonSerializer)`
* `json_ser_kv_string(ser: JsonSerializer, key: string, val: string)`
* `json_ser_kv_int(ser: JsonSerializer, key: string, val: i64)`
* `json_ser_kv_bool(ser: JsonSerializer, key: string, val: bool)`
* `json_ser_kv_null(ser: JsonSerializer, key: string)`
* `json_ser_finish(ser: JsonSerializer) -> string`



