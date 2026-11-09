#!/bin/sh
# witems → punisher 단독판 이식 도우미.
# witems 커밋 범위에서 Punisher 관련 변경만 뽑아, 단독판 경로·import 로 바꾼 패치 3개를 만든다.
#   사용법: scripts/witems-sync.sh <witems 저장소 경로> <기준 커밋> <대상 커밋> [출력 폴더]
#   예:     scripts/witems-sync.sh ~/repo/witems-workspace/witems-claude 98308e6 HEAD
# 결과(출력 폴더, 기본 .sync/<기준>..<대상>):
#   files.txt        변경 파일 목록(name-status) — 이식 보고서에 그대로 붙인다
#   commits.txt      범위 안의 관련 커밋
#   server.patch     apps/punisher-server (store·migrations 제외) → internal/, cmd/ 등
#   store.diff       apps/punisher-server/internal/store, migrations — PostgreSQL 이라 손으로 SQLite 포팅
#   web.patch        apps/web/src/**/punisher/** → web/src/**
#   kit.patch        packages/ui/src, packages/config(styles·tailwind) → web/src/shared/ui/kit, web/src/styles, web/tailwind.preset.js
#   web-other.txt    punisher 밖 apps/web 변경 중 학습 관련(app/ 라우트 등) — 단독판 pages/ 에 손으로 반영
# 적용: git apply --recount --reject <patch> (web.patch 는 --directory=web) 후 *.rej 를 손으로 정리.
set -eu

if [ $# -lt 3 ]; then
  sed -n '2,15p' "$0"; exit 2
fi
witems=$1; from=$2; to=$3
here=$(cd "$(dirname "$0")/.." && pwd)
out=${4:-"$here/.sync/$from..$to"}
mkdir -p "$out"

# 단독판에서 바뀐 모듈 경로·import. 새 규칙이 생기면 여기에만 추가한다.
go_rewrite() {
  sed -e 's#github.com/wooo-jin/witems/apps/punisher-server#github.com/wooo-jin/punisher#g' \
      -e 's#github.com/wooo-jin/witems/packages/go-shared/llmjson#github.com/wooo-jin/punisher/internal/llmjson#g' \
      -e 's#github.com/wooo-jin/witems/packages/go-shared/llm"#github.com/wooo-jin/punisher/internal/llm"#g' \
      -e 's#\*store\.PostgresStore#store.Store#g'
}
web_rewrite() {
  sed -e 's#from "@witems/ui"#from "@/src/shared/ui/kit"#g' \
      -e "s#from '@witems/config/theme'#from '@/src/shared/lib/theme'#g" \
      -e 's#import Link from "next/link";#import { Link } from "@/src/shared/lib/router";#g' \
      -e 's#from "next/navigation"#from "@/src/shared/lib/router"#g' \
      -e 's#/workspace/punisher/#/#g' -e 's#"/workspace/punisher"#"/"#g' \
      -e 's#/api/punisher/v1#/v1#g' \
      -e 's#/settings/learning/prompts#/settings/prompts#g' -e 's#"/settings/learning"#"/settings"#g' \
      -e '/^[+ ]"use client";$/d'
}

g() { git -C "$witems" "$@"; }

g log --oneline "$from..$to" -- apps/punisher-server 'apps/web/src/**/punisher/**' packages/ui packages/config > "$out/commits.txt"
{
  g diff --name-status "$from" "$to" -- apps/punisher-server
  g diff --name-status "$from" "$to" -- 'apps/web/src/**/punisher/**'
  g diff --name-status "$from" "$to" -- packages/ui/src packages/config/styles packages/config/tailwind
} > "$out/files.txt"

g diff "$from" "$to" --relative=apps/punisher-server -- apps/punisher-server \
  ':!apps/punisher-server/internal/store' ':!apps/punisher-server/migrations' ':!apps/punisher-server/Dockerfile' \
  ':!apps/punisher-server/cmd' ':!apps/punisher-server/internal/config' ':!apps/punisher-server/internal/middleware' \
  | go_rewrite > "$out/server.patch"
g diff "$from" "$to" -- apps/punisher-server/internal/store apps/punisher-server/migrations > "$out/store.diff"
g diff "$from" "$to" --relative=apps/web -- 'apps/web/src/**/punisher/**' | web_rewrite > "$out/web.patch"
g diff "$from" "$to" -- packages/ui/src packages/config/styles/design-tokens.css packages/config/styles/animations.css packages/config/tailwind/preset.js \
  | sed -e 's#\([ab]\)/packages/ui/src/#\1/web/src/shared/ui/kit/#g' \
        -e 's#\([ab]\)/packages/config/styles/#\1/web/src/styles/#g' \
        -e 's#\([ab]\)/packages/config/tailwind/preset.js#\1/web/tailwind.preset.js#g' \
  | web_rewrite > "$out/kit.patch"
g diff --name-status "$from" "$to" -- apps/web | grep -iE 'learn|punisher|prompt' | grep -v '/punisher/' > "$out/web-other.txt" || true

echo "출력: $out"
wc -l "$out"/*.txt "$out"/*.patch "$out"/*.diff | sed 's#'"$out"'/##'
echo
echo "다음: cd $here && git apply --recount --check $out/server.patch && git apply --recount --check --directory=web $out/web.patch"
