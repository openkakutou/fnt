package fnt

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fixtureDir = "testdata/elecbyte"

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Expected values were read by hand. f-4x6.def [Def]: Type = bitmap,
// BankType = palette, Size = 3,6, Spacing = 1,0, Offset = 0,1,
// File = f-4x6.sff (no Colors key, so clamped to 1). Glyphs come from the
// sprite list of f-4x6.sff parsed with python struct: sff v2 header, sprite
// table offset u32 at byte 36, count u32 at 40 (94), 28-byte entries
// (group, image, width, height as u16, axisX, axisY as i16). That gives
// sprite (0,65) 'A' width 3 axisX 0, (0,35) '#' width 5 axisX 0,
// (0,48) '0' width 3.
func TestLoadV2_ElecbyteF4x6_MatchesHandReadValues(t *testing.T) {
	f, err := LoadV2(filepath.Join(fixtureDir, "f-4x6.def"))
	if err != nil {
		t.Fatalf("LoadV2: %v", err)
	}
	if f.Size != (Point{3, 6}) || f.Spacing != (Point{1, 0}) || f.Offset != (Point{0, 1}) {
		t.Errorf("Size/Spacing/Offset = %v %v %v, want {3 6} {1 0} {0 1}", f.Size, f.Spacing, f.Offset)
	}
	if f.Colors != 1 || f.Type != "bitmap" || f.BankType != "palette" || f.File != "f-4x6.sff" {
		t.Errorf("Colors/Type/BankType/File = %d %q %q %q", f.Colors, f.Type, f.BankType, f.File)
	}
	want := map[rune]GlyphV2{
		'A': {0, 65, 3, 0}, '#': {0, 35, 5, 0}, '0': {0, 48, 3, 0},
	}
	for r, g := range want {
		if f.Banks[0][r] != g {
			t.Errorf("glyph %q = %+v, want %+v", r, f.Banks[0][r], g)
		}
	}
	if len(f.Banks) != 1 || len(f.Banks[0]) != 94 {
		t.Errorf("banks = %d, bank 0 glyphs = %d, want 1 and 94", len(f.Banks), len(f.Banks[0]))
	}
}

// font2.sff, same python derivation: (0,65) 'A' width 8, (0,48) '0' width 6.
func TestLoadV2_ElecbyteFont2_MatchesHandReadValues(t *testing.T) {
	f, err := LoadV2(filepath.Join(fixtureDir, "font2.def"))
	if err != nil {
		t.Fatalf("LoadV2: %v", err)
	}
	if f.Banks[0]['A'] != (GlyphV2{0, 65, 8, 0}) || f.Banks[0]['0'] != (GlyphV2{0, 48, 6, 0}) {
		t.Errorf("glyphs A, 0 = %+v, %+v", f.Banks[0]['A'], f.Banks[0]['0'])
	}
}

// Sprite counts read with python struct (sff v2 header: sprite count u32 at
// byte 40): f-4x6 94, font2 94, num1 11. Every sprite is in group 0 and
// BankType is palette, so each font has one bank holding that many glyphs.
func TestLoadV2_ElecbyteFonts_GlyphsMatchSpritesBothWays(t *testing.T) {
	wantCount := map[string]int{"f-4x6": 94, "font2": 94, "num1": 11}
	for name, want := range wantCount {
		t.Run(name, func(t *testing.T) {
			f, err := LoadV2(filepath.Join(fixtureDir, name+".def"))
			if err != nil {
				t.Fatalf("LoadV2: %v", err)
			}
			if len(f.Banks) != 1 || len(f.Banks[0]) != want {
				t.Fatalf("banks = %d, bank 0 glyphs = %d, want 1 and %d", len(f.Banks), len(f.Banks[0]), want)
			}
			sprites := 0
			for _, group := range f.Sprites {
				for _, s := range group.Sprites {
					sprites++
					g, ok := f.Banks[s.Group][rune(s.Image)]
					if !ok {
						t.Errorf("sprite (%d,%d) has no glyph", s.Group, s.Image)
						continue
					}
					if g != (GlyphV2{s.Group, s.Image, s.Width, s.AxisX}) {
						t.Errorf("glyph %+v differs from sprite (%d,%d) width %d axisX %d", g, s.Group, s.Image, s.Width, s.AxisX)
					}
				}
			}
			if sprites != want {
				t.Errorf("loaded %d sprites, want %d", sprites, want)
			}
		})
	}
}

