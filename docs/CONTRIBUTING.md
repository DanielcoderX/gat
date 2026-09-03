---
layout: default
title: Contributing
nav_order: 8
description: "Guidelines for contributing to the Gat programming language."
permalink: /CONTRIBUTING.html
---

# Contributing to Gat

We welcome contributions to the Gat compiler, standard library, documentation, and tooling!

---

## 1. Development Setup

Gat is self-hosting and requires no heavy build systems or external compilers:
- On Windows: Requires PowerShell.
- On Linux: Requires `bash` and a working 64-bit kernel (or WSL2).

### Clone the Repository
```bash
git clone https://github.com/DanielcoderX/gat.git
cd gat
```

---

## 2. Running Tests & Self-Hosting Verification

Before submitting changes, always run the full test and verification suite:

### Windows
```powershell
.\test.ps1
```
This runs:
1. Stage 2 compiler compilation (`bin/gatc.exe -> bin/gatc-stage2.exe`).
2. Stage 3 compiler compilation (`bin/gatc-stage2.exe -> bin/gatc-stage3.exe`).
3. Bitwise identity check between Stage 2 and Stage 3 (`fc /b`).
4. 23 feature and optimization test suites.
5. 7 negative compiler diagnostics tests.
6. Language Server (LSP) tests.
7. Linux direct syscall suite via WSL (if installed).

### Linux (WSL / Native)
```bash
# Run the Linux direct syscall suite
./bin/gatc examples/test_linux_suite.gat -o bin/test_linux_suite --target=linux
./bin/test_linux_suite
```

---

## 3. Coding Guidelines

1. **Self-Hosting Discipline**: When editing compiler source files (`src/*.gat`), ensure you only use language features currently supported by the baseline bootstrap compiler (`bin/gatc.exe`).
2. **Deterministic Output**: Compiler code generation must be deterministic; changes must not break the bitwise-identical stage comparison.
3. **Commit Convention**: We follow Conventional Commits (e.g. `feat: ...`, `fix: ...`, `docs: ...`, `perf: ...`, `test: ...`).
