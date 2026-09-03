---
layout: default
title: Getting Started
nav_order: 2
description: "How to install, build, and run Gat programs."
permalink: /getting-started.html
---

# Getting Started with Gat

This guide walks you through setting up the Gat compiler, writing your first program, and understanding the core toolchain workflows.

---

## 1. Quick Installation

### Option A: Pre-built Release Binaries (Recommended)
Download the latest pre-compiled archive for your OS from [Releases](https://github.com/DanielcoderX/gat/releases/latest):

1. **Extract the archive**:
   - On Windows: Extract `gat-v0.1.0-windows-x64.zip` to a folder (e.g. `C:\tools\gat`).
   - On Linux: Extract `gat-v0.1.0-linux-x64.tar.gz` (`tar -xzf gat-v0.1.0-linux-x64.tar.gz -C ~/.local/`).
2. **Add `bin/` to your PATH**:
   - The archive contains `bin/gat` (the CLI driver) and `bin/gatc` (the core compiler).

### Option B: Clone and Verify from Source
Clone the repository:
```bash
git clone https://github.com/DanielcoderX/gat.git
cd gat
```

Run the automated test and bootstrap suite:
```powershell
# On Windows
.\test.ps1
```

---

## 2. Your First Program

Create a file named `hello.gat`:

```rust
// hello.gat
fn main() -> i64 {
    print("Hello from Gat!\n");
    return 0;
}
```

### Running Directly
You can run any `.gat` file directly with the CLI driver:
```powershell
gat run hello.gat
```
This compiles the code into a temporary native executable, runs it, forwards arguments and exit codes, and cleans up automatically.

### Compiling to Native Executable
To produce an optimized, standalone binary:
```powershell
gat build hello.gat -o hello.exe
```

Run the resulting binary:
```powershell
.\hello.exe
```

---

## 3. Cross-Compiling for Linux

Gat includes built-in dual backends. Without installing cross-compilers or GCC toolchains, you can emit raw Linux ELF64 binaries directly:

```powershell
gat build hello.gat -o hello_linux --target=linux
```

Transfer `hello_linux` to any x86-64 Linux server or WSL distribution:
```bash
chmod +x hello_linux
./hello_linux
```
Output:
```
Hello from Gat!
```

---

## 4. Managing Projects with `gat` CLI

Gat includes an integrated package and project manager:

### Create a New Project
```powershell
gat init my_app
cd my_app
```
This generates:
- `gat.mod`: Project manifest specifying name, version, and dependencies.
- `src/main.gat`: Default entrypoint.

### Build and Run Projects
```powershell
gat build
gat run
```

---

## 5. Next Steps

- Explore the [**Curated Example Gallery**](examples.html) to see language features in action.
- Read the [**Language Specification**](LANGUAGE_SPEC.html) for detailed syntax and type semantics.
- Discover available standard library modules in [**Standard Library**](STDLIB.html).
