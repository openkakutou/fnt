package fnt

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openkakutou/sff"
)

// testPalette returns a palette where entry i is (i, i+1, i+2) mod 256.
func testPalette() [256]RGB {
	var p [256]RGB
	for i := range p {
		p[i] = RGB{byte(i), byte(i + 1), byte(i + 2)}
	}
	return p
}

// buildPCXBlock returns a PCX block: header + RLE pixels + 768-byte palette.
func buildPCXBlock(t *testing.T, w, h int, pixels []byte, pal [256]RGB) []byte {
	t.Helper()
	pcx, err := sff.EncodePCX(&sff.PCXImage{Width: w, Height: h, BytesPerPixel: 1, Pixels: pixels})
	if err != nil {
		t.Fatalf("EncodePCX: %v", err)
	}
	for _, c := range pal {
		pcx = append(pcx, c.R, c.G, c.B)
	}
	return pcx
}

// buildFnt assembles a v1 .fnt: header, PCX block, text block.
func buildFnt(pcx []byte, text string) []byte {
	out := make([]byte, 32)
	copy(out, "ElecbyteFnt\x00")
	binary.LittleEndian.PutUint16(out[14:], 1) // ver2 = 1 (bytes 12..15 = 00 00 01 00)
	binary.LittleEndian.PutUint32(out[16:], 32)
	binary.LittleEndian.PutUint32(out[20:], uint32(len(pcx)))
	binary.LittleEndian.PutUint32(out[24:], uint32(32+len(pcx)))
	binary.LittleEndian.PutUint32(out[28:], uint32(len(text)))
	out = append(out, pcx...)
	return append(out, text...)
}

// stripPixels builds a w x h strip whose pixel (x, y) = (x + 10*y) mod 251.
func stripPixels(w, h int) []byte {
	px := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px[y*w+x] = byte((x + 10*y) % 251)
		}
	}
	return px
}

func parseFntText(t *testing.T, w, h int, text string) *Font {
	t.Helper()
	data := buildFnt(buildPCXBlock(t, w, h, stripPixels(w, h), testPalette()), text)
	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f
}

func TestParse_ReadsVariableFont_WithExplicitXAndWidth(t *testing.T) {
	text := "[Def]\nSize = 3,6\nSpacing = 1,2\nOffset = -1,4\nColors = 16\nType = Variable\n\n" +
		"[Map]\nA 0 3\nB 3 5\nC 8 2\n"
	f := parseFntText(t, 12, 6, text)

	if f.Size != (Point{3, 6}) || f.Spacing != (Point{1, 2}) || f.Offset != (Point{-1, 4}) {
		t.Errorf("def points: size=%v spacing=%v offset=%v", f.Size, f.Spacing, f.Offset)
	}
	if f.Colors != 16 || f.Type != "variable" {
		t.Errorf("colors=%d type=%q", f.Colors, f.Type)
	}
	if f.Ver != 0 || f.Ver2 != 1 {
		t.Errorf("ver=%d ver2=%d", f.Ver, f.Ver2)
	}
	want := map[rune]Glyph{'A': {0, 3}, 'B': {3, 5}, 'C': {8, 2}}
	if len(f.Glyphs) != 3 {
		t.Fatalf("glyph count = %d, want 3: %v", len(f.Glyphs), f.Glyphs)
	}
	for r, g := range want {
		if f.Glyphs[r] != g {
			t.Errorf("glyph %q = %v, want %v", r, f.Glyphs[r], g)
		}
	}
}

func TestParse_ReadsImagePixelsAndPalette(t *testing.T) {
	f := parseFntText(t, 5, 3, "[Def]\nSize=5,3\n[Map]\nA\n")
	if f.Image.Width != 5 || f.Image.Height != 3 {
		t.Fatalf("image size = %dx%d", f.Image.Width, f.Image.Height)
	}
	// pixel (4,2) = 4+20 = 24
	if got := f.Image.Pixels[2*5+4]; got != 24 {
		t.Errorf("pixel(4,2) = %d, want 24", got)
	}
	if len(f.Image.Pixels) != 15 {
		t.Errorf("pixels len = %d, want 15", len(f.Image.Pixels))
	}
	if f.Image.Palette[0] != (RGB{0, 1, 2}) || f.Image.Palette[255] != (RGB{255, 0, 1}) {
		t.Errorf("palette[0]=%v palette[255]=%v", f.Image.Palette[0], f.Image.Palette[255])
	}
}

