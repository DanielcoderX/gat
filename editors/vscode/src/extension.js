// ============================================================================
// editors/vscode/src/extension.js - VS Code Extension Client for Gat LSP
// ============================================================================

const path = require('path');
const vscode = require('vscode');
const { LanguageClient, TransportKind } = require('vscode-languageclient/node');

let client;

function activate(context) {
  const serverModule = context.asAbsolutePath(path.join('server.js'));

  const serverOptions = {
    run: { module: serverModule, transport: TransportKind.stdio },
    debug: { module: serverModule, transport: TransportKind.stdio }
  };

  const clientOptions = {
    documentSelector: [{ scheme: 'file', language: 'gat' }],
    synchronize: {
      fileEvents: vscode.workspace.createFileSystemWatcher('**/*.gat')
    }
  };

  client = new LanguageClient(
    'gatLanguageServer',
    'Gat Language Server',
    serverOptions,
    clientOptions
  );

  client.start();
}

function deactivate() {
  if (!client) {
    return undefined;
  }
  return client.stop();
}

module.exports = {
  activate,
  deactivate
};
