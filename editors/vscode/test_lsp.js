// ============================================================================
// editors/vscode/test_lsp.js - Automated LSP Server Verification Suite
// ============================================================================

const path = require('path');
const fs = require('fs');
const { spawn } = require('child_process');

console.log('=== Running Gat LSP Server Automated Tests ===');

const serverPath = path.resolve(__dirname, 'server.js');
const server = spawn('node', [serverPath], { cwd: path.resolve(__dirname, '..', '..') });

let buffer = Buffer.alloc(0);
let msgId = 1;
const pendingRequests = new Map();

function send(msg) {
  const json = JSON.stringify(msg);
  const header = `Content-Length: ${Buffer.byteLength(json, 'utf8')}\r\n\r\n`;
  server.stdin.write(header + json);
}

function request(method, params) {
  return new Promise((resolve) => {
    const id = msgId++;
    pendingRequests.set(id, resolve);
    send({ jsonrpc: '2.0', id, method, params });
  });
}

function notify(method, params) {
  send({ jsonrpc: '2.0', method, params });
}

server.stdout.on('data', (chunk) => {
  buffer = Buffer.concat([buffer, chunk]);

  while (true) {
    const headerEnd = buffer.indexOf('\r\n\r\n');
    if (headerEnd === -1) break;

    const header = buffer.slice(0, headerEnd).toString('utf8');
    const match = /Content-Length:\s*(\d+)/i.exec(header);
    if (!match) {
      buffer = buffer.slice(headerEnd + 4);
      continue;
    }

    const contentLength = parseInt(match[1], 10);
    const totalLength = headerEnd + 4 + contentLength;
    if (buffer.length < totalLength) break;

    const body = buffer.slice(headerEnd + 4, totalLength).toString('utf8');
    buffer = buffer.slice(totalLength);

    try {
      const msg = JSON.parse(body);
      if (msg.id && pendingRequests.has(msg.id)) {
        const resolve = pendingRequests.get(msg.id);
        pendingRequests.delete(msg.id);
        resolve(msg.result);
      }
    } catch (e) {
      console.error(e);
    }
  }
});

async function runTests() {
  // 1. Initialize
  const rootDir = path.resolve(__dirname, '..', '..');
  const initRes = await request('initialize', {
    rootUri: 'file:///' + rootDir.replace(/\\/g, '/'),
    capabilities: {}
  });

  if (!initRes.capabilities.definitionProvider || !initRes.capabilities.hoverProvider) {
    console.error('[FAIL] Missing server capabilities:', initRes);
    process.exit(1);
  }
  console.log('  [PASS] 1. LSP Initialize handshake');

  // 2. Open document & check definitions
  const testFile = path.join(rootDir, 'examples', 'test_first_class_fn.gat');
  const fileUri = 'file:///' + testFile.replace(/\\/g, '/');
  const content = fs.readFileSync(testFile, 'utf8');

  notify('textDocument/didOpen', {
    textDocument: { uri: fileUri, languageId: 'gat', version: 1, text: content }
  });

  // Test definition of "add"
  const lines = content.split(/\r?\n/);
  let addLine = 0;
  let addChar = 3;
  for (let i = 0; i < lines.length; i++) {
    if (lines[i].includes('fn add(')) {
      addLine = i;
      addChar = lines[i].indexOf('add');
      break;
    }
  }

  const defRes = await request('textDocument/definition', {
    textDocument: { uri: fileUri },
    position: { line: addLine, character: addChar + 1 }
  });

  if (!defRes || !defRes.range) {
    console.error('[FAIL] Go-to-definition failed for add:', defRes);
    process.exit(1);
  }
  console.log('  [PASS] 2. Go-to-definition for top-level function');

  // 3. Hover
  const hoverRes = await request('textDocument/hover', {
    textDocument: { uri: fileUri },
    position: { line: addLine, character: addChar + 1 }
  });

  if (!hoverRes || !hoverRes.contents || !hoverRes.contents.value.includes('fn add')) {
    console.error('[FAIL] Hover failed for add:', hoverRes);
    process.exit(1);
  }
  console.log('  [PASS] 3. Hover signature display');

  // 4. Document Symbols
  const symRes = await request('textDocument/documentSymbol', {
    textDocument: { uri: fileUri }
  });

  if (!Array.isArray(symRes) || symRes.length === 0) {
    console.error('[FAIL] Document symbols empty:', symRes);
    process.exit(1);
  }
  console.log(`  [PASS] 4. Document symbols outline (${symRes.length} symbols extracted)`);

  // 5. Completions
  const compRes = await request('textDocument/completion', {
    textDocument: { uri: fileUri },
    position: { line: 0, character: 0 }
  });

  if (!Array.isArray(compRes) || compRes.length < 20) {
    console.error('[FAIL] Completion items insufficient:', compRes);
    process.exit(1);
  }
  console.log(`  [PASS] 5. Autocomplete suggestions (${compRes.length} items returned)`);

  console.log('\n[PASS] All LSP Server tests completed successfully!');
  server.kill();
  process.exit(0);
}

runTests();
