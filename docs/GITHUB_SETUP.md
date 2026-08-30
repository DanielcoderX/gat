# GitHub Repository Setup Guide for `gat`

This document provides recommended settings, metadata, and procedures for publishing and maintaining the `gat` repository on GitHub.

---

## 1. Repository Metadata

- **Repository Name**: `gat`
- **One-Line Description**:
  > A self-hosting, low-level systems programming language with ARC memory management and native Windows x86-64 PE code emission.
- **Website / Link**: `https://github.com/DanielcoderX/gat` (or project site)
- **Topics / Tags**:
  - `programming-language`
  - `compiler`
  - `self-hosting`
  - `systems-programming`
  - `windows`
  - `x86-64`
  - `arc`
  - `pe-format`
  - `lsp`

---

## 2. Git & File Attributes

The repository uses `.gitattributes` to ensure proper handling of line endings and pre-built binaries:
- `*.exe binary` prevents Git from corrupting Windows seed binaries (`bin/gat.exe`, `bin/gatc.exe`).
- `*.gat text eol=lf` enforces LF line endings across source code.
- `*.ps1 text eol=crlf` maintains Windows PowerShell script conventions.

---

## 3. Continuous Integration (CI)

A GitHub Actions workflow is provided in `.github/workflows/test.yml`:
- Runs on `windows-latest`.
- Automatically executes the full 3-stage bitwise self-compilation verification and language feature test suite via `powershell -ExecutionPolicy Bypass -File .\test.ps1`.
- Verifies pull requests and commits to `master`/`main`.

---

## 4. Branch Protection & Workflow

For a single-developer repository:
- Keep `master` (or `main`) as the default branch.
- Automated CI will run on every push and PR.
- When expanding to outside contributors, enable branch protection requiring status checks to pass before merging.

---

## 5. Binary Distribution & Release Strategy

- **Seed Binaries**: The repo includes `bin/gatc.exe` (seed compiler) and `bin/gat.exe` (CLI driver) so new users can clone and build immediately without requiring any pre-existing compiler.
- **Releases**: For tagged releases (e.g. `v0.3.0`), attach standalone zip archives containing the built binaries (`gat.exe`, `gatc.exe`) and standard library (`std/`) to GitHub Releases.
