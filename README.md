# The `gat` Programming Language

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/DanielcoderX/gat?color=blue)](https://github.com/DanielcoderX/gat/releases/latest)
[![Documentation](https://img.shields.io/badge/Docs-Online-blueviolet)](https://danielcoderx.github.io/gat/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20x64%20%7C%20Linux%20x64-lightgrey.svg)](#dual-backend-architecture)
[![Self-Hosting: 100%](https://img.shields.io/badge/Self--Hosting-100%25%20Bitwise%20Verified-brightgreen.svg)](#self-hosting-verification)
[![Tooling: LSP 3.17](https://img.shields.io/badge/Tooling-LSP%20%2B%20VS%20Code-orange.svg)](editors/vscode/)

`gat` is a compiled, low-level systems programming language featuring automatic reference counting (ARC), deterministic destructors, and direct machine code emission for Windows PE32+ and Linux ELF64. The compiler is **100% self-hosted** with bitwise-reproducible multi-stage bootstrap.

📖 **Read the official documentation**: [danielcoderx.github.io/gat](https://danielcoderx.github.io/gat/)

---

## What Makes `gat` Different?

| Feature | Why It Matters |
|---|---|
| **Zero-Dependency Direct Syscalls** | On Linux, `gat` emits direct x86-64 `syscall` instructions (`sys_read`, `sys_write`, `sys_mmap`, `sys_clone`, `sys_nanosleep`). **Zero libc, musl, or dynamic linker dependencies.** On Windows, it links directly to `kernel32.dll` with no C runtime (`MSVCRT`) requirement. |
| **Dual Memory Model** | Choose between stack-allocated value types (`struct`, zero heap/refcount overhead) and heap-allocated reference types (`class`, managed via deterministic non-atomic ARC). |
| **Deterministic RAII Destructors** | `deinit` blocks execute the exact instant an object's reference count hits zero. No stop-the-world garbage collection pauses. |
| **Cycle Breaking via `weak T`** | Native non-owning `weak T` references prevent reference cycles from leaking memory. |
| **100% Bitwise Self-Hosting** | `src/compiler.gat` compiles itself across stages (`stage2 == stage3`) with **100% exact bitwise identity** on both Windows and Linux. |
| **Built-in Toolchain & LSP** | Includes a full CLI driver (`gat run`, `gat build`, `gat check`), a package manager (`gat.mod` / `gat.lock`), and a complete Language Server Protocol (LSP 3.17) implementation for VS Code. |

---

## Try `gat` in 60 Seconds

### 1. Download Pre-Built Binaries
Download the latest v0.1.0 release for your platform from [GitHub Releases](https://github.com/DanielcoderX/gat/releases/latest):
- **Windows x86-64**: `gat-v0.1.0-windows-x64.zip` (extract and add `bin/` to PATH)
- **Linux x86-64**: `gat-v0.1.0-linux-x64.tar.gz` (`tar -xzf gat-v0.1.0-linux-x64.tar.gz`)

### 2. Run Hello World
```powershell
# Windows
.\bin\gat.exe run examples\showcase\01_hello_world.gat

# Linux
./bin/gat run examples/showcase/01_hello_world.gat
```

### 3. Build a Standalone Native Executable
```powershell
# Windows PE32+ (Standalone .exe)
.\bin\gat.exe build examples\showcase\01_hello_world.gat -o hello.exe
.\hello.exe

# Linux ELF64 (Raw Syscall Binary - Cross-compile or Native)
.\bin\gat.exe build examples\showcase\01_hello_world.gat -o hello_linux --target=linux
```

---

## Curated Example Gallery (`examples/showcase/`)

Explore our beginner-friendly tutorial gallery in [`examples/showcase/`](examples/showcase/):

| # | Example | Concept Demonstrated |
|---|---|---|
| 01 | [`01_hello_world.gat`](examples/showcase/01_hello_world.gat) | Minimal entrypoint, `fn main() -> i64`, intrinsic `print` |
| 02 | [`02_fizzbuzz.gat`](examples/showcase/02_fizzbuzz.gat) | Range loops (`for i in 1..21`), conditionals, string interpolation |
| 03 | [`03_struct_vs_class.gat`](examples/showcase/03_struct_vs_class.gat) | **Value vs Reference**: Stack `struct` vs Heap ARC `class` |
| 04 | [`04_arc_deinit.gat`](examples/showcase/04_arc_deinit.gat) | Deterministic ARC lifecycle, RAII destructors (`deinit`) |
| 05 | [`05_weak_references.gat`](examples/showcase/05_weak_references.gat) | Non-owning `weak T` references, cycle breaking, `weak_upgrade` |
| 06 | [`06_enum_match.gat`](examples/showcase/06_enum_match.gat) | Strongly typed `enum`, pattern matching with `match` |
| 07 | [`07_modules.gat`](examples/showcase/07_modules.gat) | Namespaced modules (`import ... as alias;`), modular architecture |
| 08 | [`08_word_count_cli.gat`](examples/showcase/08_word_count_cli.gat) | Command line arguments (`get_cmd_arg`), file I/O, text parsing |
| 09 | [`09_cross_platform.gat`](examples/showcase/09_cross_platform.gat) | Cross-platform APIs, Windows PE and Linux ELF64 parity |

> *Note: Every example above is verified against the compiler on both Windows and Linux.*

---

## Dual Backend Architecture

`gat` emits raw machine code directly without calling an external assembler or linker:

- **Windows x86-64 (PE32+)**:
  - Emits PE headers, `.text`, `.rdata`, `.data`, and `.pdata` sections.
  - Generates Import Address Table (IAT) binding only to `KERNEL32.dll`.
  - Zero dependencies on `msvcrt.dll` or the Visual C++ runtime.

- **Linux x86-64 (ELF64)**:
  - Emits static ELF64 executable (`ET_EXEC`).
  - Runtime operations map directly to Linux kernel syscalls via `syscall` instruction:
    - Heap: `sys_mmap` (9) and `sys_munmap` (11)
    - File & Console I/O: `sys_read` (0), `sys_write` (1), `sys_open` (2), `sys_close` (3), `sys_stat` (4)
    - Concurrency: `sys_clone` (56) with native atomic CAS mutexes
    - Lifecycle: `sys_exit_group` (231), `sys_getpid` (39), `sys_nanosleep` (35)
  - Zero shared library dependencies (`ldd` reports "not a dynamic executable").

---

## Self-Hosting Verification

The compiler (`src/compiler.gat`) is written entirely in `gat` and compiles itself deterministically:

```powershell
# 1. Compile compiler with current binary -> stage2
.\bin\gatc.exe src\compiler.gat -o bin\gatc-stage2.exe

# 2. Compile compiler with stage2 binary -> stage3
.\bin\gatc-stage2.exe src\compiler.gat -o bin\gatc-stage3.exe

# 3. Verify exact 100% bitwise identity
fc.exe /b bin\gatc-stage2.exe bin\gatc-stage3.exe
# Result: "FC: no differences encountered"
```

The same bootstrap verification runs natively on Linux:
```bash
./bin/gatc src/compiler.gat -o gatc-gen2 --target=linux
./gatc-gen2 src/compiler.gat -o gatc-gen3 --target=linux
cmp gatc-gen2 gatc-gen3
# Result: 0 differences (100% exact match)
```

---

## Package Manager (`gat.mod`)

Gat projects declare dependencies in `gat.mod`:

```
module my_app

require github.com/user/gat-json v1.0.0
require ./local_packages/math_lib v0.1.0
```

- Initialize a project: `gat init my_app`
- Fetch & cache dependencies: `gat fetch` (generates `gat.lock` and populates `.gat/deps/`)
- See [docs/MODULES.md](docs/MODULES.md) for details.

---

## Editor Support (VS Code & LSP)

An official Visual Studio Code extension and standalone Language Server Protocol (LSP 3.17) server are included in [`editors/vscode/`](editors/vscode/):
- **Features**: Real-time syntax highlighting, compiler error diagnostics, go-to-definition, hover documentation, document symbols outline, and autocomplete.
- **Install**: Link `editors/vscode/` into `%USERPROFILE%\.vscode\extensions\gat-language`.
- See [editors/vscode/README.md](editors/vscode/README.md) for details.

---

## Contributing

Contributions to the compiler, standard library, documentation, and tooling are welcome!

1. **Test Suite**: Always verify changes by running the test suite:
   ```powershell
   powershell -ExecutionPolicy Bypass -File .\test.ps1
   ```
   Ensure all 23 language test suites, diagnostic negative tests, LSP verification, and multi-stage self-hosting bitwise identity tests pass.
2. **Commit Convention**: We follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat: ...`, `fix: ...`, `docs: ...`, `test: ...`).
3. **Architecture Reference**: See [`docs/CONTRIBUTING.md`](docs/CONTRIBUTING.md) and [`docs/LANGUAGE_SPEC.md`](docs/LANGUAGE_SPEC.md).

---

## Documentation Links

- [Official Documentation Site](https://danielcoderx.github.io/gat/)
- [Language Specification](docs/LANGUAGE_SPEC.md)
- [Standard Library Reference](docs/STDLIB.md)
- [Modules & Package Manager](docs/MODULES.md)
- [Dual Backend Architecture](docs/dual_backend.md)
- [Contributing Guide](docs/CONTRIBUTING.md)

---

## License

This project is licensed under the [MIT License](LICENSE).
