// Package fnt reads MUGEN / Ikemen GO bitmap font (.fnt) files as a pure
// data model, with no rendering dependency.
package fnt

// Point is an integer pair (x, y).
type Point struct {
	X, Y int
}

// RGB is one 8-bit palette entry.
type RGB struct {
	R, G, B byte
}

// Glyph locates one character inside the font strip: columns
// [X, X+Width) over the full image height.
type Glyph struct {
	X     int
	Width int
}

// Image is the font strip: 8-bit palette indices plus the 256-color palette.
// Ikemen GO renders palette index 0 as transparent; Palette holds the raw
// colors as stored in the file.
type Image struct {
	Width   int
	Height  int
	Pixels  []byte // row-major palette indices, len = Width*Height
	Palette [256]RGB
}

// Font is a parsed v1 .fnt file.
type Font struct {
	// Ver and Ver2 are the two header uint16 values at bytes 12-15
	// (0 and 1 in real v1 files, as seen in 1.0 files).
	Ver, Ver2 uint16
	Size      Point
	Spacing   Point
	Offset    Point
	// Colors is the [Def] Colors value clamped to [1, 255].
	Colors int
	// Type is the lowercased [Def] Type ("fixed", "variable", ...).
	Type   string
	Glyphs map[rune]Glyph
	Image  Image
}
