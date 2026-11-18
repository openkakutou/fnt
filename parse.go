package fnt

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/openkakutou/sff"
)

const (
	headerSize     = 32
	signature      = "ElecbyteFnt\x00"
	pcxHeaderSize  = 128
	paletteSize    = 768
	maxUint16Value = math.MaxUint16
)

// mapLineRe splits a [Map] line into char token, optional start x and
// optional width, exactly as Ikemen GO's loadFntV1 does.
var mapLineRe = regexp.MustCompile(`(\S+)(?:\s+(\S+)(?:\s+(\S+))?)?`)

// Parse decodes a v1 .fnt file, following Ikemen GO's loadFntV1
// (src/font.go, https://github.com/ikemen-engine/Ikemen-GO).
//
// Layout (little-endian): 12-byte signature "ElecbyteFnt\x00", ver and ver2
// (uint16 each), then the offset and length of the PCX block and of the text
// block (uint32 each). The PCX block is a 128-byte header, RLE pixels and,
// in its last 768 bytes, the RGB palette. The text block holds the [Def]
// and [Map] sections. Malformed input returns an error, never a panic.
func Parse(data []byte) (*Font, error) {
	if len(data) < len(signature) {
		return nil, fmt.Errorf("fnt: file too short for signature: got %d bytes, need %d", len(data), len(signature))
	}
	if string(data[:len(signature)]) != signature {
		return nil, fmt.Errorf("fnt: unrecognized signature %q, want %q", data[:len(signature)], signature)
	}
	if len(data) < headerSize {
		return nil, fmt.Errorf("fnt: truncated header: got %d bytes, need %d", len(data), headerSize)
	}

	font := &Font{Glyphs: map[rune]Glyph{}}
	font.Ver = binary.LittleEndian.Uint16(data[12:14])
	font.Ver2 = binary.LittleEndian.Uint16(data[14:16])
	pcxOffset := uint64(binary.LittleEndian.Uint32(data[16:20]))
	pcxLength := uint64(binary.LittleEndian.Uint32(data[20:24]))
	txtOffset := uint64(binary.LittleEndian.Uint32(data[24:28]))
	txtLength := uint64(binary.LittleEndian.Uint32(data[28:32]))

	err := parseImage(data, pcxOffset, pcxLength, &font.Image)
	if err != nil {
		return nil, err
	}

	text, err := textBlock(data, txtOffset, txtLength)
	if err != nil {
		return nil, err
	}
	parseText(text, font)
	return font, nil
}

func parseImage(data []byte, offset, length uint64, img *Image) error {
	fileSize := uint64(len(data))
	if length < pcxHeaderSize+paletteSize {
		return fmt.Errorf("fnt: pcx block length %d is smaller than header and palette (%d)", length, pcxHeaderSize+paletteSize)
	}
	if offset > fileSize || length > fileSize-offset {
		return fmt.Errorf("fnt: pcx block [%d, +%d) lies beyond the %d-byte file", offset, length, fileSize)
	}
	block := data[offset : offset+length]

	pcx, err := sff.DecodePCX(block[:length-paletteSize])
	if err != nil {
		return fmt.Errorf("fnt: decoding pcx: %w", err)
	}
	if pcx.BytesPerPixel > 1 {
		return errors.New("fnt: pcx is not 8-bit indexed")
	}
	img.Width = pcx.Width
	img.Height = pcx.Height
	img.Pixels = pcx.Pixels

	rgb := block[length-paletteSize:]
	for i := range img.Palette {
		img.Palette[i] = RGB{rgb[i*3], rgb[i*3+1], rgb[i*3+2]}
	}
	return nil
}

// textBlock returns the text bytes. Like Ikemen GO, a file shorter than the
// declared text length yields the bytes that exist; a block starting at or
// past the end of the file is an error (a zero-length block is always empty).
func textBlock(data []byte, offset, length uint64) (string, error) {
	if length == 0 {
		return "", nil
	}
	fileSize := uint64(len(data))
	if offset >= fileSize {
		return "", fmt.Errorf("fnt: text block offset %d lies beyond the %d-byte file", offset, fileSize)
	}
	end := fileSize
	if length < fileSize-offset {
		end = offset + length
	}
	return string(data[offset:end]), nil
}

// parseText walks the sections in order and uses only the first [Def] and
// the first [Map], as loadFntV1 does. A [Map] placed before [Def] sees a
// zero Size.w, as in Ikemen GO.
func parseText(text string, font *Font) {
	lines := splitAndTrim(text, "\n")
	seenDef, seenMap := false, false
	i := 0
	for {
		name := ""
		for i < len(lines) {
			name = sectionName(lines[i])
			i++
			if name != "" {
				break
			}
		}
		if name == "" {
			break
		}
		if name == "map" && !seenMap {
			seenMap = true
			i = parseMap(lines, i, font)
			continue
		}
		if name == "def" && !seenDef {
			seenDef = true
			i = parseDef(lines, i, font)
		}
	}
}

// sectionName mirrors Ikemen GO's SectionName: "[Name]" (a trailing comment
// is allowed), lowercased up to the first space; "" when not a section line.
func sectionName(line string) string {
	if len(line) == 0 || line[0] != '[' {
		return ""
	}
	trimmed := strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
	if trimmed[len(trimmed)-1] != ']' {
		return ""
	}
	inner := trimmed[1:strings.Index(trimmed, "]")]
	space := strings.Index(inner, " ")
	if space >= 0 {
		inner = inner[:space+1]
	}
	return strings.ToLower(inner)
}

