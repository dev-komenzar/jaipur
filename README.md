# Jaipur

画像アーカイブファイル（ZIP/RAR）内の画像をmozjpegで圧縮し、再アーカイブするCLIツールです。

## 機能

- ZIP/RARアーカイブ内の画像をmozjpegで圧縮
- 複数ファイルの並列処理（CPUコア数の半分、最低1）
- ファイル名から不要な文字列を削除
- 圧縮品質の指定

## 対応フォーマット

### 入力フォーマット
| フォーマット | 拡張子 |
|-------------|--------|
| JPEG | `.jpg`, `.jpeg` |
| PNG | `.png` |
| GIF | `.gif` |
| AVIF | `.avif` |

### 出力フォーマット
| フォーマット | 説明 |
|-------------|------|
| JPEG | mozjpegによる高品質圧縮（プログレッシブJPEG） |

すべての入力画像は `.jpg` 形式に変換されます。

## 必要条件

- Go 1.23以上
- mozjpeg（システムにインストール済みであること）

## インストール

```bash
go install
```

## 使い方

```bash
# 単一ファイルを変換
jaipur convert [ファイル名.zip]

# ディレクトリ内の全アーカイブを変換
jaipur convert [ディレクトリパス]

# 品質を指定（デフォルト: 70）
jaipur convert --quality 80 [ファイル名.zip]
jaipur convert -q 80 [ファイル名.zip]

# ファイル名から特定の文字列を削除
jaipur convert --remove "不要な文字列" [ファイル名.zip]
```

### ディレクトリモード

画像が入ったディレクトリを直接処理することもできます。

```bash
# 画像ディレクトリをZIPに圧縮
jaipur convert --directory ./images

# 画像ディレクトリを別ディレクトリに出力（アーカイブ化しない）
jaipur convert --directory ./images --output dir

# 品質指定も併用可能
jaipur convert --directory ./images -q 80 --output zip
```

**注意**: `--directory`オプション使用時にZIP/RARファイルを引数に指定するとエラーになります。

## オプション

| オプション | 短縮形 | 説明 | デフォルト |
|-----------|--------|------|-----------|
| `--quality` | `-q` | mozjpegの圧縮品質（1-100） | 70 |
| `--remove` | - | ファイル名から削除する文字列 | - |
| `--directory` | `-d` | 画像ディレクトリのパスを指定 | - |
| `--output` | `-o` | 出力形式 (`zip` or `dir`) | `zip` |

## 出力

- 変換後のファイルは `box/` ディレクトリに保存されます
- 変換完了した元ファイルは `done/` ディレクトリに移動されます
- 250MB以上のファイルには `(BIG)` プレフィックスが付きます
- エラーが発生したファイルには `(ERROR)` プレフィックスが付きます

## 依存ライブラリ

- [urfave/cli](https://github.com/urfave/cli) - CLIフレームワーク
- [mholt/archiver](https://github.com/mholt/archiver) - アーカイブ処理
- [nickalie/go-mozjpegbin](https://github.com/nickalie/go-mozjpegbin) - mozjpegバインディング
- [gen2brain/avif](https://github.com/gen2brain/avif) - AVIFデコーダー
- [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup) - 並列処理

## ライセンス

MIT
