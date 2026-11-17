# 判断記録

- 2026-11-17 .claude/settings.json の git push deny を外し、強制 push・タグ push 等のみ deny に変更（ユーザー承認済み）
- 2026-11-17 Wails は v2.10.2 に固定（v2.17 は Go 1.26 必須。go.mod は toolchain 自動切替で 1.26.0）
- 2026-11-17 assistant の usage は message.id で重複排除（最後の値を採用）。user_msgs は tool_result・isMeta・システムタグを除く
- 2026-11-17 applier はシンボリックリンクへの書き込みを拒否。失敗時は作成済みバックアップを削除。ロールバックは apply 履歴のみ対象
- 2026-11-17 runner は Setpgid でプロセスグループごと kill（macOS/Linux 前提）。stdout/stderr は job.log に統合
- 2026-11-17 ロジックは internal/service に集約（app.go は薄い層）。E2E は testdata を TempDir に複製し cwd を書き換えて実行（/home/demo 等に触れない）。Wails バインディングは wails generate module で生成し frontend/wailsjs をコミット
- 2026-11-17 wails.json の author（テンプレートが個人メールを埋めていた）を削除。過去コミットには残っているためユーザーに通知。dmg ビルド/起動確認は macOS 実機が必要で未検証（CI/release で検証する）
- 2026-11-17 build/darwin/Info*.plist は v2.17 CLI が生成した safeBundleID を使っており、固定した Wails v2.10.2 では解決できないため {{.Name}} に置換（CI の dmg-smoke で検出）
- 2026-11-17 独立レビューの承認は、リポジトリオーナー・github-actions・claude の投稿で、本文の最終行がマーカー、sha が head に一致するもののみ有効（公開リポジトリで第三者が承認を偽装できないように）。ブランチ保護（#47）が入るまでは merge-pr.sh を通さない手動マージ自体は防げない
- 2026-11-17 Windows 対応を必須とした（ユーザー判断、#46）。テスト用 claude を bash から Go 製（internal/testutil/fakeclaude）に置換して全 OS で共通化。パス比較は Windows のみ大文字小文字を無視。Unix 専用 API は *_unix.go に分離し check-rules で強制
- 2026-11-17 Windows の `claude.cmd`（npm shim）は Go の exec が引数の特殊文字を拒否するため `claude.exe` を優先。プロンプトに特殊文字（% や " など）を入れない
- 2026-11-17 go-webview2 は Wails v2.10.2 が想定する v1.0.19 に固定（v2.17 時代の go.mod 由来の v1.0.22 だと Windows ビルドが失敗した）。ローカルの Wails CLI も v2.10.2 に揃える（別版は生成物を書き換える）
- 2026-11-17 WSL 対応は「Linux 版を WSL 内（WSLg）で動かす」方式（ユーザー判断）。WebKitGTK 4.1（`-tags webkit2_41`）を使い、Release には添付せず `make linux` でビルド。CI は linux-build-smoke でビルド確認のみ（GUI 起動は未検証）
- 2026-11-17 開発プロセスでの個人情報流出防止（#72。ユーザー補足: 機能ではなく開発の進め方の話）: 検査は決定論的なスクリプト（check-privacy）を pre-commit / pre-push / CI の 3 か所に置き、GitHub への投稿は gh-post.sh に集約して直接の gh 投稿を deny。この環境の OS ユーザー名・ホームのパス・git の実メールも検出対象にする。AI 側はインジェクション対策を rules / agent / review-guidelines / status.sh（外部投稿者のタイトルを出さない）に入れる。当初の解釈（アプリの送信前検査）の実装は stash に退避し #75 で採否確認
- 2026-11-17 履歴に残った作者メールは問題にしない（ユーザー判断、#27 をクローズ）。履歴は書き換えない。新しいコミットは ship.sh が noreply を使う
- 2026-11-17 プロンプトインジェクションを仕組みで弾く（#78）: 検出は目印（警告）にとどめ、権限の境界（settings・security.md・agents・フック・検査/マージのスクリプト）の変更は PreToolUse フックで必ず人の確認（ask）に回す。日常的に編集するファイル（CLAUDE.md・通常のスクリプト・コード）は対象外にして自律実行を妨げない。issue/PR の読み取りは gh-read.sh に一本化し、外部投稿者の内容は <untrusted-data> で囲む
- 2026-11-17 #79 レビュー対応: 信頼する投稿者は所有者と bot 表記（app/…・…[bot]）のみ（claude・github-actions は人が同名アカウントを作れるため外した）。gh run view --log・gh repo view（README）は囲みなしで読めるが、優先度が低く許容する（必要になれば gh-read に寄せる）
- 2026-11-17 承認フロー（#81。ユーザー要望: 承認を求める際は修正内容が伝わるように整理・第一承認者として別エージェントの評価を追加・変更を追跡可能に）: 境界ファイルの変更は change-approver の評価記録（.git 配下・30 分・評価した内容のハッシュに結び付く）が無ければフックが拒否し、APPROVE のときだけ人の確認（なぜ・実行内容・影響・評価の順）。Bash による境界ファイルの書き換えは拒否（Edit / Write を使う）。評価は docs/approval-log.md に残り CI（check-approvals）が確認する。自己承認は規則で禁止（機械では防げない）。Bash の文字列判定は glob・変数・スクリプト経由の迂回を防げない（最終防衛線は人の確認・台帳・独立レビュー）。新しいエージェント定義はセッション再起動まで使えないため、初回の評価は code-reviewer に change-approver.md を読ませて実施した
