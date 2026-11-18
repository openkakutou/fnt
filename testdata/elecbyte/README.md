# Elecbyte MUGEN fonts (test fixtures)

Real `.fnt` v1 fonts used by this repo's fixture-driven tests. They are copied unmodified, not part of the library.

## Source

Downloaded from [`ikemen-engine/Ikemen_GO-Elecbyte-Screenpack`](https://github.com/ikemen-engine/Ikemen_GO-Elecbyte-Screenpack), commit `91e9ff0cd21def4cf1fe2780e7709e2243b82bc9`, folder `font/`:

`f-4x6.fnt`, `f-6x8f.fnt`, `f-6x9.fnt`, `f-6x9f.fnt`, `jg.fnt`, `name1.fnt`, `num1.fnt`

These are the MUGEN font files Elecbyte ships with MUGEN, bundled by Ikemen GO.

## Licence

These files are **not** under this repository's MIT licence. They keep their own licence, quoted from the source repository's `LICENCE.txt`:

> Electbyte's Mugen font files are licensed under a Creative Commons 3.0 Non-commercial License, with optional attribution.
> http://creativecommons.org/licenses/by-nc/3.0/

They are © Elecbyte, redistributed here for non-commercial testing only.

## Golden files

`<name>.golden.json` next to each font records what `Parse` reads from it. The golden test compares against it. After a deliberate parser change, regenerate with `go test -run TestParse_ElecbyteFonts -update` and review the diff.
