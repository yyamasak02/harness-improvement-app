---
paths: ["CHANGELOG.md", "wails.json", "scripts/release.sh", "scripts/check-version.sh", ".github/workflows/release.yml"]
description: バージョン・CHANGELOG・リリースタグの規則（リリースに関わる作業で必ず従う）
---
# リリース規則

## バージョンとタグ
- **SemVer**（`MAJOR.MINOR.PATCH`）。タグは `vX.Y.Z`（pre-release は `vX.Y.Z-rc.N`。`-` を含むタグは GitHub 上で pre-release になる）。
- バージョンの**単一の出典は `wails.json` の `info.productVersion`**。`frontend/package.json` と一致させる（`scripts/check-version.sh` が CI・pre-commit で検査）。
- 0.x の間は、互換性を壊す変更は MINOR、機能追加・修正は PATCH でもよい。1.0.0 以降は SemVer を厳守する。

## CHANGELOG
- `CHANGELOG.md`（Keep a Changelog 形式）の `## [Unreleased]` に、**`feat` / `fix` の PR で必ず追記**する（`scripts/check-changelog.sh` が CI で検査。ユーザーに見えない変更は PR 本文に `[no-changelog]: 理由`）。
- 区分: 追加 / 変更 / 非推奨 / 削除 / 修正 / セキュリティ。ユーザーが読んで分かる言葉で書く（実装の詳細は書かない）。
- リリース時に `scripts/release.sh prepare` が `[Unreleased]` を `[X.Y.Z] - 日付` に確定する。Release ノートはこの節から生成される。

## 手順（`release` skill に詳細）
1. 事前確認: open の p0/p1 issue、人の検証（`type/verification`・`status/needs-human`）が済んでいること。
2. `bash scripts/release.sh prepare X.Y.Z` → リリース PR（`chore: vX.Y.Z のリリース準備`）→ 独立レビュー → `merge-pr.sh`
3. `bash scripts/release.sh tag X.Y.Z`（main・CI 成功・整合を検査し、ローカルに注釈付きタグを作る）
4. **タグの push は人が行う**（`git push origin vX.Y.Z`。エージェントは権限上できない・やってはいけない）。
5. push 後、`release.yml` が検証 → macOS / Windows ビルド → Release 作成・成果物添付。成功を確認する。

## 初回リリースは rc で配布経路を検証する
- 最初の正式版（`v0.1.0`）の前に、`v0.1.0-rc.1` を切って `release.yml`（macOS / Windows のビルドと Release 添付）が実際に動くことを確認する。`-` を含むタグは自動で pre-release になる。
- 手順は通常と同じ（`release.sh prepare 0.1.0-rc.1` → PR → `release.sh tag 0.1.0-rc.1` → 人が push）。push 後に Release へ `harness-app.dmg`・`*installer.exe` が添付されたかを確認し、問題があれば修正して `rc.2` を出す（同じ番号は使い回さない）。
- rc で問題が無ければ、同じ内容で正式版を出す。rc から変更が無く `[Unreleased]` が空でも `release.sh prepare 0.1.0` は通る（rc の節を参照する一文が入る）。
- 正式版が出るまでは README の「インストール」に暫定手順（ソースビルド）を残す。正式版の公開後に、その案内を整理する。

## 禁止
- タグを手で作って push しない／既存タグを付け替えない・削除しない（公開済みのタグは不変）。
- 検証（check-version・CI）を飛ばして Release を作らない。`gh release` は使わない（権限で禁止）。
- 問題があれば修正版を PATCH で出す（同じ番号を使い回さない）。