// num1.def read by eye: Size 9,13, Spacing 1,1, Offset 0,0, BankType palette.
// num1.sff, python struct: 11 sprites, all group 0: images 48..57 ('0'-'9')
// 9 wide, and image 111 ('o') 14 wide, all axisX 0. The sff has 8 palettes
// (count u32 at byte 48), but BankType palette keeps only group 0: one bank.
func TestLoadV2_ElecbyteNum1_MatchesHandReadValues(t *testing.T) {
	f, err := LoadV2(filepath.Join(fixtureDir, "num1.def"))
	if err != nil {
		t.Fatalf("LoadV2: %v", err)
	}
	if f.Size != (Point{9, 13}) || f.Spacing != (Point{1, 1}) || f.Offset != (Point{0, 0}) || f.BankType != "palette" {
		t.Errorf("Size/Spacing/Offset/BankType = %v %v %v %q", f.Size, f.Spacing, f.Offset, f.BankType)
	}
	want := map[rune]GlyphV2{'0': {0, 48, 9, 0}, '9': {0, 57, 9, 0}, 'o': {0, 111, 14, 0}}
	for r, g := range want {
		if f.Banks[0][r] != g {
			t.Errorf("glyph %q = %+v, want %+v", r, f.Banks[0][r], g)
		}
	}
	if len(f.Banks) != 1 {
		t.Errorf("banks = %d, want 1", len(f.Banks))
	}
}

func TestLoadV2_Truetype_ReturnsFontWithoutGlyphs(t *testing.T) {
	f, err := LoadV2(filepath.Join(fixtureDir, "msgothic-tt36.def"))
	if err != nil {
		t.Fatalf("LoadV2: %v", err)
	}
	if f.Type != "truetype" || f.File != "msgothic.ttc" || f.Size != (Point{18, 36}) {
		t.Errorf("Type/File/Size = %q %q %v", f.Type, f.File, f.Size)
	}
	if len(f.Banks) != 0 {
		t.Errorf("banks = %d, want 0", len(f.Banks))
	}
}