func TestParse_FixedFont_PlacesCharsOnConsecutiveSizeWidthColumns(t *testing.T) {
	f := parseFntText(t, 12, 4, "[Def]\nSize = 4,4\nType = Fixed\n[Map]\nA\nB\nC\n")
	if f.Type != "fixed" {
		t.Errorf("type = %q", f.Type)
	}
	want := map[rune]Glyph{'A': {0, 4}, 'B': {4, 4}, 'C': {8, 4}}
	for r, g := range want {
		if f.Glyphs[r] != g {
			t.Errorf("glyph %q = %v, want %v", r, f.Glyphs[r], g)
		}
	}
}

func TestParse_MapLine_WithoutWidth_UsesSizeWidthFromGivenX(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 3,2\n[Map]\nA 10\nB\n")
	if f.Glyphs['A'] != (Glyph{10, 3}) {
		t.Errorf("A = %v, want {10 3}", f.Glyphs['A'])
	}
	if f.Glyphs['B'] != (Glyph{13, 3}) {
		t.Errorf("B = %v, want {13 3}", f.Glyphs['B'])
	}
}

func TestParse_MapLine_ExplicitWidthAdvancesRunningOffset(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 3,2\n[Map]\nA 0 7\nB\n")
	if f.Glyphs['B'] != (Glyph{7, 3}) {
		t.Errorf("B = %v, want {7 3}", f.Glyphs['B'])
	}
}

func TestParse_MapHexToken_AcceptsMixedCaseDigitsAndPrefix(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 2,2\n[Map]\n0x5B 0 2\n0X5c 2 2\n0x41\n")
	if f.Glyphs['['] != (Glyph{0, 2}) {
		t.Errorf("'[' = %v", f.Glyphs['['])
	}
	if f.Glyphs['\\'] != (Glyph{2, 2}) {
		t.Errorf("'\\\\' = %v", f.Glyphs['\\'])
	}
	if f.Glyphs['A'] != (Glyph{4, 2}) {
		t.Errorf("A = %v", f.Glyphs['A'])
	}
}

func TestParse_MapLiteralChars_AreCaseSensitive(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 2,2\n[Map]\na\nA\n")
	if f.Glyphs['a'] != (Glyph{0, 2}) || f.Glyphs['A'] != (Glyph{2, 2}) {
		t.Errorf("a=%v A=%v", f.Glyphs['a'], f.Glyphs['A'])
	}
}

func TestParse_MapMultiByteToken_KeepsOnlyFirstByte(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 2,2\n[Map]\nAB 0 2\n")
	if f.Glyphs['A'] != (Glyph{0, 2}) || len(f.Glyphs) != 1 {
		t.Errorf("glyphs = %v", f.Glyphs)
	}
}

func TestParse_MapComments_AreStripped(t *testing.T) {
	text := "[Def]\nSize = 2,2 ; the size\n[Map]\n; whole line\nA 0 2 ; trailing\nB ; only char\n"
	f := parseFntText(t, 20, 2, text)
	if f.Glyphs['A'] != (Glyph{0, 2}) || f.Glyphs['B'] != (Glyph{2, 2}) || len(f.Glyphs) != 2 {
		t.Errorf("glyphs = %v", f.Glyphs)
	}
}

func TestParse_OnlyFirstMapAndFirstDefAreUsed(t *testing.T) {
	text := "[Def]\nSize = 2,2\nType = fixed\n[Map]\nA\n" +
		"[Def]\nSize = 9,9\nType = variable\n[Map]\nZ\n"
	f := parseFntText(t, 20, 2, text)
	if f.Size != (Point{2, 2}) || f.Type != "fixed" {
		t.Errorf("size=%v type=%q", f.Size, f.Type)
	}
	if _, ok := f.Glyphs['Z']; ok || len(f.Glyphs) != 1 {
		t.Errorf("glyphs = %v", f.Glyphs)
	}
}

func TestParse_SectionNamesAndDefKeys_AreCaseInsensitive(t *testing.T) {
	f := parseFntText(t, 20, 2, "[DEF]\nSIZE = 2,2\nTYPE = FIXED\ncolors = 300\n[mAp]\nA\n")
	if f.Size != (Point{2, 2}) || f.Type != "fixed" || f.Colors != 255 {
		t.Errorf("size=%v type=%q colors=%d", f.Size, f.Type, f.Colors)
	}
	if f.Glyphs['A'] != (Glyph{0, 2}) {
		t.Errorf("A = %v", f.Glyphs['A'])
	}
}

