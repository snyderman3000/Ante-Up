package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	tabRun = iota
	tabDecks
	tabJokers
	numTabs
)

var tabNames = []string{"RUN", "DECKS", "JOKERS"}

const (
	sortWins = iota
	sortRate
	sortPicked
	sortName
	numSorts
)

var sortNames = []string{"most wins", "win rate", "most picked", "A-Z"}

type UI struct {
	env  *Env
	a    *Assets
	prof *Profile
	run  *Run
	hist *History

	profErr, runErr string
	profileSlot     string

	tab      int
	bossCur  int
	deckRow  int // 0..len(Decks) (last = all decks)
	deckCol  int // 1..8 stakes, 9 = total
	jokerCur int
	jokerTop int
	jokerSrt int
	jokers   []*JokerStats

	fS, fL *Font // body (2× pixel font) and headings (3×)
	fM     *Font // dense tables (1.5×)
	pixel  bool  // true when using Balatro's own font
	toast  string
	toastT time.Time
	quit   bool
	bg     *Canvas
}

func NewUI(env *Env, w, h int) *UI {
	u := &UI{env: env, deckCol: 1}
	u.a = LoadAssets(filepath.Join(env.DataDir, "game"))
	if u.a.Font != nil {
		if f, err := tryFont(u.a.Font, 32); err == nil {
			u.fS = f
			u.fL, _ = tryFont(u.a.Font, 48)
			u.fM, _ = tryFont(u.a.Font, 24)
			u.pixel = true
		}
	}
	if u.fS == nil {
		u.fS = loadFont(fallbackTTF, 28)
		u.fL = loadFont(fallbackTTF, 40)
	}
	if u.fM == nil {
		u.fM = loadFont(fallbackTTF, 22)
	}
	u.hist = LoadHistory(env.DataDir)
	u.Reload()
	return u
}

func tryFont(b []byte, size float64) (f *Font, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return loadFont(b, size), nil
}

// Reload re-reads the profile and the run save.
func (u *UI) Reload() {
	slot := "1"
	if s, err := ReadJKR(filepath.Join(u.env.SaveDir, "settings.jkr")); err == nil {
		if p := s.Str("profile"); p != "" {
			slot = p
		}
	}
	u.profileSlot = slot
	dir := filepath.Join(u.env.SaveDir, slot)
	u.prof, u.profErr = nil, ""
	if p, err := LoadProfile(filepath.Join(dir, "profile.jkr"), u.a.Names); err != nil {
		u.profErr = err.Error()
		log.Printf("profile: %v", err)
	} else {
		u.prof = p
	}
	u.run, u.runErr = nil, ""
	if r, err := LoadRun(filepath.Join(dir, "save.jkr"), u.hist); err != nil {
		u.runErr = err.Error()
		log.Printf("run: %v", err)
	} else {
		u.run = r
		for i, row := range r.Rows {
			if row.State == BossCurrent || (r.BossPhase == "Defeated" && row.Info.Key == r.BossKey) {
				u.bossCur = i
			}
		}
	}
	u.sortJokers()
}

func (u *UI) say(s string) { u.toast, u.toastT = s, time.Now().Add(2*time.Second) }

func (u *UI) sortJokers() {
	u.jokers = nil
	if u.prof == nil {
		return
	}
	u.jokers = append(u.jokers, u.prof.Jokers...)
	byName := func(i, j int) bool { return u.jokers[i].Name < u.jokers[j].Name }
	sort.SliceStable(u.jokers, func(i, j int) bool {
		a, b := u.jokers[i], u.jokers[j]
		switch u.jokerSrt {
		case sortWins:
			if a.Total.W != b.Total.W {
				return a.Total.W > b.Total.W
			}
			if a.Total.L != b.Total.L {
				return a.Total.L < b.Total.L
			}
			return a.Count > b.Count
		case sortRate:
			// rank by rate, but runs with more games first on ties; no-game jokers last
			ra, rb := a.Total.Rate(), b.Total.Rate()
			if ra != rb {
				return ra > rb
			}
			return a.Total.Games() > b.Total.Games()
		case sortPicked:
			if a.Count != b.Count {
				return a.Count > b.Count
			}
		}
		return byName(i, j)
	})
	if u.jokerCur >= len(u.jokers) {
		u.jokerCur = 0
	}
}

