---
status: todo
depends_on: [001]
---
# `.fnt` v2 reading through `sff`

## Description
Roadmap decision `035`: `.fnt` v2 is a text definition whose glyphs are sprites in an `.sff`. Parse it and resolve glyphs through the `sff` module, including Ikemen GO's extensions where they exist.

🔒 MUST: MUGEN 1.1 and Ikemen GO, real files.

## Acceptance Criteria
- [ ] A real v2 font resolves every glyph to its sprite
- [ ] A missing referenced `.sff` is a clear error
