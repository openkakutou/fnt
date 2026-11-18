# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-11-18

### Added

- Read MUGEN 1.1 / Ikemen GO version 2 fonts: the font definition file and the sprite file holding its characters, with every character resolved to its sprite (width and drawing offset) per bank, following Ikemen GO's own rules. TrueType font definitions are read without glyphs, and a missing or unreadable sprite file is reported with a clear error naming it.

## [0.1.0] - 2026-11-18

### Added

- Read MUGEN/Ikemen GO version 1 bitmap fonts: the font settings (size, spacing, offset, colors, fixed or variable type), the image area of every character, and the decoded font image with its palette, following Ikemen GO's own rules. A damaged font file is reported with a clear error instead of crashing.

[Unreleased]: https://github.com/openkakutou/fnt/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/openkakutou/fnt/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/openkakutou/fnt/releases/tag/v0.1.0
