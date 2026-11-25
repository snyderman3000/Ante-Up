package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// ---------- profile: deck × stake records, jokers, career ----------

type Record struct{ W, L int }

func (r Record) Games() int { return r.W + r.L }

// Rate is the win rate in percent, or -1 when there are no finished runs.
func (r Record) Rate() int {
	if r.Games() == 0 {
		return -1
	}
	return (r.W*100 + r.Games()/2) / r.Games()
}

func (r *Record) Add(o Record) { r.W += o.W; r.L += o.L }

type DeckStats struct {
	Key, Name string
	ByStake   [9]Record // 1..8
	Total     Record
	BestWon   int // highest stake with a win (0 = none)
}

type JokerStats struct {
	Key, Name string
	Count     int // times picked up, across all runs
	ByStake   [9]Record
	Total     Record
}

type Profile struct {
	Name        string
	Decks       []*DeckStats
	StakeTotals [9]Record
	Total       Record
	Jokers      []*JokerStats
	Streak      int
	BestStreak  int
	BestAnte    int
	BestRound   int
	BestHand    float64
	MostMoney   int
	Rounds      int
	BossStreak  int
	MaxStake    int
}

func stakeRecords(t T) [9]Record {
	var out [9]Record
	for s := 1; s <= 8; s++ {
		k := strconv.Itoa(s)
		out[s] = Record{t.Int("wins", k), t.Int("losses", k)}
	}
	return out
}

func LoadProfile(path string, names map[string]string) (*Profile, error) {
	t, err := ReadJKR(path)
	if err != nil {
		return nil, err
	}
	p := &Profile{Name: t.Str("name")}
	usage := t.Tab("deck_usage")
	for _, d := range Decks {
		ds := &DeckStats{Key: d.Key, Name: d.Name}
		ds.ByStake = stakeRecords(usage.Tab(d.Key))
		for s := 1; s <= 8; s++ {
			ds.Total.Add(ds.ByStake[s])
			p.StakeTotals[s].Add(ds.ByStake[s])
			if ds.ByStake[s].W > 0 {
				ds.BestWon = s
			}
		}
		p.Total.Add(ds.Total)
		p.Decks = append(p.Decks, ds)
	}
	for key := range t.Tab("joker_usage") {
		u := t.Tab("joker_usage", key)
		js := &JokerStats{Key: key, Name: names[key], Count: u.Int("count")}
		if js.Name == "" {
			js.Name = prettyKey(key)
		}
		js.ByStake = stakeRecords(u)
		for s := 1; s <= 8; s++ {
			js.Total.Add(js.ByStake[s])
		}
		p.Jokers = append(p.Jokers, js)
	}
	sort.Slice(p.Jokers, func(i, j int) bool { return p.Jokers[i].Name < p.Jokers[j].Name })
	hs := t.Tab("high_scores")
	p.Streak = hs.Int("current_streak", "amt")
	p.BestStreak = hs.Int("win_streak", "amt")
	p.BestAnte = hs.Int("furthest_ante", "amt")
	p.BestRound = hs.Int("furthest_round", "amt")
	p.BestHand = hs.Num("hand", "amt")
	p.MostMoney = hs.Int("most_money", "amt")
	p.BossStreak = hs.Int("boss_streak", "amt")
	p.Rounds = t.Int("career_stats", "c_rounds")
	p.MaxStake = t.Int("stake")
	return p, nil
}

// prettyKey turns "j_ride_the_bus" into "Ride The Bus".
func prettyKey(k string) string {
	if len(k) > 2 && k[1] == '_' {
		k = k[2:]
	}
	b := []byte(k)
	up := true
	for i, c := range b {
		if c == '_' {
			b[i] = ' '
			up = true
			continue
		}
		if up && c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
		up = false
	}
	return string(b)
}

// ---------- the run in progress ----------

type BossState int

const (
	BossLater   BossState = iota // can't show up yet (ante too low)
	BossPool                     // could be the next boss
	BossWaiting                  // unseen, eligible, but others come first (only after a full cycle)
	BossCurrent                  // this ante's boss
	BossBeaten                   // beaten earlier this run
	BossSeen                     // came up this run, but may have been rerolled away
	BossBanned                   // banned by a challenge
)

type BossRow struct {
	Info  *BossInfo
	State BossState
	Used  int
	From  int // for BossLater: the ante it unlocks
	Again bool
}

type Run struct {
	Path      string
	Saved     time.Time
	Seed      string
	DeckKey   string
	DeckName  string
	Stake     int
	Ante      int
	Round     int
	Dollars   int
	WinAnte   int
	Won       bool
	BossKey   string
	BossPhase string // Select / Upcoming / Current / Defeated
	PickAnte  int    // the ante the next boss will be picked for
	Rerolled  int    // bosses that came up but were rerolled away (estimate)
	Rows      []*BossRow
	PoolSize  int
	MostHand  string
}

