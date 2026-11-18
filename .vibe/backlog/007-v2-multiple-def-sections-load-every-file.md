---
status: todo
---
# v2: load the file of every `[Def]` section, as Ikemen GO does

## Description
Ikemen GO's `loadFntV2` (`src/font.go`) applies each `[Def]` section in turn through `loadDefInfo`. For every section with a non-empty `File`, it loads that file right away (`LoadFntSff` for a bitmap font, `LoadFntTtf` for truetype), using the `Type` and `BankType` in force at that point. A font whose `[Def]` sections name several sprite files therefore gets the glyphs of all of them.

`fnt` v0.2.0 (backlog 002) narrows this: `ParseV2`/`LoadV2` read a single sff, the one named by the last section with a `File`, with the final `BankType`, and drop the glyphs of the earlier files. The doc comment of `ParseV2` in `fontv2.go` records this narrowing.

Reproduce Ikemen GO's behaviour exactly:
- one sprite load per section that has a `File`, in file order, with that section's `Type` and `BankType` at that moment;
- glyphs merged across files the way Ikemen GO writes them (`f.images[group][rune]`), so a later file overrides an earlier glyph if Ikemen GO does — check this in the source, and keep first-wins inside one sff (`src/image.go` `loadSff`, already done in 002);
- palettes as Ikemen GO collects them across files, if they are per load;
- a truetype section among bitmap ones, handled as Ikemen GO does.

`ParseV2` takes a single `sffData io.ReaderAt` today, so it cannot be given several files: its signature has to change, for example to take a function that returns the bytes of a named file. `LoadV2` resolves each `File` beside the `.def` with the existing rules (case-insensitive, `\`, no escape, no symlinks). Bump the minor version, since the public API changes.

🔒 MUST: Ikemen GO behaviour, cited from its source (`src/font.go`, `src/image.go`), verified on a real or derived fixture.

## Acceptance Criteria
- [ ] A `.def` with two `[Def]` sections naming two sff files gets the glyphs of both, with any overlap resolved as in Ikemen GO (cited)
- [ ] Each load uses the `BankType` and `Type` of its own section at that point, as in Ikemen GO
- [ ] The `ParseV2` doc comment no longer describes a narrowing; single-file fonts and their goldens are unchanged
- [ ] A missing file in any section is a clear error naming it

## Notes
Raised after 002 (2026-11-18); the Product Owner requires Ikemen GO's behaviour.
