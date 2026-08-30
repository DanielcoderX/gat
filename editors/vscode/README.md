# Gat Language Extension for VS Code

Official Visual Studio Code extension providing syntax highlighting, Language Server Protocol (LSP) diagnostics, go-to-definition, hover information, and snippets for the **Gat** programming language.

## Features

- **Syntax Highlighting**: Comprehensive TextMate grammar covering the full Gat language specification:
  - Keywords, storage types (`fn`, `struct`, `class`, `enum`, `let`, `deinit`), and modifiers (`raw`, `weak`, `new`).
  - Primitive types (`i8`, `i64`, `bool`, `string`, `void`, `array`) and custom types.
  - String interpolation expressions (`{...}`).
  - Built-in runtime functions and standard library combinators.
- **Diagnostics**: Real-time error squiggles powered directly by `gat check` in the self-hosting Gat compiler.
- **Go to Definition**: Jump directly to declarations of local variables, function definitions, structs, classes, enums, and fields (including imported modules).
- **Hover**: Inspect signatures and declarations.
- **Document Symbols / Outline**: File navigation outline for functions, structs, classes, and enums.
- **Snippets**: Quick expansion for `fn`, `class`, `struct`, `enum`, `main`, and `match`.

## Installation

### Method 1: Local Link
Copy or symlink `editors/vscode` into your VS Code extensions folder:
- **Windows**: `%USERPROFILE%\.vscode\extensions\gat-language`
- **macOS / Linux**: `~/.vscode/extensions/gat-language`

### Method 2: Standalone LSP Server
The LSP server (`editors/vscode/server.js`) can be run with any LSP-compatible editor (Neovim, Helix, Sublime Text, Emacs):
```bash
node editors/vscode/server.js
```
