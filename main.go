package main

import (
	"flag"
	"fmt"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

const version = "0.1.0"

type Env struct {
	SDRoot  string
	AppDir  string
	DataDir string
	SaveDir string // Balatro's save folder (settings.jkr, 1/, 2/, 3/)
}

func newEnv() *Env {
	exe, _ := os.Executable()
	appDir := filepath.Dir(exe)
	sd := os.Getenv("AU_SDROOT")
	if sd == "" {
		sd = "/mnt/SDCARD"
	}
	env := &Env{SDRoot: sd, AppDir: appDir, DataDir: filepath.Join(appDir, "data"),
		SaveDir: filepath.Join(sd, "Saves", "CurrentProfile", "saves", "Balatro")}
	if d := os.Getenv("AU_DATA"); d != "" {
		env.DataDir = d
	}
	if d := os.Getenv("AU_SAVES"); d != "" {
		env.SaveDir = d
	}
	os.MkdirAll(env.DataDir, 0o755)
	return env
}

var tzRe = regexp.MustCompile(`^UTC([+-])(\d{1,2}):?(\d{2})?$`)

// useOnionTimezone applies the zone set in Onion's Tweaks (.tmp_update/config/.tz,
// a POSIX string like "UTC+06:00" meaning six hours *behind* UTC).
func useOnionTimezone(sd string) {
	b, err := os.ReadFile(filepath.Join(sd, ".tmp_update", "config", ".tz"))
	if err != nil {
		return
	}
	m := tzRe.FindStringSubmatch(string(trimSpace(b)))
	if m == nil {
		return
	}
	h, _ := strconv.Atoi(m[2])
	mm, _ := strconv.Atoi(m[3])
	off := h*3600 + mm*60
	if m[1] == "+" {
		off = -off
	}
	time.Local = time.FixedZone(string(trimSpace(b)), off)
}

func trimSpace(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == '\r' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}

func main() {
	shot := flag.String("shot", "", "render a tab to PNG instead of running (run|decks|jokers)")
	out := flag.String("out", "shot.png", "PNG path for -shot")
	dump := flag.Bool("dump", false, "print what Ante Up reads from the saves and exit")
	flag.Parse()

	env := newEnv()
	useOnionTimezone(env.SDRoot)

	if *dump {
		u := NewUI(env, 640, 480)
		dumpState(u)
		return
	}
	if *shot != "" {
		c := NewCanvas(640, 480)
		u := NewUI(env, c.W, c.H)
		switch *shot {
		case "decks":
			u.tab = tabDecks
			u.deckRow, u.deckCol = 8, 1
		case "jokers":
			u.tab = tabJokers
		}
		u.Draw(c)
		f, err := os.Create(*out)
		if err != nil {
			log.Fatal(err)
		}
		png.Encode(f, c)
		f.Close()
		return
	}

	logf, _ := os.OpenFile(filepath.Join(env.DataDir, "anteup.log"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if logf != nil {
		log.SetOutput(logf)
	}
	log.Printf("Ante Up %s starting, saves=%s", version, env.SaveDir)

	fb, err := OpenFramebuffer()
	if err != nil {
		log.Fatalf("framebuffer: %v", err)
	}
	defer fb.Close()
	c := NewCanvas(fb.W, fb.H)
	keys := make(chan int, 32)
	if err := ReadInput("/dev/input/event0", keys); err != nil {
		log.Fatalf("input: %v", err)
	}
	u := NewUI(env, c.W, c.H)
	log.Printf("pixel font: %v, blinds art: %v, names: %d", u.pixel, u.a.Blinds != nil, len(u.a.Names))
	u.Draw(c)
	fb.Present(c)
	tick := time.NewTicker(250 * time.Millisecond) // only to clear toasts
	defer tick.Stop()
	toastShown := false
	for !u.quit {
		select {
		case k := <-keys:
			u.Key(k)
		case <-tick.C:
			on := u.toast != "" && time.Now().Before(u.toastT)
			if on == toastShown {
				continue
			}
		}
		toastShown = u.toast != "" && time.Now().Before(u.toastT)
		if u.quit {
			break
		}
		u.Draw(c)
		fb.Present(c)
	}
	log.Printf("bye")
	fb.Clear()
}

func dumpState(u *UI) {
	fmt.Printf("profile slot %s, err=%q, assets: font=%v blinds=%v chips=%v centers=%v jokers=%v names=%d pos=%d\n",
		u.profileSlot, u.profErr, u.a.Font != nil, u.a.Blinds != nil, u.a.Chips != nil, u.a.Centers != nil, u.a.Jokers != nil, len(u.a.Names), len(u.a.Pos))
	if p := u.prof; p != nil {
		fmt.Printf("career %d W %d L (%s), streak %d best %d\n", p.Total.W, p.Total.L, pct(p.Total), p.Streak, p.BestStreak)
		for _, d := range p.Decks {
			if d.Total.Games() == 0 {
				continue
			}
			fmt.Printf("  %-15s", d.Name)
			for s := 1; s <= 8; s++ {
				fmt.Printf(" %d-%d", d.ByStake[s].W, d.ByStake[s].L)
			}
			fmt.Printf("  %s\n", pct(d.Total))
		}
		for i, j := range u.jokers {
			if i >= 5 {
				break
			}
			fmt.Printf("  joker %-20s picked %d, %d W %d L\n", j.Name, j.Count, j.Total.W, j.Total.L)
		}
	}
	if r := u.run; r != nil {
		fmt.Printf("run: %s, %s, ante %d, round %d, seed %s, boss %s (%s), next pick ante %d, pool %d, rerolled %d\n",
			r.DeckName, stakeName(r.Stake), r.Ante, r.Round, r.Seed, r.BossKey, r.BossPhase, r.PickAnte, r.PoolSize, r.Rerolled)
		for _, row := range r.Rows {
			s, _ := u.bossStatus(row)
			fmt.Printf("  %-14s used=%d  %s\n", row.Info.Name, row.Used, s)
		}
		b := bossByKey(r.BossKey)
		if b != nil {
			fmt.Printf("boss text: %s\n", u.a.BossText(b, r.MostHand))
		}
	} else {
		fmt.Printf("run: none (%s)\n", u.runErr)
	}
}
