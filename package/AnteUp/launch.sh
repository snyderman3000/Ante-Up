#!/bin/sh
# Ante Up — Balatro stats and boss tracker for OnionOS
appdir=/mnt/SDCARD/App/AnteUp
cd "$appdir" || exit 1
mkdir -p data/game

# Borrow the font, art and text from your own copy of Balatro (nothing from
# the game ships with Ante Up). Done once, and again if the game file changes.
game=""
for f in /mnt/SDCARD/Roms/PORTS/Games/Balatro/Balatro.exe \
         /mnt/SDCARD/Roms/PORTS/Games/Balatro/Balatro.love \
         /mnt/SDCARD/Roms/PORTS/Balatro/Balatro.exe; do
    [ -f "$f" ] && { game="$f"; break; }
done
if [ -n "$game" ]; then
    stamp="$(ls -l "$game" | awk '{print $5}')-$(date -r "$game" +%s 2>/dev/null)"
    if [ "$(cat data/game/.stamp 2>/dev/null)" != "$stamp" ]; then
        rm -f data/game/*.png data/game/*.lua data/game/*.ttf
        /mnt/SDCARD/.tmp_update/bin/7z e -y -o"$appdir/data/game" "$game" \
            resources/fonts/m6x11plus.ttf \
            resources/textures/1x/BlindChips.png resources/textures/1x/chips.png \
            resources/textures/1x/Enhancers.png resources/textures/1x/Jokers.png \
            localization/en-us.lua game.lua > data/unpack.log 2>&1 \
            && echo "$stamp" > data/game/.stamp
        sync
    fi
fi

"$appdir/anteup" > "$appdir/data/stdout.log" 2>&1
sync
