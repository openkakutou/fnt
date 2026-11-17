---
status: todo
---
# Data model and `.fnt` v1 reading

## Description
Roadmap decision `035`: a pure-data model (font header: size, spacing, offset, type; character map: character → image region) and `Parse` for `.fnt` v1 (embedded PCX image + text character map). Reuse `sff`'s PCX decoding if it is exported, otherwise decide in an ADR.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO, validated against real lifebar fonts; ground layout rules in a source.

## Acceptance Criteria
- [ ] A real MUGEN v1 font parses into the model with correct glyph regions
- [ ] Malformed input returns a descriptive error, never a panic
- [ ] Fixed and variable width fonts both handled
