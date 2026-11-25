package main

// What Ante Up knows about Balatro's decks, stakes and boss blinds.
// Names, minimum antes and colours match the game (v1.0.1); the art, the
// font and the effect text are read at run time from the player's own copy
// of the game (see assets.go), so none of the game's files ship with the app.

import (
	"image/color"
	"strconv"
)

type DeckInfo struct {
	Key, Name string
}

var Decks = []DeckInfo{
	{"b_red", "Red Deck"}, {"b_blue", "Blue Deck"}, {"b_yellow", "Yellow Deck"},
	{"b_green", "Green Deck"}, {"b_black", "Black Deck"}, {"b_magic", "Magic Deck"},
	{"b_nebula", "Nebula Deck"}, {"b_ghost", "Ghost Deck"}, {"b_abandoned", "Abandoned Deck"},
	{"b_checkered", "Checkered Deck"}, {"b_zodiac", "Zodiac Deck"}, {"b_painted", "Painted Deck"},
	{"b_anaglyph", "Anaglyph Deck"}, {"b_plasma", "Plasma Deck"}, {"b_erratic", "Erratic Deck"},
}

func deckByName(name string) string {
	for _, d := range Decks {
		if d.Name == name {
			return d.Key
		}
	}
	return ""
}

func deckName(key string) string {
	for _, d := range Decks {
		if d.Key == key {
			return d.Name
		}
	}
	return key
}

type StakeInfo struct {
	Name  string
	Col   color.RGBA
	ChipX int // position in chips.png
	ChipY int
}

// Index 1..8 = stake level.
var Stakes = []StakeInfo{
	{},
	{"White", rgb(0xf0, 0xf0, 0xf0), 0, 0},
	{"Red", rgb(0xfe, 0x5f, 0x55), 1, 0},
	{"Green", rgb(0x4b, 0xc2, 0x92), 2, 0},
	{"Black", rgb(0x4f, 0x63, 0x67), 4, 0},
	{"Blue", rgb(0x00, 0x9d, 0xff), 3, 0},
	{"Purple", rgb(0x8a, 0x71, 0xe1), 0, 1},
	{"Orange", rgb(0xfd, 0xa2, 0x00), 1, 1},
	{"Gold", rgb(0xda, 0xb7, 0x72), 2, 1},
}

func stakeName(n int) string {
	if n >= 1 && n < len(Stakes) {
		return Stakes[n].Name + " Stake"
	}
	return "Stake " + strconv.Itoa(n)
}

type BossInfo struct {
	Key      string
	Name     string
	Min      int  // earliest ante it can appear
	Showdown bool // final boss (ante 8, 16, ...)
	Col      color.RGBA
	Row      int    // row in BlindChips.png
	Effect   string // short fallback description, used when the game's text can't be read
}

func hex(v uint32) color.RGBA { return rgb(uint8(v>>16), uint8(v>>8), uint8(v)) }

// Regular bosses in the order the game's collection shows them, then finals.
var Bosses = []BossInfo{
	{"bl_hook", "The Hook", 1, false, hex(0xa84024), 7, "Discards 2 random cards from your hand each time you play"},
	{"bl_ox", "The Ox", 6, false, hex(0xb95b08), 2, "Playing your most played hand sets money to $0"},
	{"bl_house", "The House", 2, false, hex(0x5186a8), 3, "First hand is drawn face down"},
	{"bl_wall", "The Wall", 2, false, hex(0x8a59a5), 9, "Extra large blind (4x)"},
	{"bl_wheel", "The Wheel", 2, false, hex(0x50bf7c), 10, "1 in 7 cards get drawn face down"},
	{"bl_arm", "The Arm", 2, false, hex(0x6865f3), 11, "Lowers the level of the hand you play"},
	{"bl_club", "The Club", 1, false, hex(0xb9cb92), 4, "Clubs are debuffed"},
	{"bl_fish", "The Fish", 2, false, hex(0x3e85bd), 5, "Cards drawn after each hand are face down"},
	{"bl_psychic", "The Psychic", 1, false, hex(0xefc03c), 12, "Every hand must use 5 cards"},
	{"bl_goad", "The Goad", 1, false, hex(0xb95c96), 13, "Spades are debuffed"},
	{"bl_water", "The Water", 2, false, hex(0xc6e0eb), 14, "Start with no discards"},
	{"bl_window", "The Window", 1, false, hex(0xa9a295), 6, "Diamonds are debuffed"},
	{"bl_manacle", "The Manacle", 1, false, hex(0x575757), 8, "Hand size -1"},
	{"bl_eye", "The Eye", 3, false, hex(0x4b71e4), 17, "Can't repeat a hand type this round"},
	{"bl_mouth", "The Mouth", 2, false, hex(0xae718e), 18, "Only one hand type allowed this round"},
	{"bl_plant", "The Plant", 4, false, hex(0x709284), 19, "Face cards are debuffed"},
	{"bl_serpent", "The Serpent", 5, false, hex(0x439a4f), 15, "Always draws just 3 cards after a play or discard"},
	{"bl_pillar", "The Pillar", 1, false, hex(0x7e6752), 16, "Cards already played this ante are debuffed"},
	{"bl_needle", "The Needle", 2, false, hex(0x5c6e31), 20, "Only 1 hand"},
	{"bl_head", "The Head", 1, false, hex(0xac9db4), 21, "Hearts are debuffed"},
	{"bl_tooth", "The Tooth", 3, false, hex(0xb52d2d), 22, "Lose $1 for each card played"},
	{"bl_flint", "The Flint", 2, false, hex(0xe56a2f), 24, "Base chips and mult are halved"},
	{"bl_mark", "The Mark", 2, false, hex(0x6a3847), 23, "Face cards are drawn face down"},
	{"bl_final_acorn", "Amber Acorn", 8, true, hex(0xfda200), 27, "Flips and shuffles your jokers"},
	{"bl_final_leaf", "Verdant Leaf", 8, true, hex(0x56a786), 28, "Every card is debuffed until you sell a joker"},
	{"bl_final_vessel", "Violet Vessel", 8, true, hex(0x8a71e1), 29, "Huge blind (6x)"},
	{"bl_final_heart", "Crimson Heart", 8, true, hex(0xac3232), 25, "A random joker is disabled each hand"},
	{"bl_final_bell", "Cerulean Bell", 8, true, hex(0x009cfd), 26, "One card is always forced into your selection"},
}

func bossByKey(k string) *BossInfo {
	for i := range Bosses {
		if Bosses[i].Key == k {
			return &Bosses[i]
		}
	}
	return nil
}