// ---------- input ----------

func (u *UI) Key(k int) {
	switch k {
	case BtnB, BtnMenu:
		u.quit = true
		return
	case BtnL1:
		u.tab = (u.tab + numTabs - 1) % numTabs
		return
	case BtnR1:
		u.tab = (u.tab + 1) % numTabs
		return
	case BtnSelect, BtnStart:
		u.Reload()
		u.say("Reloaded from your save")
		return
	}
	switch u.tab {
	case tabRun:
		u.keyRun(k)
	case tabDecks:
		u.keyDecks(k)
	case tabJokers:
		u.keyJokers(k)
	}
}

const bossCols = 4

func (u *UI) keyRun(k int) {
	n := len(Bosses)
	switch k {
	case BtnLeft:
		u.bossCur = (u.bossCur + n - 1) % n
	case BtnRight:
		u.bossCur = (u.bossCur + 1) % n
	case BtnUp:
		if u.bossCur-bossCols >= 0 {
			u.bossCur -= bossCols
		}
	case BtnDown:
		if u.bossCur+bossCols < n {
			u.bossCur += bossCols
		}
	}
}

func (u *UI) keyDecks(k int) {
	switch k {
	case BtnLeft:
		u.deckCol = (u.deckCol+7)%9 + 1
	case BtnRight:
		u.deckCol = u.deckCol%9 + 1
	case BtnUp:
		u.deckRow = (u.deckRow + len(Decks)) % (len(Decks) + 1)
	case BtnDown:
		u.deckRow = (u.deckRow + 1) % (len(Decks) + 1)
	}
}

const jokerRows = 13

func (u *UI) keyJokers(k int) {
	n := len(u.jokers)
	if k == BtnY || k == BtnX {
		u.jokerSrt = (u.jokerSrt + 1) % numSorts
		u.sortJokers()
		u.jokerCur, u.jokerTop = 0, 0
		u.say("Sorted by " + sortNames[u.jokerSrt])
		return
	}
	if n == 0 {
		return
	}
	switch k {
	case BtnUp:
		u.jokerCur = (u.jokerCur + n - 1) % n
	case BtnDown:
		u.jokerCur = (u.jokerCur + 1) % n
	case BtnLeft:
		u.jokerCur = max(0, u.jokerCur-jokerRows)
	case BtnRight:
		u.jokerCur = min(n-1, u.jokerCur+jokerRows)
	}
	if u.jokerCur < u.jokerTop {
		u.jokerTop = u.jokerCur
	}
	if u.jokerCur >= u.jokerTop+jokerRows {
		u.jokerTop = u.jokerCur - jokerRows + 1
	}
}

// ---------- drawing helpers ----------

// textY returns the y to pass to Text so capitals sit vertically centred in a
// box starting at top with height h.
func (u *UI) textY(f *Font, top, h int) int {
	g := f.glyph('H')
	capTop := f.ascent + g.off.Y
	capH := 0
	if g.mask != nil {
		capH = g.mask.Bounds().Dy()
	}
	return top + (h-capH)/2 - capTop
}

func (u *UI) panel(c *Canvas, x, y, w, h int, col color.RGBA) {
	c.RoundRect(x, y+3, w, h, 6, mix(col, colBlack, 0.45))
	c.RoundRect(x, y, w, h, 6, col)
}

func (u *UI) pill(c *Canvas, x, y, w, h int, s string, col, txt color.RGBA) {
	u.panel(c, x, y, w, h, col)
	c.TextCenter(u.fS, x+w/2, u.textY(u.fS, y, h), s, txt)
}

