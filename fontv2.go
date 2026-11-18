package fnt

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/openkakutou/sff"
)

// GlyphV2 is one character's sprite in the font's .sff: Group and Image
// identify it among FontV2.Sprites, Width and AxisX are copied from it.
type GlyphV2 struct {
	Group, Image, Width, AxisX int
}

// FontV2 is a parsed v2 font: a text .def plus the sprites of its .sff.
type FontV2 struct {
	Size, Spacing, Offset Point
	Colors                int
	Type, BankType, File  string
	// Banks maps a bank (sprite group) to its glyphs, keyed by the sprite's
	// image number as a rune.
	Banks   map[int]map[rune]GlyphV2
	Sprites []sff.SpriteGroup
}

// ParseV2 parses a v2 .def and, for a bitmap font, the sprites in sffData.
// Like Ikemen GO's loadFntV2 (src/font.go) it reads only the [Def] section
// and ignores the others. Each [Def] section is applied in turn as
// loadDefInfo does: Type and Colors are rewritten by every section (absent
// gives "" and 1), while Size, Spacing, Offset and BankType keep their value
// when the key is absent. Ikemen GO loads a file per section that has a File
// key; here one sff is read, that of the last section with a File, and
// BankType is the final one.
// A truetype section has no sprites, sffData may be nil and no glyphs are
// built (Ikemen GO loads the TTF named by File instead). Otherwise a File
// needs sffData, and no File gives no glyphs, as Ikemen GO skips loading.
func ParseV2(def []byte, sffData io.ReaderAt) (*FontV2, error) {
	font, needsSff := parseDefV2(def)
	if !needsSff {
		return font, nil
	}
	if sffData == nil {
		return nil, fmt.Errorf("fnt: font %q needs the sff data of %q", font.Type, font.File)
	}
	err := font.loadSprites(sffData)
	if err != nil {
		return nil, fmt.Errorf("fnt: reading %q: %w", font.File, err)
	}
	return font, nil
}

// parseDefV2 reads the [Def] sections of a v2 .def (loadFntV2, loadDefInfo).
// BankType defaults to "palette" as in Ikemen GO's newFnt. needsSff is true
// when the last section with a File is not truetype.
func parseDefV2(def []byte) (font *FontV2, needsSff bool) {
	text := strings.TrimPrefix(string(def), "\ufeff")
	lines := splitAndTrim(text, "\n")
	shared := Font{}
	font = &FontV2{BankType: "palette", Banks: map[int]map[rune]GlyphV2{}}
	i := 0
	for i < len(lines) {
		name := sectionName(lines[i])
		i++
		if name != "def" {
			continue
		}
		var section map[string]string
		section, i = readSection(lines, i)
		applyDef(section, &shared)
		bankType, hasBankType := section["banktype"]
		if hasBankType {
			font.BankType = strings.ToLower(bankType)
		}
		file := section["file"]
		if file != "" {
			font.File = file
			needsSff = shared.Type != "truetype"
		}
	}
	font.Size, font.Spacing, font.Offset = shared.Size, shared.Spacing, shared.Offset
	font.Colors, font.Type = shared.Colors, shared.Type
	return font, needsSff
}

// loadSprites mirrors LoadFntSff: a sprite becomes a glyph, keyed by its
// image number, when its group is 0 or BankType is "sprite". For a repeated
// group and image the first sprite wins: Ikemen GO's loadSff (src/image.go)
// logs "Duplicate sprite key" and ignores the later one.
//
// Palettes are not resolved here: with BankType "palette", bank n uses the
// sff palette (0, n) (palette 0 when absent), which Sprites lets callers
// look up through the sff package.
func (f *FontV2) loadSprites(sffData io.ReaderAt) error {
	groups, err := sff.Load(sffData)
	if err != nil {
		return err
	}
	f.Sprites = groups
	for _, group := range groups {
		for _, sprite := range group.Sprites {
			if sprite.Group != 0 && f.BankType != "sprite" {
				continue
			}
			bank := f.Banks[sprite.Group]
			if bank == nil {
				bank = map[rune]GlyphV2{}
				f.Banks[sprite.Group] = bank
			}
			_, taken := bank[rune(sprite.Image)]
			if taken {
				continue
			}
			bank[rune(sprite.Image)] = GlyphV2{
				Group: sprite.Group, Image: sprite.Image,
				Width: sprite.Width, AxisX: sprite.AxisX,
			}
		}
	}
	return nil
}

// LoadV2 reads a v2 .def file and, for a bitmap font, the .sff named by its
// File key.
//
// Ikemen GO resolves File with SearchFile over the .def directory, the
// motif directory, the working directory and "data/", under "font/". This
// resolves it beside the .def only: "\" is a separator, each component
// matches exactly or else case-insensitively, and a path that is absolute,
// contains "..", or crosses a symlink is refused. A missing or unreadable
// file is an error naming the path (Ikemen GO panics).
func LoadV2(defPath string) (*FontV2, error) {
	def, err := os.ReadFile(defPath)
	if err != nil {
		return nil, fmt.Errorf("fnt: reading %q: %w", defPath, err)
	}
	font, needsSff := parseDefV2(def)
	if !needsSff {
		return font, nil
	}
	sffPath, err := resolveBeside(filepath.Dir(defPath), font.File)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(sffPath)
	if err != nil {
		return nil, fmt.Errorf("fnt: opening sff %q: %w", sffPath, err)
	}
	defer file.Close()
	err = font.loadSprites(file)
	if err != nil {
		return nil, fmt.Errorf("fnt: reading sff %q: %w", sffPath, err)
	}
	return font, nil
}

// resolveBeside finds the file named by a .def File value under dir.
func resolveBeside(dir, name string) (string, error) {
	normalized := strings.ReplaceAll(name, "\\", "/")
	if strings.HasPrefix(normalized, "/") || filepath.IsAbs(normalized) {
		return "", fmt.Errorf("fnt: sff %q lies outside the font directory", name)
	}
	parts := strings.Split(normalized, "/")
	for _, part := range parts {
		if part == ".." {
			return "", fmt.Errorf("fnt: sff %q lies outside the font directory", name)
		}
	}
	current := dir
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		next, err := matchEntry(current, part)
		if err != nil {
			return "", fmt.Errorf("fnt: sff %q not found in %q: %w", name, dir, err)
		}
		info, err := os.Lstat(next)
		if err != nil {
			return "", fmt.Errorf("fnt: sff %q not found in %q: %w", name, dir, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("fnt: sff %q goes through the symlink %q", name, next)
		}
		current = next
	}
	return current, nil
}

// matchEntry returns dir/name when it exists, else the entry of dir whose
// name equals name ignoring case.
func matchEntry(dir, name string) (string, error) {
	exact := filepath.Join(dir, name)
	_, err := os.Lstat(exact)
	if err == nil {
		return exact, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), name) {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	return "", fs.ErrNotExist
}
