#!/bin/sh
# 브랜드 PNG 생성 — 원본은 assets/*.svg (forgen 패밀리: 불꽃 + 오렌지 #f97316). 필요: rsvg-convert(librsvg).
# 결과: assets/{icon,favicon,banner,banner@2x}.png, web/public/{favicon.svg,apple-touch-icon.png,icon-192.png,icon-512.png}
set -eu
cd "$(dirname "$0")/.."
a=assets
rsvg-convert -w 512 -h 512 $a/logo.svg -o $a/icon.png
rsvg-convert -w 32 -h 32 $a/favicon.svg -o $a/favicon.png
rsvg-convert -w 800 -h 200 $a/banner.svg -o $a/banner.png
rsvg-convert -w 1600 -h 400 $a/banner.svg -o $a/banner@2x.png
cp $a/favicon.svg web/public/favicon.svg
rsvg-convert -w 180 -h 180 $a/app-icon.svg -o web/public/apple-touch-icon.png
rsvg-convert -w 192 -h 192 $a/app-icon.svg -o web/public/icon-192.png
rsvg-convert -w 512 -h 512 $a/app-icon.svg -o web/public/icon-512.png
ls -l $a/*.png web/public/*.png