func LoadRun(path string, hist *History) (*Run, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	t, err := ReadJKR(path)
	if err != nil {
		return nil, err
	}
	g := t.Tab("GAME")
	rr := g.Tab("round_resets")
	r := &Run{
		Path:      path,
		Saved:     st.ModTime(),
		Seed:      g.Str("pseudorandom", "seed"),
		Stake:     g.Int("stake"),
		Ante:      rr.Int("ante"),
		Round:     g.Int("round"),
		Dollars:   g.Int("dollars"),
		WinAnte:   g.Int("win_ante"),
		Won:       g.Bool("won"),
		BossKey:   rr.Str("blind_choices", "Boss"),
		BossPhase: rr.Str("blind_states", "Boss"),
		MostHand:  g.Str("current_round", "most_played_poker_hand"),
	}
	if r.WinAnte <= 0 {
		r.WinAnte = 8
	}
	if r.Stake <= 0 {
		r.Stake = 1
	}
	back := g.Tab("selected_back_key")
	r.DeckKey = back.Str("key")
	r.DeckName = back.Str("name")
	if r.DeckKey == "" {
		r.DeckKey = deckByName(r.DeckName)
	}
	if r.DeckName == "" {
		r.DeckName = deckName(r.DeckKey)
	}
	defeated := r.BossPhase == "Defeated"

	used := g.Tab("bosses_used")
	banned := g.Tab("banned_keys")
	hist.Note(r)
	confirmed := hist.Beaten(r)

	total := 0
	for _, b := range Bosses {
		total += used.Int(b.Key)
	}
	expected := r.Ante // bosses beaten (ante-1) plus this ante's boss
	if defeated {
		expected = r.Ante - 1 // ante already went up; next boss not picked yet
	}
	if r.Rerolled = total - expected; r.Rerolled < 0 {
		r.Rerolled = 0
	}

	r.PickAnte = r.Ante + 1
	if defeated {
		r.PickAnte = r.Ante
	}
	eligible := func(b *BossInfo, a int) bool {
		if _, no := banned[b.Key]; no {
			return false
		}
		if b.Showdown {
			return a%r.WinAnte == 0 && a >= 2
		}
		return b.Min <= max(1, a) && (max(1, a)%r.WinAnte != 0 || a < 2)
	}
	minUse := 1 << 30
	for i := range Bosses {
		if eligible(&Bosses[i], r.PickAnte) {
			minUse = min(minUse, used.Int(Bosses[i].Key))
		}
	}

	for i := range Bosses {
		b := &Bosses[i]
		row := &BossRow{Info: b, Used: used.Int(b.Key)}
		inPool := eligible(b, r.PickAnte) && row.Used == minUse
		switch {
		case b.Key == r.BossKey && !defeated:
			row.State = BossCurrent
		case row.Used > 0:
			row.State = BossBeaten
			if r.Rerolled > 0 && !confirmed[b.Key] && !(b.Key == r.BossKey && defeated) {
				row.State = BossSeen
			}
			row.Again = inPool
		default:
			if _, no := banned[b.Key]; no {
				row.State = BossBanned
			} else if inPool {
				row.State = BossPool
			} else if eligible(b, r.PickAnte) {
				row.State = BossWaiting
			} else {
				row.State = BossLater
				row.From = nextAnteFor(b, r.PickAnte, r.WinAnte)
			}
		}
		if inPool {
			r.PoolSize++
		}
		r.Rows = append(r.Rows, row)
	}
	return r, nil
}

// nextAnteFor is the first ante ≥ from where boss b can be picked.
func nextAnteFor(b *BossInfo, from, win int) int {
	if b.Showdown {
		a := ((from + win - 1) / win) * win
		if a < 2 {
			a = win
		}
		return a
	}
	a := max(from, b.Min)
	if a%win == 0 && a >= 2 {
		a++
	}
	return a
}

// ---------- history: which boss each ante had, as seen by Ante Up ----------
// The save only counts how many times each boss came up, so a boss rerolled
// away with Director's Cut looks the same as one that was beaten. Each time
// Ante Up opens it notes the current ante's boss; the last boss noted for a
// past ante is the one that was actually fought.

type History struct {
	path string
	Runs map[string]map[string]string `json:"runs"` // seed → ante → boss key
	Last map[string]int64              `json:"last"` // seed → unix time last seen
}

func LoadHistory(dir string) *History {
	h := &History{path: filepath.Join(dir, "history.json"), Runs: map[string]map[string]string{}, Last: map[string]int64{}}
	if b, err := os.ReadFile(h.path); err == nil {
		json.Unmarshal(b, h)
		if h.Runs == nil {
			h.Runs = map[string]map[string]string{}
		}
		if h.Last == nil {
			h.Last = map[string]int64{}
		}
	}
	return h
}

func (h *History) key(r *Run) string { return r.Seed + "|" + r.DeckKey + "|" + strconv.Itoa(r.Stake) }

func (h *History) Note(r *Run) {
	if h == nil || r.Seed == "" || r.BossKey == "" {
		return
	}
	k := h.key(r)
	m := h.Runs[k]
	if m == nil {
		m = map[string]string{}
		h.Runs[k] = m
	}
	ante := r.Ante
	if r.BossPhase == "Defeated" {
		ante-- // the boss on the board is the one just beaten, from the previous ante
	}
	m[strconv.Itoa(ante)] = r.BossKey
	h.Last[k] = time.Now().Unix()
	// keep the file small: forget runs not seen for 60 days
	for s, t := range h.Last {
		if time.Since(time.Unix(t, 0)) > 60*24*time.Hour {
			delete(h.Last, s)
			delete(h.Runs, s)
		}
	}
	h.save()
}

// Beaten returns the bosses known to have been fought in earlier antes.
func (h *History) Beaten(r *Run) map[string]bool {
	out := map[string]bool{}
	if h == nil {
		return out
	}
	cur := r.Ante
	for a, b := range h.Runs[h.key(r)] {
		n, _ := strconv.Atoi(a)
		if n < cur {
			out[b] = true
		}
	}
	return out
}

func (h *History) save() {
	if h.path == "" {
		return
	}
	b, _ := json.MarshalIndent(h, "", " ")
	tmp := h.path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, h.path)
	}
}