func (u *UI) buildBG(c *Canvas) {
	u.bg = NewCanvas(c.W, c.H)
	b := u.bg
	cx, cy := float64(c.W)/2, float64(c.H)*0.42
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			dx, dy := (float64(x)-cx)/float64(c.W), (float64(y)-cy)/float64(c.H)
			d := math.Sqrt(dx*dx + dy*dy)
			// faint swirl bands, like the game's background
			a := math.Atan2(dy, dx)
			band := 0.5 + 0.5*math.Sin(a*2+d*14)
			t := math.Min(1, d*1.5)
			col := mix(colFelt, colBG, t)
			col = mix(col, mix(colFelt, colGreen, 0.25), band*0.12*(1-t))
			b.Set(x, y, col)
		}
	}
}

func (u *UI) Draw(c *Canvas) {
	if u.bg == nil || u.bg.W != c.W {
		u.buildBG(c)
	}
	copy(c.Pix, u.bg.Pix)
	u.drawHeader(c)
	switch u.tab {
	case tabRun:
		u.drawRun(c)
	case tabDecks:
		u.drawDecks(c)
	case tabJokers:
		u.drawJokers(c)
	}
	u.drawFooter(c)
	if u.toast != "" && time.Now().Before(u.toastT) {
		w := u.fS.Width(u.toast) + 28
		x := (c.W - w) / 2
		u.panel(c, x, 396, w, 36, colOrange)
		c.TextCenter(u.fS, c.W/2, u.textY(u.fS, 396, 36), u.toast, colWhite)
	}
}

func (u *UI) drawHeader(c *Canvas) {
	c.Fill(0, 0, c.W, 46, colPanelDk)
	c.Fill(0, 46, c.W, 2, colEdge)
	y := u.textY(u.fL, 0, 46)
	x := c.Text(u.fL, 12, y+2, "ANTE", colBlack)
	c.Text(u.fL, 12, y, "ANTE", colRed)
	c.Text(u.fL, x+12, y+2, "UP", colBlack)
	c.Text(u.fL, x+10, y, "UP", colBlue)
	// tabs, right-aligned
	tx := c.W - 10
	for i := numTabs - 1; i >= 0; i-- {
		w := u.fS.Width(tabNames[i]) + 24
		tx -= w
		col, txt := colPanel2, colDim
		if i == u.tab {
			col, txt = colRed, colWhite
		}
		u.pill(c, tx, 7, w, 30, tabNames[i], col, txt)
		tx -= 6
	}
}

func (u *UI) drawFooter(c *Canvas) {
	y0 := c.H - 34
	c.Fill(0, y0, c.W, 34, colPanelDk)
	c.Fill(0, y0, c.W, 2, colEdge)
	var hints [][2]string
	switch u.tab {
	case tabRun:
		hints = [][2]string{{"✥", "boss"}}
	case tabDecks:
		hints = [][2]string{{"✥", "deck/stake"}}
	case tabJokers:
		hints = [][2]string{{"↕", "joker"}, {"Y", "sort"}}
	}
	hints = append(hints, [2]string{"L/R", "tab"}, [2]string{"SEL", "reload"}, [2]string{"B", "quit"})
	x := 10
	ty := u.textY(u.fS, y0+2, 32)
	for _, h := range hints {
		bw := u.fS.Width(h[0]) + 12
		c.RoundRect(x, y0+5, bw, 26, 5, colPanel2)
		c.TextCenter(u.fS, x+bw/2, u.textY(u.fS, y0+5, 26), h[0], colWhite)
		x = c.Text(u.fS, x+bw+6, ty, h[1], colText) + 16
	}
}

func pct(r Record) string {
	if r.Rate() < 0 {
		return "-"
	}
	return strconv.Itoa(r.Rate()) + "%"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return "1 " + one
	}
	return strconv.Itoa(n) + " " + many
}

// ---------- RUN tab ----------

func shortBoss(b *BossInfo) string {
	n := strings.TrimPrefix(b.Name, "The ")
	if b.Showdown {
		if i := strings.IndexByte(n, ' '); i > 0 {
			n = n[i+1:]
		}
	}
	return n
}

