# jaipur - 画像アーカイブ・変換ツール

## ファイル探索にはSerena MCPを使う

## 自己改善ループ

- ユーザーから修正を受けたら、Serena MCPを使ってそのパターンを記録する
- 同じミスを繰り返さないように、自分へのルールを書く
- ミス率が下がるまで、ルールを徹底的に改善し続ける
- セッション開始時にserenaの「教訓（Lessons）」を参照し、プロジェクトに関連する教訓をレビューする

### 自律的なバグ修正

- バグレポートを受けたら、手取り足取り教えてもらわずにそのまま修正する
- ログ・エラー・失敗しているテストを見て、自分で解決する
- ユーザーのコンテキスト切り替えをゼロにする

## コミットメッセージの形式

```
(fix|docs|add|update|feat|chore): タイトル

## 問題
- 問題の記述

## 原因
- 分析した問題の原因を説明

## 修正
- 実行した修正項目
```
- それぞれを箇条書きで5行以内にまとめる

## リポジトリ構造

```
jaipur/
├── main.go                  # CLIエントリポイント（urfave/cli）。convertコマンド定義、フラグ解析、アーカイブ modes・ディレクトリ modes の振り分け
├── sub.go                    # ユーティリティ関数群。ファイル名リネーム、プレフィックス付与（(ERROR)/(BIG)）、パス操作、shellエスケープ解除
├── README.md                 # プロジェクト説明・使い方・オプション一覧
├── flake.nix                 # Nix開発環境定義（Go toolchain, mozjpeg, ビルド設定）
├── flake.lock                # Nix依存関係ロックファイル
├── go.mod                    # Goモジュール定義（module: jaipur, go 1.23）
├── go.sum                    # Go依存関係チェックサム
└── pkg/
    ├── convert/
    │   ├── convert.go        # 変換の中核ロジック。Convert（アーカイブ解凍→mozjpeg圧縮→再アーカイブ）と ConvertDirectory（ディレクトリ直销・ZIP/dir出力）
    │   ├── progress.go       # スレッドセーフな進捗カウンター（ProgressCounter）。並列処理中の進捗表示
    │   └── worker.go         # mozjpeg実行（runMozjpeg）とAVIF→PNG変換（convertAVIFtoPNG）。全画像を.jpgに変換
    └── read/
        └── read.go           # ディレクトリからZIP/RARファイルのみを抽出（ReadDir）
```

### 処理フロー

1. `main.go` が CLI 引数を解析し、`--directory` の有無で modes を振り分け
2. **アーカイブ modes**: ZIP/RAR を `pkg/convert/Convert` で解凍 → 画像抽出 → mozjpeg 圧縮 → 再アーカイブ
3. **ディレクトリ modes**: `pkg/convert/ConvertDirectory` で画像ディレクトリを直接処理 → ZIP or ディレクトリ出力
4. 変換後ファイルは `box/` に格納、元ファイルは `done/` に移動
5. 250MB超のファイルには `(BIG)`、エラーファイルには `(ERROR)` プレフィックスを付与

### 主要パッケージの依存関係

- `main.go` → `pkg/convert`（converter）, `pkg/read`（read）
- `pkg/convert` → `archiver/v3`（アーカイブ操作）, `mozjpegbin`（mozjpeg実行）, `avif`（AVIFデコード）, `errgroup`（並列処理）
- `pkg/read` → 標準ライブラリのみ

### 実行時の出力ディレクトリ

| ディレクトリ | 用途 |
|-------------|------|
| `box/` | 変換後のファイルが出力される |
| `done/` | 変換が完了した元ファイルが移動される |
