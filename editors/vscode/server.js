#!/usr/bin/env node
// ============================================================================
// editors/vscode/server.js - Standalone Gat Language Server (LSP 3.17)
// ============================================================================

const fs = require('fs');
const path = require('path');
const { spawn } = require('child_process');

// In-memory document store: URI -> { text, version, lines }
const documents = new Map();

// Workspace root
let workspaceRoot = process.cwd();

// Find gat binary path
function findGatBinary() {
  const candidates = [
    path.join(workspaceRoot, 'bin', 'gat.exe'),
    path.join(workspaceRoot, 'bin', 'gatc.exe'),
    path.join(__dirname, '..', '..', 'bin', 'gat.exe'),
    path.join(__dirname, '..', '..', 'bin', 'gatc.exe'),
    'gat.exe',
    'gatc.exe'
  ];
  for (const c of candidates) {
    if (fs.existsSync(c)) {
      return path.resolve(c);
    }
  }
  return 'gat.exe';
}

// Convert URI to file path
function uriToPath(uri) {
  let decoded = decodeURIComponent(uri);
  if (decoded.startsWith('file:///')) {
    decoded = decoded.substring(8);
    if (process.platform === 'win32' && decoded.match(/^[a-zA-Z]:/)) {
      // e.g. C:/path...
    } else if (process.platform === 'win32' && decoded.startsWith('/')) {
      decoded = decoded.substring(1);
    }
  }
  return path.normalize(decoded);
}

// Convert file path to URI
function pathToUri(filePath) {
  let normalized = filePath.replace(/\\/g, '/');
  if (!normalized.startsWith('/')) {
    normalized = '/' + normalized;
  }
  return encodeURI('file://' + normalized);
}

// Send JSON-RPC response or notification
function send(msg) {
  const json = JSON.stringify(msg);
  const header = `Content-Length: ${Buffer.byteLength(json, 'utf8')}\r\n\r\n`;
  process.stdout.write(header + json);
}

function sendResponse(id, result) {
  send({ jsonrpc: '2.0', id, result });
}

function sendError(id, code, message) {
  send({ jsonrpc: '2.0', id, error: { code, message } });
}

function sendNotification(method, params) {
  send({ jsonrpc: '2.0', method, params });
}

// Run gat compiler diagnostics
function runDiagnostics(uri) {
  const filePath = uriToPath(uri);
  if (!fs.existsSync(filePath)) return;

  const gatBin = findGatBinary();
  const args = ['check', filePath];

  const proc = spawn(gatBin, args, { cwd: workspaceRoot, shell: true });
  let stdout = '';
  let stderr = '';

  proc.stdout.on('data', (d) => { stdout += d.toString(); });
  proc.stderr.on('data', (d) => { stderr += d.toString(); });

  proc.on('close', () => {
    const combined = stdout + '\n' + stderr;
    const diagnostics = [];

    // Parse [Parser Error] line L:C: message
    const parserRegex = /\[Parser Error\]\s+line\s+(\d+):(\d+):\s*(.*)/g;
    let match;
    while ((match = parserRegex.exec(combined)) !== null) {
      const line = Math.max(0, parseInt(match[1], 10) - 1);
      const col = Math.max(0, parseInt(match[2], 10) - 1);
      const msg = match[3];

      diagnostics.push({
        severity: 1, // Error
        range: {
          start: { line, character: col },
          end: { line, character: col + 5 }
        },
        message: msg,
        source: 'gat'
      });
    }

    // Parse [Type Error] message
    const typeRegex = /\[Type Error\]\s*(.*)/g;
    while ((match = typeRegex.exec(combined)) !== null) {
      const msg = match[1];
      // Type errors without direct line info attach to top of file or matching symbol
      diagnostics.push({
        severity: 1, // Error
        range: {
          start: { line: 0, character: 0 },
          end: { line: 0, character: 1 }
        },
        message: msg,
        source: 'gat'
      });
    }

    sendNotification('textDocument/publishDiagnostics', {
      uri,
      diagnostics
    });
  });
}

// Extract word under position
function getWordAt(text, line, character) {
  const lines = text.split(/\r?\n/);
  if (line >= lines.length) return '';
  const l = lines[line];
  let start = character;
  let end = character;
  const isWordChar = (c) => /[a-zA-Z0-9_]/.test(c);

  while (start > 0 && isWordChar(l[start - 1])) start--;
  while (end < l.length && isWordChar(l[end])) end++;
  return l.substring(start, end);
}