func tempDef(t *testing.T, def string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "x.def")
	if err := os.WriteFile(path, []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadV2_MissingSff_ErrorNamesTheFile(t *testing.T) {
	path := tempDef(t, string(readFixture(t, "f-4x6.def")))
	_, err := LoadV2(path)
	if err == nil || !strings.Contains(err.Error(), "f-4x6.sff") || !strings.Contains(err.Error(), "not found") || !strings.Contains(err.Error(), filepath.Dir(path)) {
		t.Fatalf("err = %v, want 'not found' naming f-4x6.sff and %s", err, filepath.Dir(path))
	}
}

func TestLoadV2_FileOutsideTheDirectory_IsRefused(t *testing.T) {
	for _, file := range []string{"../f-4x6.sff", `..\f-4x6.sff`, "/etc/passwd", "sub/../../f-4x6.sff"} {
		path := tempDef(t, "[Def]\nType = bitmap\nFile = "+file+"\n")
		if _, err := LoadV2(path); err == nil || !strings.Contains(err.Error(), "outside") {
			t.Errorf("File %q: err = %v, want an 'outside' error", file, err)
		}
	}
}

func TestLoadV2_Symlink_IsRefused(t *testing.T) {
	path := tempDef(t, "[Def]\nType = bitmap\nFile = link.sff\n")
	target, err := filepath.Abs(filepath.Join(fixtureDir, "f-4x6.sff"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(filepath.Dir(path), "link.sff")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := LoadV2(path); err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Errorf("err = %v, want a symlink error", err)
	}
}

func TestLoadV2_CaseInsensitiveAndBackslashFile_Resolves(t *testing.T) {
	path := tempDef(t, "[Def]\nType = bitmap\nFile = sub\\F-4X6.SFF\n")
	sub := filepath.Join(filepath.Dir(path), "SUB")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "f-4x6.sff"), readFixture(t, "f-4x6.sff"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := LoadV2(path)
	if err != nil {
		t.Fatalf("LoadV2: %v", err)
	}
	if len(f.Banks[0]) != 94 {
		t.Errorf("glyphs = %d, want 94", len(f.Banks[0]))
	}
}

func TestParseV2_MalformedSff_ReturnsError(t *testing.T) {
	def := readFixture(t, "f-4x6.def")
	_, err := ParseV2(def, bytes.NewReader([]byte("not an sff at all")))
	if err == nil || !strings.Contains(err.Error(), `reading "f-4x6.sff"`) {
		t.Fatalf("err = %v, want a reading \"f-4x6.sff\" error", err)
	}
}

func TestParseV2_BitmapWithoutSff_ReturnsError(t *testing.T) {
	_, err := ParseV2(readFixture(t, "f-4x6.def"), nil)
	if err == nil || !strings.Contains(err.Error(), "needs the sff data") {
		t.Fatalf("err = %v, want a 'needs the sff data' error", err)
	}
}

func TestLoadV2_MissingDef_ReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope.def")
	_, err := LoadV2(path)
	if err == nil || !strings.Contains(err.Error(), path) {
		t.Fatalf("err = %v, want an error naming %s", err, path)
	}
}

func TestLoadV2_GarbageSff_ErrorWrapsReadingSffAndNamesPath(t *testing.T) {
	path := tempDef(t, "[Def]\nType = bitmap\nFile = bad.sff\n")
	sffPath := filepath.Join(filepath.Dir(path), "bad.sff")
	if err := os.WriteFile(sffPath, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadV2(path)
	if err == nil || !strings.Contains(err.Error(), "reading sff") || !strings.Contains(err.Error(), sffPath) {
		t.Fatalf("err = %v, want 'reading sff' naming %s", err, sffPath)
	}
}

// Bitmap .def without File: Ikemen GO's loadDefInfo loads nothing when the
// file key is empty, so the font has no glyphs and no error.
func TestParseV2_BitmapWithoutFile_HasNoGlyphs(t *testing.T) {
	f, err := ParseV2([]byte("[Def]\nType = bitmap\nSize = 3,6\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Banks) != 0 || f.Size != (Point{3, 6}) {
		t.Errorf("banks = %d, Size = %v", len(f.Banks), f.Size)
	}
}

// Two [Def] sections, as loadFntV2/loadDefInfo (src/font.go) process them
// one by one: the second omits File, so nothing is loaded for it and the
// first section's sprites stay; Size and Spacing keep their value when the
// key is absent; Type and Colors are rewritten (Type "", Colors clamped to 1).
func TestParseV2_SecondDefWithoutFile_KeepsFirstSectionsSprites(t *testing.T) {
	def := "[Def]\nType = bitmap\nSize = 3,4\nColors = 7\nFile = f.sff\n[Def]\nSpacing = 2,2\n"
	f, err := ParseV2([]byte(def), bytes.NewReader(readFixture(t, "f-4x6.sff")))
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Banks[0]) != 94 || f.File != "f.sff" {
		t.Errorf("glyphs = %d, File = %q, want 94 and f.sff", len(f.Banks[0]), f.File)
	}
	if f.Size != (Point{3, 4}) || f.Spacing != (Point{2, 2}) {
		t.Errorf("Size/Spacing = %v %v, want {3 4} {2 2}", f.Size, f.Spacing)
	}
	if f.Type != "" || f.Colors != 1 {
		t.Errorf("Type/Colors = %q %d, want \"\" and 1", f.Type, f.Colors)
	}
}

// Ikemen GO's loadSff (src/image.go) logs "Duplicate sprite key" and keeps
// the first sprite of a key. Entry 2 of f-4x6.sff, (0,35) width 5, becomes
// (0,33): the first entry (0,33) width 3 must win and '#' disappears.
func TestParseV2_DuplicateSprite_FirstOneWins(t *testing.T) {
	sffData := readFixture(t, "f-4x6.sff")
	table := int(binary.LittleEndian.Uint32(sffData[36:40]))
	binary.LittleEndian.PutUint16(sffData[table+2*28+2:], 33)
	f, err := ParseV2([]byte("[Def]\nType = bitmap\nFile = f.sff\n"), bytes.NewReader(sffData))
	if err != nil {
		t.Fatal(err)
	}
	if f.Banks[0]['!'] != (GlyphV2{0, 33, 3, 0}) {
		t.Errorf("'!' = %+v, want width 3 (first entry)", f.Banks[0]['!'])
	}
	if len(f.Banks[0]) != 93 {
		t.Errorf("glyphs = %d, want 93", len(f.Banks[0]))
	}
}

func TestParseV2_BomBeforeFirstSection_IsIgnored(t *testing.T) {
	f, err := ParseV2([]byte("\xef\xbb\xbf[Def]\nType = TrueType\nFile = a.ttf\nSize = 4,5\n"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.Type != "truetype" || f.Size != (Point{4, 5}) {
		t.Errorf("Type/Size = %q %v", f.Type, f.Size)
	}
}

// With BankType = sprite every group is a bank; with palette only group 0.
// No real fixture has a group above 0, so two sprites of f-4x6.sff get
// group 1 by patching the group field of the first two 28-byte sprite
// entries (table offset read from the u32 at header byte 36).
func TestParseV2_BankTypeSprite_GroupsBecomeBanks(t *testing.T) {
	sffData := readFixture(t, "f-4x6.sff")
	table := int(binary.LittleEndian.Uint32(sffData[36:40]))
	binary.LittleEndian.PutUint16(sffData[table:], 1)
	binary.LittleEndian.PutUint16(sffData[table+28:], 1)
	def := "[Def]\nType = bitmap\nBankType = %s\nFile = f.sff\n"
	bySprite, err := ParseV2([]byte(strings.Replace(def, "%s", "Sprite", 1)), bytes.NewReader(sffData))
	if err != nil {
		t.Fatal(err)
	}
	// Sprites (0,33) and (0,34) became (1,33) and (1,34).
	if len(bySprite.Banks[1]) != 2 || bySprite.Banks[1]['!'] != (GlyphV2{1, 33, 3, 0}) {
		t.Errorf("bank 1 = %+v", bySprite.Banks[1])
	}
	if len(bySprite.Banks[0]) != 92 {
		t.Errorf("bank 0 glyphs = %d, want 92", len(bySprite.Banks[0]))
	}
	byPalette, err := ParseV2([]byte(strings.Replace(def, "%s", "palette", 1)), bytes.NewReader(sffData))
	if err != nil {
		t.Fatal(err)
	}
	if len(byPalette.Banks) != 1 || len(byPalette.Banks[0]) != 92 {
		t.Errorf("palette banks = %d, bank 0 glyphs = %d, want 1 and 92", len(byPalette.Banks), len(byPalette.Banks[0]))
	}
}