func TestParse_Colors_ClampedToOneAtLowEnd(t *testing.T) {
	f := parseFntText(t, 4, 2, "[Def]\nColors = 0\n[Map]\nA 0 1\n")
	if f.Colors != 1 {
		t.Errorf("colors = %d, want 1", f.Colors)
	}
}

func TestParse_Numbers_TakeLeadingIntegerIgnoringGarbage(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 3px, x\nSpacing = +2, -1abc\n[Map]\nA 4z 2\n")
	if f.Size != (Point{3, 0}) {
		t.Errorf("size = %v, want {3 0}", f.Size)
	}
	if f.Spacing != (Point{2, -1}) {
		t.Errorf("spacing = %v, want {2 -1}", f.Spacing)
	}
	if f.Glyphs['A'] != (Glyph{4, 2}) {
		t.Errorf("A = %v, want {4 2}", f.Glyphs['A'])
	}
}

// Ikemen GO (uint16 arithmetic): with ofs=5 and width -3, the glyph takes
// width 3 at X=5, then ofs = 5 + (5+3) + 3 = 16 (not 8).
func TestParse_MapNegativeWidth_UsesAbsoluteWidthAndIkemenOffsetArithmetic(t *testing.T) {
	f := parseFntText(t, 30, 2, "[Def]\nSize = 5,2\n[Map]\nA\nB 5 -3\nC\n")
	if f.Glyphs['B'] != (Glyph{5, 3}) {
		t.Errorf("B = %v, want {5 3}", f.Glyphs['B'])
	}
	if f.Glyphs['C'] != (Glyph{16, 5}) {
		t.Errorf("C = %v, want {16 5}", f.Glyphs['C'])
	}
}

func TestParse_MapEndsAtNextSectionLine(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 2,2\n[Map]\nA\n[Other]\nB\n")
	if _, ok := f.Glyphs['B']; ok {
		t.Errorf("B must not be mapped: %v", f.Glyphs)
	}
}

func TestParse_TextShorterThanDeclared_UsesAvailableBytes(t *testing.T) {
	pcx := buildPCXBlock(t, 20, 2, stripPixels(20, 2), testPalette())
	data := buildFnt(pcx, "[Def]\nSize = 2,2\n[Map]\nA\nB\n")
	binary.LittleEndian.PutUint32(data[28:], 5000)
	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Glyphs['B'] != (Glyph{2, 2}) {
		t.Errorf("B = %v", f.Glyphs['B'])
	}
}

func TestParse_CRLFLines_AreTrimmed(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\r\nSize = 2,2\r\nType = Fixed\r\n[Map]\r\nA\r\n")
	if f.Type != "fixed" || f.Glyphs['A'] != (Glyph{0, 2}) {
		t.Errorf("type=%q A=%v", f.Type, f.Glyphs['A'])
	}
}

func TestParse_ReturnsError_OnBadSignature(t *testing.T) {
	data := buildFnt(buildPCXBlock(t, 4, 2, stripPixels(4, 2), testPalette()), "[Map]\nA\n")
	copy(data, "ElecbyteSff\x00")
	_, err := Parse(data)
	if err == nil || !strings.Contains(err.Error(), "signature") {
		t.Errorf("err = %v, want a signature error", err)
	}
}

func TestParse_ReturnsError_OnTruncatedHeader(t *testing.T) {
	data := buildFnt(buildPCXBlock(t, 4, 2, stripPixels(4, 2), testPalette()), "")
	cases := []struct {
		length int
		want   string
	}{
		{0, "too short for signature"},
		{5, "too short for signature"},
		{12, "truncated header"},
		{31, "truncated header"},
	}
	for _, c := range cases {
		_, err := Parse(data[:c.length])
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("len %d: err = %v, want substring %q", c.length, err, c.want)
		}
	}
}

func TestParse_ReturnsError_WhenPCXBlockBeyondFile(t *testing.T) {
	data := buildFnt(buildPCXBlock(t, 4, 2, stripPixels(4, 2), testPalette()), "[Map]\nA\n")
	binary.LittleEndian.PutUint32(data[16:], 0xFFFFFFF0)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "beyond") {
		t.Errorf("offset beyond file: err = %v, want \"beyond\"", err)
	}
	data = buildFnt(buildPCXBlock(t, 4, 2, stripPixels(4, 2), testPalette()), "[Map]\nA\n")
	binary.LittleEndian.PutUint32(data[20:], 0xFFFFFFFF)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "beyond") {
		t.Errorf("length beyond file: err = %v, want \"beyond\"", err)
	}
}