func (u *UI) drawRun(c *Canvas) {
	if u.run == nil {
		u.panel(c, 40, 120, c.W-80, 200, colPanel)
		c.TextCenter(u.fL, c.W/2, 150, "No run in progress", colText)
		msg := "Start a run in Balatro, then open Ante Up again (or press SELECT to reload)."
		for i, l := range u.fS.Wrap(msg, c.W-130) {
			c.TextCenter(u.fS, c.W/2, 214+i*30, l, colDim)
		}
		return
	}
	r := u.run
	// --- run info (left) ---
	u.panel(c, 8, 56, 300, 96, colPanel)
	if back := u.a.DeckBack(r.DeckKey); back != nil {
		c.Sprite(back, 16, 61, 64, 86, 0)
	} else {
		c.RoundRect(16, 61, 64, 86, 5, colRed)
	}
	if st := u.a.StakeChip(r.Stake); st != nil {
		c.Sprite(st, 62, 125, 29, 29, 0)
	}
	x := 92
	c.Text(u.fS, x, u.textY(u.fS, 60, 28), u.fS.Ellipsize(r.DeckName, 210), colWhite)
	sc := Stakes[min(max(r.Stake, 1), 8)].Col
	c.Text(u.fS, x, u.textY(u.fS, 88, 28), stakeName(r.Stake), sc)
	line := fmt.Sprintf("Ante %d  $%d", r.Ante, r.Dollars)
	if r.Won {
		line = fmt.Sprintf("Ante %d  Endless", r.Ante)
	}
	c.Text(u.fS, x, u.textY(u.fS, 116, 28), line, colGold)

	// --- selected boss (right) ---
	row := r.Rows[u.bossCur]
	b := row.Info
	px := 316
	u.panel(c, px, 56, c.W-px-8, 96, colPanel)
	if chip := u.a.BlindChip(b); chip != nil {
		c.Sprite(chip, px+6, 60, 34, 34, grey(row))
	} else {
		c.Circle(px+23, 77, 15, b.Col, true, 0)
	}
	right := c.W - 16
	c.Text(u.fS, px+46, u.textY(u.fS, 58, 21), u.fS.Ellipsize(b.Name, right-px-46), mix(b.Col, colWhite, 0.45))
	status, scol := u.bossStatus(row)
	c.Text(u.fM, px+46, u.textY(u.fM, 80, 16), u.fM.Ellipsize(status, right-px-46), scol)
	ew := right - px - 8
	el := u.fM.Wrap(u.a.BossText(b, r.MostHand), ew)
	if len(el) > 2 {
		el = []string{el[0], u.fM.Ellipsize(strings.Join(el[1:], " "), ew)}
	}
	for i, l := range el {
		c.Text(u.fM, px+8, u.textY(u.fM, 104+i*20, 16), l, colText)
	}

	// --- board ---
	gx, gy, cw, ch := 8, 160, 155, 38
	for i, rw := range r.Rows {
		col, rowi := i%bossCols, i/bossCols
		x := gx + col*(cw+3)
		y := gy + rowi*(ch+3)
		u.drawBossCell(c, x, y, cw, ch, rw, i == u.bossCur)
	}
}

func grey(r *BossRow) float64 {
	switch r.State {
	case BossLater, BossBanned, BossWaiting:
		return 0.85
	}
	return 0
}

func (u *UI) bossStatus(row *BossRow) (string, color.RGBA) {
	r := u.run
	switch row.State {
	case BossCurrent:
		if r.BossPhase == "Current" {
			return "Fighting it now", colRed
		}
		return fmt.Sprintf("Boss of ante %d", r.Ante), colOrange
	case BossBeaten:
		if r.BossPhase == "Defeated" && row.Info.Key == r.BossKey {
			return "Just beaten!", colGreen
		}
		s := "Beaten this run"
		if row.Used > 1 {
			s = fmt.Sprintf("Beaten (came up %dx)", row.Used)
		}
		return s, colGreen
	case BossSeen:
		return "Seen (beaten or rerolled)", colGreen
	case BossPool:
		return fmt.Sprintf("Could be ante %d boss", r.PickAnte), colBlue
	case BossWaiting:
		return "Not yet - others first", colDim
	case BossBanned:
		return "Banned this run", colFaint
	}
	if row.Info.Showdown {
		return fmt.Sprintf("Final boss, ante %d", row.From), colDim
	}
	return fmt.Sprintf("Can't appear until ante %d", row.From), colDim
}

