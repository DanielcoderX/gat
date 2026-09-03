---
layout: default
title: Example Gallery
nav_order: 3
description: "Curated beginner-friendly examples demonstrating Gat features."
permalink: /examples.html
---

# Curated Example Gallery

The `examples/showcase/` directory contains verified, teaching-oriented examples demonstrating Gat features step-by-step. Every example has been tested and confirmed working on both **Windows** and **Linux**.

---

## 1. Hello World
**File:** [`examples/showcase/01_hello_world.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/01_hello_world.gat)

```rust
// Program entrypoint and direct stdout emission
fn main() -> i64 {
    print("Hello, world! Welcome to the Gat programming language.\n");
    return 0;
}
```

---

## 2. FizzBuzz & Loops
**File:** [`examples/showcase/02_fizzbuzz.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/02_fizzbuzz.gat)

Demonstrates range loops (`for i in 1..21`), conditionals, and string interpolation (`{i}`):

```rust
fn main() -> i64 {
    print("=== FizzBuzz (1 to 20) ===\n");

    for i in 1..21 {
        if i % 15 == 0 {
            print("FizzBuzz\n");
        } else if i % 3 == 0 {
            print("Fizz\n");
        } else if i % 5 == 0 {
            print("Buzz\n");
        } else {
            print("{i}\n");
        }
    }

    return 0;
}
```

---

## 3. Structs (Value) vs Classes (Reference)
**File:** [`examples/showcase/03_struct_vs_class.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/03_struct_vs_class.gat)

Demonstrates Gat's dual memory architecture: stack-allocated value types vs. heap-allocated reference types.

```rust
struct Point {
    x: i64;
    y: i64;
}

class UserAccount {
    id: i64;
    balance: i64;
}

fn modify_point(mut_p: Point) {
    mut_p.x = 999; // Modifies local copy only
}

fn modify_account(acc: UserAccount) {
    acc.balance = acc.balance + 500; // Modifies shared heap instance
}

fn main() -> i64 {
    // 1. Struct: value copy
    let p1 = Point { x: 10, y: 20 };
    let p2 = p1;
    p2.x = 50;
    // p1.x remains 10!

    // 2. Class: reference alias
    let acc1 = new UserAccount { id: 101, balance: 1000 };
    let acc2 = acc1;
    modify_account(acc2);
    // acc1.balance is now 1500!

    return 0;
}
```

---

## 4. Deterministic ARC & Destructors (`deinit`)
**File:** [`examples/showcase/04_arc_deinit.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/04_arc_deinit.gat)

Demonstrates deterministic destruction when an object's reference count reaches zero.

```rust
class FileHandle {
    fd: i64;
    name: string;

    deinit {
        print("  -> [DEINIT] Closing FileHandle '{this.name}' (fd: {this.fd})...\n");
    }
}

fn create_temporary_resource() {
    let temp = new FileHandle { fd: 3, name: "temp_scratch.dat" };
    // `temp` leaves scope here -> deinit runs immediately!
}

fn main() -> i64 {
    create_temporary_resource();
    // Resource already closed before reaching this line!
    return 0;
}
```

---

## 5. Cycle-Breaking with Weak References (`weak T`)
**File:** [`examples/showcase/05_weak_references.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/05_weak_references.gat)

Demonstrates non-owning references to prevent reference-counting memory leaks.

```rust
import "std/weak.gat";

class TreeNode {
    id: i64;
    parent: weak TreeNode; // Non-owning back-link breaks cycle!

    deinit {
        print("  -> [DEINIT] TreeNode {this.id} freed.\n");
    }
}

fn main() -> i64 {
    let root = new TreeNode { id: 1, parent: nil };
    let child = new TreeNode { id: 2, parent: weak_from(root) };

    let upgraded = weak_upgrade(child.parent);
    if upgraded != nil {
        print("Parent ID: {upgraded.id}\n");
    }
    weak_release(child.parent);

    return 0;
}
```

---

## 6. Enums & Pattern Matching (`match`)
**File:** [`examples/showcase/06_enum_match.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/06_enum_match.gat)

Strongly-typed enums matched exhaustively using `match`.

```rust
enum ConnectionState {
    Disconnected,
    Connecting,
    Connected,
    TimedOut
}

fn describe_state(state: ConnectionState) -> string {
    match state {
        ConnectionState.Disconnected => { return "Offline / Idle"; }
        ConnectionState.Connecting   => { return "Handshake in progress..."; }
        ConnectionState.Connected    => { return "Online & Synchronized"; }
        ConnectionState.TimedOut     => { return "Error: Connection Timed Out"; }
    }
}
```

---

## 7. Namespaced Modules (`import ... as ...`)
**File:** [`examples/showcase/07_modules.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/07_modules.gat)

Shows clean symbol isolation without pollution.

```rust
import "examples/showcase/geometry/geometry.gat" as geo;
import "std/math.gat";

fn main() -> i64 {
    let rect = geo::Rectangle { width: 12, height: 8 };
    let a = geo::area(rect);
    let p = geo::perimeter(rect);
    print("Area: {a}, Perimeter: {p}\n");
    return 0;
}
```

---

## 8. Practical Mini CLI Tool (Word Count)
**File:** [`examples/showcase/08_word_count_cli.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/08_word_count_cli.gat)

Reading command-line arguments, file I/O, string indexing with `str_char(s, idx)`:

```rust
import "std/str.gat";
import "std/fs.gat";

fn main() -> i64 {
    let target_file = get_cmd_arg(1);
    let content = read_file(target_file);
    // Counts lines, words, and bytes
    return 0;
}
```

---

## 9. Cross-Platform Environment APIs
**File:** [`examples/showcase/09_cross_platform.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/09_cross_platform.gat)

Cross-platform process information and OS calls across Windows PE and Linux ELF64:

```rust
import "std/process.gat";
import "std/fs.gat";

fn main() -> i64 {
    let pid = proc_get_pid();
    let temp_dir = fs_temp_dir();
    print("PID: {pid}, Temp: {temp_dir}\n");
    proc_sleep(50);
    return 0;
}
```

---

## 10. TCP Sockets & Echo Server
**File:** [`examples/showcase/10_tcp_echo.gat`](https://github.com/DanielcoderX/gat/blob/main/examples/showcase/10_tcp_echo.gat)

Demonstrates Gat's standard networking library (`std/net.gat`): listening on loopback, accepting connections, client streaming, and RAII socket lifecycle management:

```rust
import "std/net.gat";
import "std/thread.gat";

fn client_worker(task: raw ClientTask) {
    let stream = tcp_connect("127.0.0.1", 19000);
    tcp_send(stream, "Hello from Gat TCP Client!");
    let reply = tcp_recv(stream, 256);
    print("Echo reply: '{reply}'\n");
    tcp_close(stream);
}

fn main() -> i64 {
    let listener = tcp_listen("127.0.0.1", 19000);
    let thread = thread_spawn(client_worker, raw task);
    let client = tcp_accept(listener);
    let msg = tcp_recv(client, 256);
    tcp_send(client, msg); // echo back
    thread_join(thread);
    tcp_close(client);
    tcp_listener_close(listener);
    return 0;
}
```

