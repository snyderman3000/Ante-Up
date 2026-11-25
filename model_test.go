package main

import (
	"bytes"
	"compress/flate"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jkr(t *testing.T, path, text string) {
	var b bytes.Buffer
	w, _ := flate.NewWriter(&b, flate.BestCompression)
	w.Write([]byte(text))
	w.Close()
	os.MkdirAll(filepath.Dir(path), 0o755)
	if err := os.WriteFile(path, b.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseLua(t *testing.T) {
	src := `return {["a"]={["b"]=3,[1]=1.5e+52,["s"]="say \"hi\"\\",},["t"]=true,["n"]=-2,x=7,{"pos1","pos2"},}`
	v, err := ParseLua([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if v.Int("a", "b") != 3 || v.Num("a", "1") != 1.5e52 || v.Str("a", "s") != `say "hi"\` {
		t.Fatalf("bad: %#v", v)
	}
	if !v.Bool("t") || v.Int("n") != -2 || v.Int("x") != 7 || v.Str("1", "2") != "pos2" {
		t.Fatalf("bad: %#v", v)
	}
}

func usedTable(m map[string]int) string {
	var b strings.Builder
	b.WriteString("{")
	for _, x := range Bosses {
		fmt.Fprintf(&b, `["%s"]=%d,`, x.Key, m[x.Key])
	}
	b.WriteString("}")
	return b.String()
}

func saveText(ante int, boss, phase string, used map[string]int) string {
	return fmt.Sprintf(`return {["GAME"]={["pseudorandom"]={["seed"]="TEST1234",},["stake"]=3,["dollars"]=12,["round"]=%d,["win_ante"]=8,
	["selected_back_key"]={["name"]="Red Deck",},["bosses_used"]=%s,
	["round_resets"]={["ante"]=%d,["blind_choices"]={["Boss"]="%s",},["blind_states"]={["Boss"]="%s",},},},}`,
		ante*3, usedTable(used), ante, boss, phase)
}

func row(r *Run, key string) *BossRow {
	for _, x := range r.Rows {
		if x.Info.Key == key {
			return x
		}
	}
	return nil
}

func TestRunStates(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "save.jkr")
	used := map[string]int{"bl_hook": 1, "bl_wall": 1, "bl_eye": 1}
	jkr(t, p, saveText(3, "bl_eye", "Upcoming", used))
	h := LoadHistory(dir)
	r, err := LoadRun(p, h)
	if err != nil {
		t.Fatal(err)
	}
	if r.DeckKey != "b_red" || r.Stake != 3 || r.Ante != 3 || r.PickAnte != 4 || r.Rerolled != 0 {
		t.Fatalf("run %+v", r)
	}
	if row(r, "bl_eye").State != BossCurrent || row(r, "bl_hook").State != BossBeaten {
		t.Fatal("current/beaten wrong")
	}
	if s := row(r, "bl_plant"); s.State != BossPool {
		t.Fatalf("plant (min 4) should be possible at ante 4, got %v", s.State)
	}
	if s := row(r, "bl_serpent"); s.State != BossLater || s.From != 5 {
		t.Fatalf("serpent: %+v", s)
	}
	if s := row(r, "bl_final_bell"); s.State != BossLater || s.From != 8 {
		t.Fatalf("bell: %+v", s)
	}

	// Beat The Eye: ante goes to 4, Boss state Defeated, choice still bl_eye.
	jkr(t, p, saveText(4, "bl_eye", "Defeated", used))
	r, _ = LoadRun(p, h)
	if row(r, "bl_eye").State != BossBeaten || r.PickAnte != 4 || r.Rerolled != 0 {
		t.Fatalf("after defeat: %+v / %+v", row(r, "bl_eye"), r)
	}

	// Ante 8: only finals are in the pool.
	jkr(t, p, saveText(7, "bl_ox", "Upcoming", map[string]int{"bl_ox": 1, "bl_hook": 1, "bl_wall": 1, "bl_eye": 1, "bl_club": 1, "bl_fish": 1, "bl_mouth": 1}))
	r, _ = LoadRun(p, h)
	if r.PickAnte != 8 || r.PoolSize != 5 || row(r, "bl_final_heart").State != BossPool || row(r, "bl_plant").State != BossLater || row(r, "bl_plant").From != 9 {
		t.Fatalf("ante 8 pool: %d %v %v", r.PoolSize, row(r, "bl_final_heart").State, row(r, "bl_plant").State)
	}

	// A reroll at ante 2: hook (ante 1), then mark came up and was rerolled into wall.
	h2 := LoadHistory(t.TempDir())
	jkr(t, p, saveText(2, "bl_mark", "Upcoming", map[string]int{"bl_hook": 1, "bl_mark": 1}))
	LoadRun(p, h2) // Ante Up saw The Mark as ante 2's boss
	jkr(t, p, saveText(2, "bl_wall", "Upcoming", map[string]int{"bl_hook": 1, "bl_mark": 1, "bl_wall": 1}))
	r, _ = LoadRun(p, h2)
	if r.Rerolled != 1 || row(r, "bl_mark").State != BossSeen || row(r, "bl_wall").State != BossCurrent {
		t.Fatalf("reroll: %d %v %v", r.Rerolled, row(r, "bl_mark").State, row(r, "bl_wall").State)
	}
}

func TestProfile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "profile.jkr")
	jkr(t, p, `return {["deck_usage"]={["b_red"]={["wins"]={[1]=2,[3]=1,},["losses"]={[3]=1,},},},
	["joker_usage"]={["j_blueprint"]={["wins"]={[1]=3,},["losses"]={[1]=1,},["count"]=9,},},
	["high_scores"]={["win_streak"]={["amt"]=4,},},}`)
	pr, err := LoadProfile(p, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	red := pr.Decks[0]
	if red.Total != (Record{3, 1}) || red.BestWon != 3 || red.Total.Rate() != 75 || pr.Total != (Record{3, 1}) {
		t.Fatalf("red: %+v", red)
	}
	if len(pr.Jokers) != 1 || pr.Jokers[0].Name != "Blueprint" || pr.Jokers[0].Total.W != 3 || pr.BestStreak != 4 {
		t.Fatalf("jokers: %+v", pr.Jokers[0])
	}
}

func TestLocalization(t *testing.T) {
	a := &Assets{Names: map[string]string{}, Text: map[string]string{}, Pos: map[string]Pos{}, SoulPos: map[string]Pos{}}
	parseLocalization(`        Blind={
            bl_ox={
                name="The Ox",
                text={
                    "Playing a {C:attention}#1#{}",
                    "sets money to {C:money}$0",
                },
            },`, a)
	if a.Names["bl_ox"] != "The Ox" || a.BossText(bossByKey("bl_ox"), "Pair") != "Playing a Pair sets money to $0" {
		t.Fatalf("%q", a.Text["bl_ox"])
	}
	parsePositions(`        j_perkeo=           {order = 150, name = "Perkeo", pos = {x=7,y=8}, soul_pos = {x=7, y=9}, set = "Joker"},
        b_red=              {name = "Red Deck",         stake = 1, pos =   {x=0,y=0}, set = "Back"},`, a)
	if a.Pos["j_perkeo"] != (Pos{7, 8}) || a.SoulPos["j_perkeo"] != (Pos{7, 9}) || a.Pos["b_red"] != (Pos{0, 0}) {
		t.Fatalf("pos %+v %+v", a.Pos, a.SoulPos)
	}
}