func (u *UI) drawBossCell(c *Canvas, x, y, w, h int, row *BossRow, sel bool) {
	b := row.Info
	bg := colPanel
	switch row.State {
	case BossBeaten, BossSeen:
		bg = mix(colPanel, colGreen, 0.22)
	case BossCurrent:
		bg = mix(colPanel, colRed, 0.45)
	case BossPool:
		bg = mix(colPanel, colBlue, 0.22)
	case BossLater, BossBanned, BossWaiting:
		bg = colPanelDk
	}
	if sel {
		c.RoundRect(x-2, y-2, w+4, h+4, 7, colWhite)
	}
	c.RoundRect(x, y, w, h, 5, bg)
	if b.Showdown && !sel {
		c.RoundRect(x, y+h-3, w, 3, 1, mix(bg, colGold, 0.6))
	}
	if chip := u.a.BlindChip(b); chip != nil {
		c.Sprite(chip, x+2, y+2, 34, 34, grey(row))
	} else {
		cc := b.Col
		if grey(row) > 0 {
			cc = mix(cc, colPanelDk, 0.7)
		}
		c.Circle(x+19, y+19, 15, cc, true, 0)
	}
	txt := colText
	if grey(row) > 0 {
		txt = colFaint
	}
	// status badge in the chip's bottom-right corner
	bx, by := x+30, y+h-9
	switch row.State {
	case BossBeaten, BossSeen:
		c.Circle(bx, by, 8, colGreen, true, 0)
		c.TextCenter(u.fS, bx+1, u.textY(u.fS, by-8, 16), "✓", colWhite)
	case BossCurrent:
		c.Circle(bx, by, 8, colGold, true, 0)
		c.TextCenter(u.fS, bx+1, u.textY(u.fS, by-8, 16), "★", colWhite)
	case BossPool:
		c.Circle(bx, by, 8, colBlue, true, 0)
		c.TextCenter(u.fS, bx+1, u.textY(u.fS, by-8, 16), "?", colWhite)
	case BossLater:
		c.TextCenter(u.fS, x+19, u.textY(u.fS, y, h), strconv.Itoa(row.From), colText)
	case BossBanned:
		c.TextCenter(u.fS, x+19, u.textY(u.fS, y, h), "×", colRed)
	}
	c.Text(u.fS, x+42, u.textY(u.fS, y, h), u.fS.Ellipsize(shortBoss(b), x+w-4-(x+42)), txt)
}

// ---------- DECKS tab ----------

