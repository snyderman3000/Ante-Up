ANTE UP for OnionOS (Miyoo Mini / Mini+)
========================================
Balatro stats the game doesn't show you: win rates for every deck and stake,
and a boss-blind board for the run you're playing.

Created by Claude (Anthropic) for Keith Snyder.

TABS (L1 / R1 to switch)
  RUN     Every boss blind, marked for the current run:
            check    beaten this run
            check ?  came up, but you may have rerolled it away
            star     this ante's boss
            ?        could be the next boss
            A6       can't show up until ante 6
          Move with the D-pad to read a boss's effect.
  DECKS   Wins-losses for each deck on each stake, and win % overall.
          The dot by a deck's name is the colour of its best stake win.
  JOKERS  How often each joker was in your hand when a run ended in a win
          or a loss. Y changes the sort (wins, win rate, picked, A-Z).

  SELECT reloads from the save, B quits.

HOW IT WORKS
  Ante Up only reads Balatro's save files
  (Saves/CurrentProfile/saves/Balatro) - it never changes them, and you don't
  need to change the Balatro port. Quit Balatro (or open the GameSwitcher and
  go home), open Ante Up, check the board, then jump back in.

  Losses are runs that ended in game over. Runs you abandon by starting a new
  one aren't counted by the game, so they don't show up here either.

  "Could be next" follows the game's own rule: the next boss is picked at
  random from the bosses allowed at that ante that have come up the fewest
  times this run.

  Rerolls: the save only counts how often each boss came up, so a boss you
  rerolled with Director's Cut looks the same as one you beat. Ante Up notes
  the boss each time you open it, which sorts most of that out.

GAME FILES
  The first time it runs, Ante Up copies the font, the boss chip, deck and
  joker art and the English text out of your own Balatro.exe into
  data/game. None of Balatro's files are included with Ante Up. Without them
  it still works, with plain colours and its own font.

LICENSES
  Ante Up: MIT. VT323 font: SIL OFL 1.1. golang.org/x/image: BSD-3-Clause.
  See the licenses folder.
