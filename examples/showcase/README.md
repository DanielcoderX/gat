# Gat Curated Example Gallery

Welcome to the **Gat** example gallery! This directory contains curated, beginner-friendly example programs designed to teach the language step-by-step.

> **Note**: This gallery is distinct from the internal test suite files in `examples/test_*.gat`. Every example here is standalone, heavily commented, and verified against the compiler.

---

## Gallery Index

| # | Example | Core Concepts |
|---|---|---|
| 01 | [`01_hello_world.gat`](01_hello_world.gat) | Program entrypoint, `fn main() -> i64`, basic `print` output |
| 02 | [`02_fizzbuzz.gat`](02_fizzbuzz.gat) | Range loops (`for i in 1..21`), conditionals, string interpolation |
| 03 | [`03_struct_vs_class.gat`](03_struct_vs_class.gat) | **Dual memory model**: Stack value `struct` vs Heap ARC `class` |
| 04 | [`04_arc_deinit.gat`](04_arc_deinit.gat) | Deterministic ARC lifecycle, RAII destructors (`deinit`) |
| 05 | [`05_weak_references.gat`](05_weak_references.gat) | Non-owning `weak T` references, cycle breaking, `weak_upgrade` |
| 06 | [`06_enum_match.gat`](06_enum_match.gat) | Strongly typed `enum`, pattern matching with `match` |
| 07 | [`07_modules.gat`](07_modules.gat) | Namespaced modules (`import ... as alias;`), modular code organization |
| 08 | [`08_word_count_cli.gat`](08_word_count_cli.gat) | Command line arguments (`get_cmd_arg`), file I/O, text traversal |
| 09 | [`09_cross_platform.gat`](09_cross_platform.gat) | Cross-platform APIs, Windows PE32+ and Linux ELF64 target support |

---

## How to Run the Examples

### On Windows
```powershell
# Run directly with the CLI:
.\bin\gat.exe run examples\showcase\01_hello_world.gat

# Or compile to a standalone native executable:
.\bin\gat.exe build examples\showcase\03_struct_vs_class.gat -o demo.exe
.\demo.exe
```

### On Linux (ELF64 Direct Syscalls)
```bash
# Cross-compile for Linux from Windows or compile natively on Linux:
gat build examples/showcase/01_hello_world.gat -o demo --target=linux
./demo
```