func (u *UI) drawDecks(c *Canvas) {
	p := u.prof
	if p == nil {
		u.panel(c, 40, 140, c.W-80, 120, colPanel)
		c.TextCenter(u.fL, c.W/2, 160, "No profile found", colText)
		c.TextCenter(u.fS, c.W/2, 214, "Play a run of Balatro first.", colDim)
		return
	}
	nameW, colW, x0 := 136, 54, 8
	sx := x0 + nameW
	totX := sx + 8*colW
	totW := c.W - 8 - totX
	// stake chips header
	for s := 1; s <= 8; s++ {
		cx := sx + (s-1)*colW + colW/2
		if s == u.deckCol {
			c.RoundRect(cx-colW/2+1, 52, colW-2, 34, 5, colPanel2)
		}
		if chip := u.a.StakeChip(s); chip != nil {
			c.Sprite(chip, cx-14, 54, 29, 29, 0)
		} else {
			c.Circle(cx, 69, 12, Stakes[s].Col, true, 0)
		}
	}
	if u.deckCol == 9 {
		c.RoundRect(totX+1, 52, totW-2, 34, 5, colPanel2)
	}
	c.TextCenter(u.fS, totX+totW/2, u.textY(u.fS, 52, 34), "ALL", colText)
	c.Text(u.fS, x0+4, u.textY(u.fS, 52, 34), "DECK", colDim)

	rowH, y0 := 18, 90
	rows := len(Decks) + 1
	for i := 0; i < rows; i++ {
		y := y0 + i*rowH
		var name string
		var recs [9]Record
		var tot Record
		best := 0
		if i < len(Decks) {
			d := p.Decks[i]
			name = strings.TrimSuffix(d.Name, " Deck")
			recs, tot, best = d.ByStake, d.Total, d.BestWon
		} else {
			y += 4
			name = "All decks"
			recs, tot = p.StakeTotals, p.Total
			c.Fill(x0, y-3, c.W-16, 1, colEdge)
		}
		selRow := i == u.deckRow
		if selRow {
			c.RoundRect(x0, y, c.W-16, rowH-1, 4, mix(colBG, colPanel2, 0.7))
		}
		ty := u.textY(u.fM, y, rowH-1)
		ncol := colText
		if tot.Games() == 0 && i < len(Decks) {
			ncol = colFaint
		}
		if best > 0 {
			c.Circle(x0+6, y+rowH/2-1, 4, Stakes[best].Col, true, 0)
		}
		c.Text(u.fM, x0+14, ty, u.fM.Ellipsize(name, nameW-16), ncol)
		for s := 1; s <= 8; s++ {
			cx := sx + (s-1)*colW
			rec := recs[s]
			cell := colPanelDk
			txt := colFaint
			label := "-"
			if rec.W > 0 {
				cell = mix(colPanelDk, Stakes[s].Col, 0.38)
				txt = colWhite
			} else if rec.L > 0 {
				cell = mix(colPanelDk, colRed, 0.22)
				txt = colText
			}
			if rec.Games() > 0 {
				label = fmt.Sprintf("%d-%d", rec.W, rec.L)
				if u.fM.Width(label) > colW-6 {
					label = pct(rec)
				}
			}
			c.RoundRect(cx+2, y+1, colW-4, rowH-3, 3, cell)
			if selRow && s == u.deckCol {
				c.Border(cx+1, y, colW-2, rowH-1, 2, colWhite)
			}
			c.TextCenter(u.fM, cx+colW/2, ty, label, txt)
		}
		tc := colFaint
		if tot.Games() > 0 {
			tc = colGold
		}
		if selRow && u.deckCol == 9 {
			c.Border(totX+1, y, totW-2, rowH-1, 2, colWhite)
		}
		c.TextCenter(u.fM, totX+totW/2, ty, pct(tot), tc)
	}

	// detail line for the selected cell
	dy := y0 + rows*rowH + 12
	u.panel(c, 8, dy, c.W-16, c.H-34-dy-8, colPanel)
	var name string
	var rec Record
	best := 0
	if u.deckRow < len(Decks) {
		d := p.Decks[u.deckRow]
		name = d.Name
		best = d.BestWon
		if u.deckCol <= 8 {
			rec = d.ByStake[u.deckCol]
		} else {
			rec = d.Total
		}
	} else {
		name = "All decks"
		if u.deckCol <= 8 {
			rec = p.StakeTotals[u.deckCol]
		} else {
			rec = p.Total
		}
	}
	where := "every stake"
	if u.deckCol <= 8 {
		where = stakeName(u.deckCol)
	}
	h := c.H - 34 - dy - 8
	l1 := fmt.Sprintf("%s - %s", name, where)
	l2 := "No finished runs yet"
	if rec.Games() > 0 {
		l2 = fmt.Sprintf("%s, %s (%s)", plural(rec.W, "win", "wins"), plural(rec.L, "loss", "losses"), pct(rec))
	}
	if u.deckRow == len(Decks) && u.deckCol == 9 {
		l2 += fmt.Sprintf(", streak %d", p.Streak)
	} else if u.deckRow < len(Decks) && u.deckCol == 9 && best > 0 {
		l2 += ", best: " + Stakes[best].Name
	}
	c.Text(u.fS, 18, u.textY(u.fS, dy, h/2+2), u.fS.Ellipsize(l1, c.W-36), colWhite)
	c.Text(u.fS, 18, u.textY(u.fS, dy+h/2-2, h/2), u.fS.Ellipsize(l2, c.W-36), colGold)
}

// ---------- JOKERS tab ----------

