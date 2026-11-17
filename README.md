# fnt

OpenKakutou `fnt` is a read/write Go library for MUGEN/Ikemen GO `.fnt` bitmap font files, v1 (embedded PCX image + character map) and v2 (text definition referencing an `.sff`), decided in roadmap decision `035`. Consumers: `game` (HUD and lifebar text), `lifebar-viewer-web`, `lifebar-editor`. It compiles to WASM and has no rendering dependency, like `sff` and `snd`.

Part of [OpenKakutou](https://github.com/openkakutou). Early stage: see `.vibe/backlog/` for the planned work.

## License

MIT, see [LICENSE](LICENSE).