func TestParse_ReturnsError_WhenPCXBlockTooSmallForHeaderAndPalette(t *testing.T) {
	data := buildFnt(buildPCXBlock(t, 4, 2, stripPixels(4, 2), testPalette()), "[Map]\nA\n")
	binary.LittleEndian.PutUint32(data[20:], 100)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "smaller than header and palette") {
		t.Errorf("err = %v, want \"smaller than header and palette\"", err)
	}
}

func TestParse_ReturnsError_OnCorruptPCX(t *testing.T) {
	pcx := buildPCXBlock(t, 4, 2, stripPixels(4, 2), testPalette())
	pcx[2] = 0 // encoding: not RLE
	data := buildFnt(pcx, "[Map]\nA\n")
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "decoding pcx") {
		t.Errorf("err = %v, want \"decoding pcx\"", err)
	}
}

func TestParse_ReturnsError_WhenTextOffsetBeyondFile(t *testing.T) {
	data := buildFnt(buildPCXBlock(t, 4, 2, stripPixels(4, 2), testPalette()), "[Map]\nA\n")
	binary.LittleEndian.PutUint32(data[24:], 0xFFFFFF00)
	if _, err := Parse(data); err == nil || !strings.Contains(err.Error(), "text block offset") {
		t.Errorf("err = %v, want \"text block offset\"", err)
	}
}

func TestParse_NeverPanics_OnTruncatedValidFile(t *testing.T) {
	data := buildFnt(buildPCXBlock(t, 8, 4, stripPixels(8, 4), testPalette()), "[Def]\nSize=2,4\n[Map]\nA\n")
	for n := 0; n < len(data); n++ {
		_, _ = Parse(data[:n])
	}
}

func TestParse_CorpusFiles_ParseWithGlyphsStartingInsideImage(t *testing.T) {
	dir := os.Getenv("FNT_CORPUS_DIR")
	if dir == "" {
		t.Skip("FNT_CORPUS_DIR not set")
	}
	paths, err := filepath.Glob(filepath.Join(dir, "*.fnt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatalf("no .fnt files in %s", dir)
	}
	for _, p := range paths {
		t.Run(filepath.Base(p), func(t *testing.T) {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			f, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if len(f.Glyphs) == 0 {
				t.Fatal("no glyphs")
			}
			if len(f.Image.Pixels) != f.Image.Width*f.Image.Height {
				t.Errorf("pixels len %d != %dx%d", len(f.Image.Pixels), f.Image.Width, f.Image.Height)
			}
			if f.Size.X <= 0 {
				t.Errorf("Size.X = %d, want > 0", f.Size.X)
			}
			if f.Colors < 1 {
				t.Errorf("Colors = %d, want >= 1", f.Colors)
			}
			if f.Type == "" {
				t.Error("empty Type")
			}
			for r, g := range f.Glyphs {
				// Ikemen GO clips a glyph running past the strip (f-6x8f.fnt's
				// last glyph overhangs by 1 px), so only the start is checked.
				if g.X >= f.Image.Width {
					t.Errorf("glyph %q starts at X=%d, beyond image width %d", r, g.X, f.Image.Width)
				}
			}
		})
	}
}

func TestParse_RealFont_F4x6_GlyphRegions(t *testing.T) {
	dir := os.Getenv("FNT_CORPUS_DIR")
	if dir == "" {
		t.Skip("FNT_CORPUS_DIR not set")
	}
	data, err := os.ReadFile(filepath.Join(dir, "f-4x6.fnt"))
	if err != nil {
		t.Fatalf("f-4x6.fnt missing from FNT_CORPUS_DIR: %v", err)
	}
	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if f.Size != (Point{3, 6}) || f.Type != "variable" {
		t.Errorf("size=%v type=%q", f.Size, f.Type)
	}
	want := map[rune]Glyph{'A': {0, 3}, 'M': {48, 5}, '[': {216, 3}}
	for r, g := range want {
		if f.Glyphs[r] != g {
			t.Errorf("glyph %q = %v, want %v", r, f.Glyphs[r], g)
		}
	}
}

func TestParse_ReturnsError_On24BitPCX(t *testing.T) {
	// 2x1 image, 3 planes, 2 bytes per line: one literal row of R,R,G,G,B,B.
	pcx := make([]byte, pcxHeaderSize)
	pcx[0], pcx[1], pcx[2], pcx[3] = 0x0A, 5, 1, 8
	binary.LittleEndian.PutUint16(pcx[8:], 1)  // xmax
	binary.LittleEndian.PutUint16(pcx[10:], 0) // ymax
	pcx[65] = 3
	binary.LittleEndian.PutUint16(pcx[66:], 2)
	pcx = append(pcx, 1, 2, 3, 4, 5, 6)
	pcx = append(pcx, make([]byte, paletteSize)...)
	_, err := Parse(buildFnt(pcx, "[Map]\nA\n"))
	if err == nil || !strings.Contains(err.Error(), "8-bit") {
		t.Errorf("err = %v, want \"8-bit\"", err)
	}
}

func TestParse_DefQuirks_FollowIkemenArithmetic(t *testing.T) {
	cases := []struct {
		name    string
		def     string
		size    Point
		spacing Point
		colors  int
	}{
		{"empty size x keeps zero", "Size = ,5", Point{0, 5}, Point{}, 1},
		{"empty spacing y keeps zero", "Spacing = 3,", Point{}, Point{3, 0}, 1},
		{"negative size clamps to 0", "Size = -4,-9", Point{0, 0}, Point{}, 1},
		{"size above uint16 clamps to 65535", "Size = 70000,65536", Point{65535, 65535}, Point{}, 1},
		{"huge colors saturate then clamp to 255", "Colors = 99999999999999", Point{}, Point{}, 255},
		{"huge negative colors clamp to 1", "Colors = -99999999999999", Point{}, Point{}, 1},
		{"negative spacing kept", "Spacing = -7,-8", Point{}, Point{-7, -8}, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := parseFntText(t, 4, 2, "[Def]\n"+c.def+"\n[Map]\nA 0 1\n")
			if f.Size != c.size || f.Spacing != c.spacing || f.Colors != c.colors {
				t.Errorf("size=%v spacing=%v colors=%d, want %v %v %d", f.Size, f.Spacing, f.Colors, c.size, c.spacing, c.colors)
			}
		})
	}
}

