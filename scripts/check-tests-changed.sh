#!/usr/bin/env bash
# 実装を変えたらテストも変える、を強制する。
#   check-tests-changed.sh staged            … ステージ済みの差分を警告のみで検査（pre-commit 用）
#   check-tests-changed.sh range <base-ref>  … <base-ref>...HEAD を検査し、違反なら失敗（pre-push / CI 用）
# 例外: PR 本文（環境変数 PR_BODY）に `[no-test-needed]: <理由>` を書く。純粋なドキュメント・設定だけの変更は対象外。
set -euo pipefail
cd "$(dirname "$0")/.."
mode="${1:-range}"
case "$mode" in
  staged) files=$(git diff --cached --name-only --diff-filter=ACMR) ;;
  range)  base="${2:-origin/main}"; files=$(git diff --name-only --diff-filter=ACMR "$base"...HEAD) ;;
  *) echo "usage: $0 staged | range <base>"; exit 2 ;;
esac

is_go_src()  { [[ "$1" == internal/*.go || "$1" == internal/**/*.go || "$1" == app.go || "$1" == main.go ]] && [[ "$1" != *_test.go ]]; }
is_fe_src()  { [[ "$1" == frontend/src/*.ts || "$1" == frontend/src/*.tsx || "$1" == frontend/src/**/*.ts || "$1" == frontend/src/**/*.tsx ]] && [[ "$1" != *.test.ts && "$1" != *.test.tsx && "$1" != */test-setup.ts && "$1" != */types.ts && "$1" != */main.tsx ]]; }

go_src=(); go_test=(); fe_src=(); fe_test=()
while IFS= read -r f; do
  [ -n "$f" ] || continue
  if   is_go_src "$f"; then go_src+=("$f")
  elif [[ "$f" == *_test.go ]]; then go_test+=("$f")
  elif is_fe_src "$f"; then fe_src+=("$f")
  elif [[ "$f" == frontend/src/*.test.ts || "$f" == frontend/src/*.test.tsx || "$f" == frontend/src/**/*.test.ts || "$f" == frontend/src/**/*.test.tsx ]]; then fe_test+=("$f")
  fi
done <<<"$files"

problems=()
[ ${#go_src[@]} -eq 0 ] || [ ${#go_test[@]} -gt 0 ] || problems+=("Go の実装を変更しましたが *_test.go の変更がありません: ${go_src[*]}")
[ ${#fe_src[@]} -eq 0 ] || [ ${#fe_test[@]} -gt 0 ] || problems+=("フロントの実装を変更しましたが *.test.ts(x) の変更がありません: ${fe_src[*]}")

[ ${#problems[@]} -gt 0 ] || { echo "check-tests-changed OK"; exit 0; }

if [[ "${PR_BODY:-}" == *"[no-test-needed]:"* ]]; then
  echo "check-tests-changed: [no-test-needed] の宣言があるため許可します"
  printf '  %s\n' "${problems[@]}"
  exit 0
fi
printf 'テストの更新が必要です:\n'; printf '  - %s\n' "${problems[@]}"
echo "  テストを追加・更新してください。リファクタリングで挙動が不変なら PR 本文に '[no-test-needed]: 理由' を書きます。"
[ "$mode" = staged ] && exit 0   # コミット時は警告のみ（プッシュ前・CI で強制）
exit 1