// Find declaration of symbol in file or imports
function findDefinition(uri, symbol) {
  if (!symbol) return null;
  const doc = documents.get(uri);
  const filesToSearch = [uri];

  // Also extract imported files
  if (doc) {
    const importRegex = /import\s+"([^"]+)";/g;
    let m;
    while ((m = importRegex.exec(doc.text)) !== null) {
      const impPath = path.resolve(workspaceRoot, m[1]);
      if (fs.existsSync(impPath)) {
        filesToSearch.push(pathToUri(impPath));
      }
    }
  }

  for (const fUri of filesToSearch) {
    let content = '';
    if (documents.has(fUri)) {
      content = documents.get(fUri).text;
    } else {
      const fPath = uriToPath(fUri);
      if (fs.existsSync(fPath)) {
        content = fs.readFileSync(fPath, 'utf8');
      }
    }

    const lines = content.split(/\r?\n/);
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Match fn declaration: fn <symbol>(
      const fnMatch = new RegExp(`\\bfn\\s+(${symbol})\\b`).exec(line);
      if (fnMatch) {
        return {
          uri: fUri,
          range: {
            start: { line: i, character: fnMatch.index + 3 },
            end: { line: i, character: fnMatch.index + 3 + symbol.length }
          }
        };
      }

      // Match struct/class/enum: struct/class/enum <symbol>
      const typeMatch = new RegExp(`\\b(struct|class|enum)\\s+(${symbol})\\b`).exec(line);
      if (typeMatch) {
        return {
          uri: fUri,
          range: {
            start: { line: i, character: typeMatch.index + typeMatch[1].length + 1 },
            end: { line: i, character: typeMatch.index + typeMatch[1].length + 1 + symbol.length }
          }
        };
      }

      // Match let variable: let <symbol>
      const letMatch = new RegExp(`\\blet\\s+(${symbol})\\b`).exec(line);
      if (letMatch) {
        return {
          uri: fUri,
          range: {
            start: { line: i, character: letMatch.index + 4 },
            end: { line: i, character: letMatch.index + 4 + symbol.length }
          }
        };
      }

      // Match struct/class field: <symbol>: <Type>
      const fieldMatch = new RegExp(`^\\s*(${symbol})\\s*:`).exec(line);
      if (fieldMatch) {
        return {
          uri: fUri,
          range: {
            start: { line: i, character: line.indexOf(symbol) },
            end: { line: i, character: line.indexOf(symbol) + symbol.length }
          }
        };
      }
    }
  }

  return null;
}

// Hover information provider
function getHover(uri, symbol) {
  if (!symbol) return null;
  const def = findDefinition(uri, symbol);
  if (!def) return null;

  let content = '';
  if (documents.has(def.uri)) {
    content = documents.get(def.uri).text;
  } else {
    content = fs.readFileSync(uriToPath(def.uri), 'utf8');
  }

  const lines = content.split(/\r?\n/);
  const defLine = lines[def.range.start.line] || '';

  return {
    contents: {
      kind: 'markdown',
      value: `\`\`\`gat\n${defLine.trim()}\n\`\`\``
    }
  };
}

// Document Symbols provider
function getDocumentSymbols(uri) {
  const doc = documents.get(uri);
  if (!doc) return [];

  const symbols = [];
  const lines = doc.text.split(/\r?\n/);

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];

    // Functions
    const fnMatch = /\bfn\s+([a-zA-Z_][a-zA-Z0-9_]*)/.exec(line);
    if (fnMatch) {
      symbols.push({
        name: fnMatch[1],
        kind: 12, // Function
        location: {
          uri,
          range: { start: { line: i, character: 0 }, end: { line: i, character: line.length } }
        }
      });
    }

    // Structs / Classes / Enums
    const typeMatch = /\b(struct|class|enum)\s+([a-zA-Z_][a-zA-Z0-9_]*)/.exec(line);
    if (typeMatch) {
      const kind = typeMatch[1] === 'struct' ? 23 : typeMatch[1] === 'class' ? 5 : 10;
      symbols.push({
        name: typeMatch[2],
        kind,
        location: {
          uri,
          range: { start: { line: i, character: 0 }, end: { line: i, character: line.length } }
        }
      });
    }
  }

  return symbols;
}

