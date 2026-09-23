# jaipur バージョニング

## 方式
- Semantic Versioning。タグは annotated の `vX.Y.Z`（`v` プレフィックス必須）。
- 履歴:
  - `v0.1.0` → `04f51c9`（履歴上の「ver 0.1」コミットに遡及付与）
  - `v0.2.0` → fb06b22（AVIF対応・ディレクトリモード・`--version` 追加）

## バイナリへのバージョン埋め込み
- `main.go` の `var version = "dev"` にビルド時注入:
  `go build -ldflags "-X main.version=v0.2.0"`
- `flake.nix` の `let version = "..."` が唯一の管理点。`ldflags = [ "-X=main.version=v${version}" ]` で埋め込む。
- `resolveVersion()` は未注入時に `runtime/debug.ReadBuildInfo().Main.Version` にフォールバック。
  - Go 1.24+ の VCS スタンピングにより、タグ付きクリーンな HEAD では `-ldflags` なしの `go build` でもタグ（例 `v0.2.0`）を自動報告する。
  - タグが無い/タグ以降のコミットでは `v0.0.0-<日時>-<hash>[+dirty]` の pseudo-version になる。

## リリース手順
1. `flake.nix` の `version` を新バージョンに更新（git タグと一致させる）
2. コミット（コミットメッセージは AGENTS.md の `(type): タイトル` + `## 問題/原因/修正` 形式）
3. `git tag -a vX.Y.Z -m "..."`
4. `git push && git push --tags`（タグは明示的に push するまでローカルのみ）

## 落とし穴
- **`vendorHash` は依存更新のたびに要更新**。陳腐化していても nix ストアに旧バージョンの
  FOD がキャッシュされていると cache-hit で隠蔽され、`version` を上げた瞬間に
  `hash mismatch in fixed-output derivation` として露呈する。
  エラー中の `got:` の値をそのまま `vendorHash` に設定すればよい。
