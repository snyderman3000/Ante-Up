package main

// Art, font and text taken from the player's own copy of Balatro.
// launch.sh unpacks a handful of files from Balatro.exe into data/game the
// first time Ante Up runs (and again if the game file changes). Everything
// here is optional: without it Ante Up falls back to its own font, plain
// colour chips and short built-in descriptions.

import (
	"image"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type Pos struct{ X, Y int }

type Assets struct {
	Font    []byte
	Blinds  *image.RGBA // BlindChips.png, 34×34 frames, 21 per row
	Chips   *image.RGBA // chips.png, stake chips 29×29
	Centers *image.RGBA // Enhancers.png, deck backs 71×95
	Jokers  *image.RGBA // Jokers.png, 71×95
	Names   map[string]string
	Text    map[string]string
	Pos     map[string]Pos
	SoulPos map[string]Pos
}

func loadPNG(path string) *image.RGBA {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil
	}
	return toRGBA(img)
}

func LoadAssets(dir string) *Assets {
	a := &Assets{Names: map[string]string{}, Text: map[string]string{}, Pos: map[string]Pos{}, SoulPos: map[string]Pos{}}
	if b, err := os.ReadFile(filepath.Join(dir, "m6x11plus.ttf")); err == nil {
		a.Font = b
	}
	a.Blinds = loadPNG(filepath.Join(dir, "BlindChips.png"))
	a.Chips = loadPNG(filepath.Join(dir, "chips.png"))
	a.Centers = loadPNG(filepath.Join(dir, "Enhancers.png"))
	a.Jokers = loadPNG(filepath.Join(dir, "Jokers.png"))
	if b, err := os.ReadFile(filepath.Join(dir, "en-us.lua")); err == nil {
		parseLocalization(string(b), a)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "game.lua")); err == nil {
		parsePositions(string(b), a)
	}
	return a
}

var (
	locEntry = regexp.MustCompile(`(?s)\b([a-z]{1,5}_[a-z0-9_]+)=\{\s*name="((?:[^"\\]|\\.)*)",\s*text=\{((?:\s*"(?:[^"\\]|\\.)*",?)*)\s*\}`)
	locLine  = regexp.MustCompile(`"((?:[^"\\]|\\.)*)"`)
	locFmt   = regexp.MustCompile(`\{[^}]*\}`)
	posLine  = regexp.MustCompile(`(?m)^\s*([a-z]{1,5}_[a-z0-9_]+)\s*=\s*\{.*?[^_]pos\s*=\s*\{\s*x\s*=\s*(\d+)\s*,\s*y\s*=\s*(\d+)\s*\}(.*)$`)
	soulPos  = regexp.MustCompile(`soul_pos\s*=\s*\{\s*x\s*=\s*(\d+)\s*,\s*y\s*=\s*(\d+)\s*\}`)
)

func parseLocalization(s string, a *Assets) {
	for _, m := range locEntry.FindAllStringSubmatch(s, -1) {
		key := m[1]
		if _, dup := a.Names[key]; dup {
			continue // first one wins (descriptions come before misc copies)
		}
		a.Names[key] = unescape(m[2])
		var lines []string
		for _, l := range locLine.FindAllStringSubmatch(m[3], -1) {
			lines = append(lines, strings.TrimSpace(locFmt.ReplaceAllString(unescape(l[1]), "")))
		}
		a.Text[key] = strings.Join(strings.Fields(strings.Join(lines, " ")), " ")
	}
}

func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	if u, err := strconv.Unquote(`"` + s + `"`); err == nil {
		return u
	}
	return s
}

func parsePositions(s string, a *Assets) {
	for _, m := range posLine.FindAllStringSubmatch(s, -1) {
		x, _ := strconv.Atoi(m[2])
		y, _ := strconv.Atoi(m[3])
		if _, dup := a.Pos[m[1]]; !dup {
			a.Pos[m[1]] = Pos{x, y}
		}
		if sm := soulPos.FindStringSubmatch(m[4]); sm != nil {
			sx, _ := strconv.Atoi(sm[1])
			sy, _ := strconv.Atoi(sm[2])
			a.SoulPos[m[1]] = Pos{sx, sy}
		}
	}
}

func sub(img *image.RGBA, x, y, w, h int) *image.RGBA {
	if img == nil {
		return nil
	}
	r := image.Rect(x*w, y*h, x*w+w, y*h+h).Add(img.Rect.Min)
	if !r.In(img.Rect) {
		return nil
	}
	return img.SubImage(r).(*image.RGBA)
}

func (a *Assets) BlindChip(b *BossInfo) *image.RGBA { return sub(a.Blinds, 0, b.Row, 34, 34) }

func (a *Assets) StakeChip(s int) *image.RGBA {
	if s < 1 || s > 8 {
		return nil
	}
	return sub(a.Chips, Stakes[s].ChipX, Stakes[s].ChipY, 29, 29)
}

func (a *Assets) DeckBack(key string) *image.RGBA {
	p, ok := a.Pos[key]
	if !ok {
		return nil
	}
	return sub(a.Centers, p.X, p.Y, 71, 95)
}

func (a *Assets) Joker(key string) (*image.RGBA, *image.RGBA) {
	p, ok := a.Pos[key]
	if !ok {
		return nil, nil
	}
	var soul *image.RGBA
	if sp, ok := a.SoulPos[key]; ok {
		soul = sub(a.Jokers, sp.X, sp.Y, 71, 95)
	}
	return sub(a.Jokers, p.X, p.Y, 71, 95), soul
}

func (a *Assets) BossText(b *BossInfo, mostHand string) string {
	t := a.Text[b.Key]
	if t == "" {
		return b.Effect
	}
	if strings.Contains(t, "#1#") {
		h := mostHand
		if h == "" {
			h = "your most played hand"
		}
		t = strings.ReplaceAll(t, "#1#", h)
	}
	return t
}
