---
status: todo
depends_on: [001, 002]
---
# Write `.fnt` v1 and v2

## Description
Roadmap decision `035`: serialize the model back, byte-exact for unmodified files, for `lifebar-editor`.

## Acceptance Criteria
- [ ] Round trip of real v1 and v2 fonts is byte-identical
- [ ] An edited glyph map re-parses to the edited model
