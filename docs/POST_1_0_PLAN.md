# Post-1.0 Roadmap Implementation Plan

## Tracks
1. **Stdlib Crypto & Hashing** (`std/crypto.gat`): Base64 encode/decode, SHA-256, HMAC-SHA256.
2. **Stdlib Async & Event-Loop Polling** (`std/event.gat`): Socket polling abstractions (`select` / `poll`), event loop, timer callbacks.
3. **Optimizer Inlining & Dead Branch Pruning** (`src/opt.gat`): Constant conditional branch elimination, small leaf function inlining.
4. **Tooling: Linter & Static Analyzer** (`src/lint.gat`, `cli/gat.gat`): `gat lint <file>` static analysis tool detecting unused bindings, unreachable code, unhandled Result/Option.

## Execution Order
- [ ] 1. Crypto (`std/crypto.gat` + `examples/test_crypto.gat`)
- [ ] 2. Event Polling (`std/event.gat` + `examples/test_event.gat`)
- [ ] 3. Optimizer passes (`src/opt.gat` + `examples/test_opt_inline.gat`)
- [ ] 4. Linter CLI (`src/lint.gat` + `gat lint`)
- [ ] 5. Full regression suite & bootstrap
