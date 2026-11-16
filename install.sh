#!/bin/sh
# Punisher 설치 (forgen 패밀리): 내려받은 punisher 실행 파일을 ~/bin 으로 복사합니다 (Homebrew 불필요).
#   사용법: sh install.sh [punisher 파일 경로]   (기본: 이 스크립트 옆의 punisher 또는 dist/punisher)
set -eu

here=$(cd "$(dirname "$0")" && pwd)
src=${1:-}
if [ -z "$src" ]; then
  for c in "$here/punisher" "$here/dist/punisher"; do
    if [ -f "$c" ]; then src=$c; break; fi
  done
fi
if [ -z "$src" ] || [ ! -f "$src" ]; then
  echo "punisher 실행 파일을 찾지 못했습니다. 경로를 지정해 주세요: sh install.sh ~/Downloads/punisher" >&2
  exit 1
fi

dest_dir="$HOME/bin"
mkdir -p "$dest_dir"
cp "$src" "$dest_dir/punisher"
chmod +x "$dest_dir/punisher"
# macOS: 인터넷에서 받은 파일의 격리 속성(Gatekeeper '확인되지 않은 개발자' 경고)을 제거합니다.
if [ "$(uname -s)" = "Darwin" ]; then
  xattr -d com.apple.quarantine "$dest_dir/punisher" 2>/dev/null || true
fi

echo "설치 완료: $dest_dir/punisher ($("$dest_dir/punisher" --version | head -1))"
echo "Punisher · part of the forgen family (https://github.com/wooo-jin/forgen) · © 2026 jang-ujin"
case ":$PATH:" in
  *":$dest_dir:"*) echo "이제 터미널에서 punisher 를 입력하면 실행됩니다." ;;
  *)
    case "${SHELL:-}" in
      */bash) shell_rc="$HOME/.bashrc" ;;
      *) shell_rc="$HOME/.zshrc" ;;
    esac
    if ! grep -qs 'export PATH="$HOME/bin:$PATH"' "$shell_rc"; then
      printf '\n# Punisher\nexport PATH="$HOME/bin:$PATH"\n' >> "$shell_rc"
      echo "~/bin 을 PATH 에 추가했습니다 ($shell_rc). 새 터미널 창을 열고 punisher 를 입력하세요."
    else
      echo "새 터미널 창을 열고 punisher 를 입력하세요."
    fi
    ;;
esac

if ! command -v claude >/dev/null 2>&1 && [ ! -x "$HOME/.local/bin/claude" ]; then
  echo
  echo "[주의] Claude Code(claude)가 보이지 않습니다. 먼저 설치하고 한 번 로그인하세요:"
  echo "  curl -fsSL https://claude.ai/install.sh | bash"
  echo "  claude        # 실행 후 안내에 따라 로그인"
fi