// parseDef reads the [Def] section lines and applies loadDefInfo.
func parseDef(lines []string, i int, font *Font) int {
	section, next := readSection(lines, i)
	applyDef(section, font)
	return next
}

// readSection reads key/value lines until the next section line, first
// occurrence of a key winning (IniSection.Parse). Keys are lowercased.
func readSection(lines []string, i int) (map[string]string, int) {
	section := map[string]string{}
	for ; i < len(lines); i++ {
		line := lines[i]
		if len(line) > 0 && line[0] == '[' {
			break
		}
		line = strings.TrimSpace(strings.SplitN(line, ";", 2)[0])
		end := strings.IndexAny(line, "= \t")
		if end <= 0 {
			continue
		}
		key := strings.ToLower(line[:end])
		value := ""
		equal := strings.Index(line, "=")
		if equal >= 0 {
			value = strings.TrimSpace(line[equal+1:])
		}
		_, exists := section[key]
		if !exists {
			section[key] = value
		}
	}
	return section, i
}

// applyDef is loadDefInfo for the fields v1 and v2 share.
func applyDef(section map[string]string, font *Font) {
	font.Type = strings.ToLower(section["type"])
	font.Size = readPoint(section["size"], font.Size, true)
	font.Spacing = readPoint(section["spacing"], font.Spacing, false)
	font.Offset = readPoint(section["offset"], font.Offset, false)
	font.Colors = clamp(int(atoi(section["colors"])), 1, 255)
}

// readPoint parses "x,y" like loadDefInfo: an empty component keeps the
// previous value. Size components are clamped to uint16 (I32ToU16).
func readPoint(value string, current Point, isSize bool) Point {
	parts := splitAndTrim(value, ",")
	result := current
	if len(parts[0]) > 0 {
		result.X = pointComponent(parts[0], isSize)
	}
	if len(parts) > 1 && len(parts[1]) > 0 {
		result.Y = pointComponent(parts[1], isSize)
	}
	return result
}

func pointComponent(text string, isSize bool) int {
	n := atoi(text)
	if isSize {
		return int(toUint16(n))
	}
	return int(n)
}

// parseMap reads glyph lines until a line starting with "[" (loadFntV1,
// "map" case). Offsets use uint16 arithmetic like Ikemen GO. A negative
// width is Ikemen GO's quirk: the glyph takes the absolute width and the
// running offset becomes ofs + (ofs + |w|) + |w| (uint16 wrap-around).
func parseMap(lines []string, i int, font *Font) int {
	sizeW := uint16(font.Size.X)
	ofs := uint16(0)
	for ; i < len(lines); i++ {
		line := lines[i]
		if len(line) > 0 && line[0] == '[' {
			break
		}
		caps := mapLineRe.FindStringSubmatch(strings.SplitN(line, ";", 2)[0])
		if len(caps) == 0 {
			continue
		}
		char := mapChar(caps[1])
		if len(caps[2]) > 0 {
			ofs = toUint16(atoi(caps[2]))
		}
		glyph := Glyph{X: int(ofs)}
		if len(caps[3]) > 0 {
			width := atoi(caps[3])
			if width < 0 {
				ofs += toUint16(int32(ofs) - width)
				width = 0 - width
			}
			glyphWidth := toUint16(width)
			glyph.Width = int(glyphWidth)
			ofs += glyphWidth - sizeW
		} else {
			glyph.Width = int(sizeW)
		}
		font.Glyphs[char] = glyph
		ofs += sizeW
	}
	return i
}

// mapChar decodes a [Map] char token: 0x/0X hex (digits until the first
// non-hex one), otherwise the token's first byte (multi-byte UTF-8 is
// truncated, as in Ikemen GO).
func mapChar(token string) rune {
	if len(token) >= 2 && token[0] == '0' && (token[1] == 'x' || token[1] == 'X') {
		var c rune
		for _, r := range strings.ToLower(token[2:]) {
			if '0' <= r && r <= '9' {
				c = c<<4 | (r - '0')
				continue
			}
			if 'a' <= r && r <= 'f' {
				c = c<<4 | (r - 'a' + 10)
				continue
			}
			break
		}
		return c
	}
	return rune(token[0])
}

func splitAndTrim(text, sep string) []string {
	parts := strings.Split(text, sep)
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return parts
}

// atoi mirrors Ikemen GO's Atoi: optional sign, leading digits, garbage
// ignored, no digits is 0, out of int32 range saturates (max, or min for
// negatives).
func atoi(text string) int32 {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return 0
	}
	digits := text
	negative := text[0] == '-'
	if text[0] == '-' || text[0] == '+' {
		digits = text[1:]
	}
	limit := int64(math.MaxInt32)
	if negative {
		limit++
	}
	var n int64
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			break
		}
		n = n*10 + int64(digits[i]-'0')
		if n > limit {
			if negative {
				return math.MinInt32
			}
			return math.MaxInt32
		}
	}
	if negative {
		n = -n
	}
	return int32(n)
}

// toUint16 mirrors Ikemen GO's I32ToU16: clamp to [0, 65535].
func toUint16(n int32) uint16 {
	if n < 0 {
		return 0
	}
	if n > maxUint16Value {
		return maxUint16Value
	}
	return uint16(n)
}

func clamp(n, low, high int) int {
	if n < low {
		return low
	}
	if n > high {
		return high
	}
	return n
}
