#!/bin/sh
# Builds dist/AnteUp-v<version>-OnionOS.zip (unzip onto the SD card root).
set -e
cd "$(dirname "$0")"
VERSION=$(grep -o 'const version = "[^"]*"' main.go | cut -d'"' -f2)
# go.sum isn't kept in the repo; tidy writes it (skipped inside a go.work workspace)
[ -n "$(go env GOWORK)" ] || go mod tidy
go test ./...
go run ./tools/icongen > package/AnteUp/icon.png
CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -trimpath -ldflags="-s -w" -o package/AnteUp/anteup .
rm -rf dist && mkdir -p dist/stage/App
cp -r package/AnteUp dist/stage/App/
rm -rf dist/stage/App/AnteUp/data
(cd dist/stage && zip -qr "../AnteUp-v$VERSION-OnionOS.zip" App)
rm -rf dist/stage
echo "Built dist/AnteUp-v$VERSION-OnionOS.zip"
