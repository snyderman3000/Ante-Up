# Ante Up

Balatro stats for **OnionOS** on the Miyoo Mini / Mini+, covering what the game doesn't show you:

- **RUN**: a board of all 28 boss blinds for the run you're playing. It shows which ones you've beaten, which one is up this ante, which ones could come up next, and which can't appear until a later ante.
- **DECKS**: wins and losses for every deck on every stake, with win rates.
- **JOKERS**: how often each joker was in your hand when a run ended in a win or a loss.

Created by Claude (Anthropic) for [snyderman3000](https://github.com/snyderman3000).

## Install

Unzip the release zip onto the root of your SD card (it adds `App/AnteUp`), then open **Ante Up** from Apps.

It works with the [Balatro for Miyoo Mini](https://github.com/Producdevity) port. It reads the saves in `Saves/CurrentProfile/saves/Balatro` and never writes to them.

## Controls

| Button | Action |
|---|---|
| L1 / R1 | Switch tabs |
| D-pad | Move (pick a boss to read its effect, a deck × stake cell, or a joker) |
| Y | Jokers: change the sort (wins, win rate, picked, A-Z) |
| SELECT | Reload from the save |
| B | Quit |

## How it works

Balatro's `.jkr` save files are compressed Lua tables. `profile.jkr` already records wins and losses per deck per stake (and per joker). `save.jkr` holds the current run, including how many times each boss has come up.

"Could be next" uses the game's own rule: the next boss is picked at random from the bosses allowed at that ante that have come up the fewest times this run.

The save can't tell a boss you rerolled with Director's Cut apart from one you beat. Ante Up notes the boss each time you open it, so those get sorted out as you play. Anything still unclear is marked "✓?".

## Game files

None of Balatro's files are included. On first launch, `launch.sh` copies the font, the boss chip, deck and joker art, and the English text out of your own `Balatro.exe` into `data/game`. Without them Ante Up still works, with plain colours and the bundled VT323 font.

## Building

```sh
./build.sh   # Go 1.24; writes dist/AnteUp-v<version>-OnionOS.zip
```

## License

MIT. Third-party licenses are in [`package/AnteUp/licenses`](package/AnteUp/licenses).
