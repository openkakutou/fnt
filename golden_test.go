package fnt

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files in testdata/elecbyte")

type goldenGlyph struct {
	Char  int `json:"char"`
	X     int `json:"x"`
	Width int `json:"width"`
}

type goldenSummary struct {
	Ver          uint16        `json:"ver"`
	Ver2         uint16        `json:"ver2"`
	Size         Point         `json:"size"`
	Spacing      Point         `json:"spacing"`
	Offset       Point         `json:"offset"`
	Colors       int           `json:"colors"`
	Type         string        `json:"type"`
	ImageWidth   int           `json:"imageWidth"`
	ImageHeight  int           `json:"imageHeight"`
	PixelsSHA256 string        `json:"pixelsSha256"`
	PaletteSHA   string        `json:"paletteSha256"`
	Glyphs       []goldenGlyph `json:"glyphs"`
}

func summarize(f *Font) goldenSummary {
	palette := make([]byte, 0, 768)
	for _, c := range f.Image.Palette {
		palette = append(palette, c.R, c.G, c.B)
	}
	pixelsSum := sha256.Sum256(f.Image.Pixels)
	paletteSum := sha256.Sum256(palette)
	glyphs := make([]goldenGlyph, 0, len(f.Glyphs))
	for r, g := range f.Glyphs {
		glyphs = append(glyphs, goldenGlyph{Char: int(r), X: g.X, Width: g.Width})
	}
	sort.Slice(glyphs, func(i, j int) bool { return glyphs[i].Char < glyphs[j].Char })
	return goldenSummary{
		Ver: f.Ver, Ver2: f.Ver2,
		Size: f.Size, Spacing: f.Spacing, Offset: f.Offset,
		Colors: f.Colors, Type: f.Type,
		ImageWidth: f.Image.Width, ImageHeight: f.Image.Height,
		PixelsSHA256: hex.EncodeToString(pixelsSum[:]),
		PaletteSHA:   hex.EncodeToString(paletteSum[:]),
		Glyphs:       glyphs,
	}
}

// diffSummaries describes the first difference between two summaries, or "".
func diffSummaries(got, want goldenSummary) string {
	g := got
	w := want
	g.Glyphs, w.Glyphs = nil, nil
	gj, _ := json.Marshal(g)
	wj, _ := json.Marshal(w)
	if string(gj) != string(wj) {
		return fmt.Sprintf("fields differ:\n got  %s\n want %s", gj, wj)
	}
	if len(got.Glyphs) != len(want.Glyphs) {
		return fmt.Sprintf("glyph count: got %d, want %d", len(got.Glyphs), len(want.Glyphs))
	}
	for i := range got.Glyphs {
		if got.Glyphs[i] != want.Glyphs[i] {
			return fmt.Sprintf("first differing glyph #%d: got %+v, want %+v", i, got.Glyphs[i], want.Glyphs[i])
		}
	}
	return ""
}

func TestParse_ElecbyteFonts_MatchGoldenFiles(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "elecbyte", "*.fnt"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 7 {
		t.Fatalf("found %d fixtures, want 7", len(paths))
	}
	for _, p := range paths {
		name := strings.TrimSuffix(filepath.Base(p), ".fnt")
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			f, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			got := summarize(f)
			goldenPath := strings.TrimSuffix(p, ".fnt") + ".golden.json"
			if *update {
				out, err := json.MarshalIndent(got, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, append(out, '\n'), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			raw, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("golden missing (run with -update): %v", err)
			}
			var want goldenSummary
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatalf("bad golden: %v", err)
			}
			if d := diffSummaries(got, want); d != "" {
				t.Errorf("%s differs from golden:\n%s", name, d)
			}
		})
	}
}

func parseFixture(t *testing.T, name string) *Font {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "elecbyte", name))
	if err != nil {
		t.Fatal(err)
	}
	f, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return f
}

// Expected values below were read by hand from the fixture bytes, not from
// Parse: the text block ([Def] and [Map] lines) of f-4x6.fnt, and the PCX
// header at file offset 64 (xmax-xmin+1 = 280, ymax-ymin+1 = 6).
func TestParse_ElecbyteF4x6_MatchesHandReadValues(t *testing.T) {
	f := parseFixture(t, "f-4x6.fnt")
	if f.Size != (Point{3, 6}) {
		t.Errorf("Size = %v, want {3 6}", f.Size)
	}
	if f.Spacing != (Point{1, 0}) {
		t.Errorf("Spacing = %v, want {1 0}", f.Spacing)
	}
	if f.Offset != (Point{0, -1}) {
		t.Errorf("Offset = %v, want {0 -1}", f.Offset)
	}
	if f.Colors != 1 {
		t.Errorf("Colors = %d, want 1", f.Colors)
	}
	if f.Type != "variable" {
		t.Errorf("Type = %q, want variable", f.Type)
	}
	if f.Image.Width != 280 || f.Image.Height != 6 {
		t.Errorf("image = %dx%d, want 280x6", f.Image.Width, f.Image.Height)
	}
	want := map[rune]Glyph{
		'A': {0, 3}, 'M': {48, 5}, 'W': {90, 5}, '0': {108, 3},
		'[': {216, 3}, // "0x5B 216 3" line
	}
	for r, g := range want {
		if f.Glyphs[r] != g {
			t.Errorf("glyph %q = %v, want %v", r, f.Glyphs[r], g)
		}
	}
}

// jg.fnt is Type=Fixed, Size 8,8, and its [Map] lists bare characters in
// order: ! " # $ % & ' { } * + , - . / 0 ... : ] < = >. A glyph's X is its
// zero-based line index times 8: '0' is line 15 (120) and '>' line 29 (232), each 8 wide.
// ']' is listed twice, at lines 26 and 60; the later line wins (60*8 = 480).
func TestParse_ElecbyteJgFixed_GlyphsFollowOrderTimesSize(t *testing.T) {
	f := parseFixture(t, "jg.fnt")
	if f.Type != "fixed" {
		t.Fatalf("Type = %q, want fixed", f.Type)
	}
	want := map[rune]Glyph{'0': {120, 8}, ']': {480, 8}, '>': {232, 8}}
	for r, g := range want {
		if f.Glyphs[r] != g {
			t.Errorf("glyph %q = %v, want %v", r, f.Glyphs[r], g)
		}
	}
}