// Autocomplete provider
function getCompletions(uri, line, character) {
  const keywords = [
    'fn', 'let', 'struct', 'class', 'enum', 'match', 'deinit', 'return',
    'if', 'else', 'while', 'for', 'in', 'new', 'raw', 'weak', 'import',
    'true', 'false', 'nil', 'i8', 'i64', 'bool', 'string', 'void', 'array'
  ];

  const builtins = [
    'print', 'alloc_mem', 'free_mem', 'read_file', 'write_file', 'str_len',
    'str_eq', 'str_char', 'str_sub', 'str_concat', 'str_from_int', 'get_cmd_arg',
    'delete_file', 'sleep', 'get_temp_dir', 'exec_cmd', 'exit_process', 'get_pid',
    'thread_spawn', 'thread_join', 'mutex_new', 'mutex_lock', 'mutex_unlock',
    'option_some', 'option_none', 'option_unwrap', 'option_map',
    'result_ok', 'result_err', 'result_unwrap', 'result_map',
    'weak_from', 'weak_upgrade', 'weak_is_alive'
  ];

  const items = [];

  for (const kw of keywords) {
    items.push({ label: kw, kind: 14 }); // Keyword
  }
  for (const b of builtins) {
    items.push({ label: b, kind: 3 }); // Function
  }

  return items;
}

// Process incoming JSON-RPC buffer
let inputBuffer = Buffer.alloc(0);

process.stdin.on('data', (chunk) => {
  inputBuffer = Buffer.concat([inputBuffer, chunk]);

  while (true) {
    const headerEnd = inputBuffer.indexOf('\r\n\r\n');
    if (headerEnd === -1) break;

    const header = inputBuffer.slice(0, headerEnd).toString('utf8');
    const match = /Content-Length:\s*(\d+)/i.exec(header);
    if (!match) {
      inputBuffer = inputBuffer.slice(headerEnd + 4);
      continue;
    }

    const contentLength = parseInt(match[1], 10);
    const totalLength = headerEnd + 4 + contentLength;
    if (inputBuffer.length < totalLength) break;

    const body = inputBuffer.slice(headerEnd + 4, totalLength).toString('utf8');
    inputBuffer = inputBuffer.slice(totalLength);

    try {
      const msg = JSON.parse(body);
      handleMessage(msg);
    } catch (e) {
      console.error('Failed to parse JSON-RPC message:', e);
    }
  }
});

function handleMessage(msg) {
  const { id, method, params } = msg;

  if (method === 'initialize') {
    if (params.rootUri) {
      workspaceRoot = uriToPath(params.rootUri);
    } else if (params.rootPath) {
      workspaceRoot = params.rootPath;
    }

    sendResponse(id, {
      capabilities: {
        textDocumentSync: 1, // Full sync
        definitionProvider: true,
        hoverProvider: true,
        documentSymbolProvider: true,
        completionProvider: {
          triggerCharacters: ['.', ':']
        }
      },
      serverInfo: {
        name: 'gat-lsp',
        version: '0.3.0'
      }
    });
    return;
  }

  if (method === 'initialized') {
    return;
  }

  if (method === 'textDocument/didOpen') {
    const { uri, text, version } = params.textDocument;
    documents.set(uri, { text, version });
    runDiagnostics(uri);
    return;
  }

  if (method === 'textDocument/didChange') {
    const { uri, version } = params.textDocument;
    const text = params.contentChanges[0].text;
    documents.set(uri, { text, version });
    runDiagnostics(uri);
    return;
  }

  if (method === 'textDocument/didSave') {
    const { uri } = params.textDocument;
    runDiagnostics(uri);
    return;
  }

  if (method === 'textDocument/definition') {
    const { textDocument, position } = params;
    const doc = documents.get(textDocument.uri);
    if (!doc) {
      sendResponse(id, null);
      return;
    }
    const word = getWordAt(doc.text, position.line, position.character);
    const def = findDefinition(textDocument.uri, word);
    sendResponse(id, def);
    return;
  }

  if (method === 'textDocument/hover') {
    const { textDocument, position } = params;
    const doc = documents.get(textDocument.uri);
    if (!doc) {
      sendResponse(id, null);
      return;
    }
    const word = getWordAt(doc.text, position.line, position.character);
    const hover = getHover(textDocument.uri, word);
    sendResponse(id, hover);
    return;
  }

  if (method === 'textDocument/documentSymbol') {
    const { textDocument } = params;
    const symbols = getDocumentSymbols(textDocument.uri);
    sendResponse(id, symbols);
    return;
  }

  if (method === 'textDocument/completion') {
    const { textDocument, position } = params;
    const completions = getCompletions(textDocument.uri, position.line, position.character);
    sendResponse(id, completions);
    return;
  }

  if (method === 'shutdown') {
    sendResponse(id, null);
    return;
  }

  if (method === 'exit') {
    process.exit(0);
    return;
  }

  if (id !== undefined) {
    sendResponse(id, null);
  }
}