func TestParse_MapStartXAboveUint16_ClampsTo65535(t *testing.T) {
	f := parseFntText(t, 4, 2, "[Def]\nSize = 2,2\n[Map]\nA 70000 2\n")
	if f.Glyphs['A'] != (Glyph{65535, 2}) {
		t.Errorf("A = %v, want {65535 2}", f.Glyphs['A'])
	}
}

func TestParse_MapBeforeDef_SeesZeroSizeWidth(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Map]\nA\nB 5\nC\n[Def]\nSize = 4,2\n")
	if f.Size != (Point{4, 2}) {
		t.Errorf("size = %v, want {4 2}", f.Size)
	}
	want := map[rune]Glyph{'A': {0, 0}, 'B': {5, 0}, 'C': {5, 0}}
	for r, g := range want {
		if f.Glyphs[r] != g {
			t.Errorf("glyph %q = %v, want %v", r, f.Glyphs[r], g)
		}
	}
}

// The dominant real-file pattern: explicit width on one line, then a bare
// char; the bare char starts where the previous glyph ended and uses Size.w.
func TestParse_VariableFont_BareCharAfterExplicitWidth_StartsAtPreviousEnd(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 3,2\nType = Variable\n[Map]\nA 0 2\nB\n")
	if f.Glyphs['A'] != (Glyph{0, 2}) || f.Glyphs['B'] != (Glyph{2, 3}) {
		t.Errorf("A=%v B=%v, want {0 2} {2 3}", f.Glyphs['A'], f.Glyphs['B'])
	}
}

// Ikemen GO assigns f.images[0][c] for every line, so a repeated char token
// is overwritten by the later line.
func TestParse_DuplicateMapChar_LaterLineWins(t *testing.T) {
	f := parseFntText(t, 20, 2, "[Def]\nSize = 2,2\n[Map]\nA 0 2\nA 6 4\n")
	if len(f.Glyphs) != 1 || f.Glyphs['A'] != (Glyph{6, 4}) {
		t.Errorf("glyphs = %v, want only A={6 4}", f.Glyphs)
	}
}

func FuzzParse(f *testing.F) {
	pcx, err := sff.EncodePCX(&sff.PCXImage{Width: 8, Height: 2, BytesPerPixel: 1, Pixels: stripPixels(8, 2)})
	if err != nil {
		f.Fatal(err)
	}
	pal := testPalette()
	for _, c := range pal {
		pcx = append(pcx, c.R, c.G, c.B)
	}
	f.Add(buildFnt(pcx, "[Def]\nSize = 2,2\nType = Fixed\n[Map]\nA 0 2\n0x42\n"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
	})
}