func (u *UI) drawJokers(c *Canvas) {
	if u.prof == nil || len(u.jokers) == 0 {
		u.panel(c, 40, 140, c.W-80, 120, colPanel)
		c.TextCenter(u.fL, c.W/2, 160, "No jokers yet", colText)
		return
	}
	lx, lw := 8, 396
	wX, lX, rX := lx+lw-138, lx+lw-92, lx+lw-6
	hy := 52
	c.Text(u.fS, lx+8, u.textY(u.fS, hy, 26), "BY "+strings.ToUpper(sortNames[u.jokerSrt]), colDim)
	c.TextCenter(u.fS, wX, u.textY(u.fS, hy, 26), "W", colBlue)
	c.TextCenter(u.fS, lX, u.textY(u.fS, hy, 26), "L", colRed)
	c.TextRight(u.fS, rX, u.textY(u.fS, hy, 26), "WIN%", colDim)
	rowH, y0 := 26, 80
	for i := 0; i < jokerRows && u.jokerTop+i < len(u.jokers); i++ {
		j := u.jokers[u.jokerTop+i]
		y := y0 + i*rowH
		bg := colPanelDk
		if (u.jokerTop+i)%2 == 1 {
			bg = mix(colPanelDk, colPanel, 0.5)
		}
		if u.jokerTop+i == u.jokerCur {
			bg = colRed
		}
		c.RoundRect(lx, y, lw, rowH-2, 4, bg)
		ty := u.textY(u.fS, y, rowH-2)
		txt := colText
		if j.Total.Games() == 0 && u.jokerTop+i != u.jokerCur {
			txt = colDim
		}
		c.Text(u.fS, lx+8, ty, u.fS.Ellipsize(j.Name, wX-24-lx-8), txt)
		c.TextCenter(u.fS, wX, ty, strconv.Itoa(j.Total.W), colText)
		c.TextCenter(u.fS, lX, ty, strconv.Itoa(j.Total.L), colText)
		c.TextRight(u.fS, rX, ty, pct(j.Total), colGold)
	}
	// scroll bar
	if n := len(u.jokers); n > jokerRows {
		h := jokerRows * rowH
		th := max(16, h*jokerRows/n)
		ty := y0 + (h-th)*u.jokerTop/(n-jokerRows)
		c.Fill(lx+lw+2, y0, 3, h, colPanelDk)
		c.Fill(lx+lw+2, ty, 3, th, colDim)
	}

	// detail
	j := u.jokers[u.jokerCur]
	px := lx + lw + 10
	pw := c.W - 8 - px
	u.panel(c, px, 52, pw, c.H-34-52-10, colPanel)
	art, soul := u.a.Joker(j.Key)
	aw, ah := 142, 190
	ax := px + (pw-aw)/2
	if art != nil {
		c.Sprite(art, ax, 60, aw, ah, 0)
		if soul != nil {
			c.Sprite(soul, ax, 60, aw, ah, 0)
		}
	} else {
		c.RoundRect(ax, 60, aw, ah, 8, colPanel2)
	}
	y := 258
	for _, l := range u.fS.Wrap(j.Name, pw-16) {
		c.TextCenter(u.fS, px+pw/2, u.textY(u.fS, y, 24), l, colWhite)
		y += 24
	}
	y += 4
	c.TextCenter(u.fS, px+pw/2, u.textY(u.fS, y, 22), fmt.Sprintf("Picked up %dx", j.Count), colDim)
	y += 24
	c.TextCenter(u.fS, px+pw/2, u.textY(u.fS, y, 22), fmt.Sprintf("%d W - %d L", j.Total.W, j.Total.L), colText)
	y += 26
	// per-stake wins as little coloured pips
	pip := 20
	sx := px + (pw-8*pip)/2
	for s := 1; s <= 8; s++ {
		rec := j.ByStake[s]
		cx := sx + (s-1)*pip + pip/2
		if rec.W > 0 {
			c.Circle(cx, y+8, 7, Stakes[s].Col, true, 0)
		} else if rec.L > 0 {
			c.Circle(cx, y+8, 7, mix(colPanel, colRed, 0.6), false, 2)
		} else {
			c.Circle(cx, y+8, 7, colPanelDk, false, 2)
		}
	}
}
