---
status: todo
depends_on: [001, 002]
---
# WASM entrypoint with a release smoke gate

## Description
Roadmap decision `035`: `cmd/wasm` exposing parse + glyph pixel resolution to JavaScript (contract style of `sff`), with `cmd/wasm/smoke.mjs` run by the release workflow.

## Acceptance Criteria
- [ ] `lifebar-viewer-web` can render a font's text through the module
- [ ] The release workflow fails if the smoke test fails
