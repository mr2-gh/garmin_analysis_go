# Garmin CLI (Garmin Connect Cycling Data Analyzer)

Garmin Connectからサイクリングデータ（実走・インドア）を抽出し、トレーニング負荷（TSS）およびPMC（Performance Management Chart）モデルに基づいた各種指標（CTL / ATL / TSB）の計算と可視化を行うGo製のCLIツールです。

Python + Streamlitで作成された元のスクリプトをGo言語に移植し、シングルバイナリでの高速な動作と簡便な環境構築を実現しました。

## 主な機能

- **Go言語による高速動作とポータビリティ**: 依存関係のセットアップが不要で、ビルドされたバイナリ単体で動作します。
- **厳密なサイクリング限定フィルター**: `road_biking`（実走）および `indoor_cycling`（インドアサイクリング）のアクティビティのみを抽出。
- **MFA（多要素認証）対応**: 安全なログイン処理と、取得したセッション（OAuth1 / OAuth2 トークン）のローカルキャッシュによるレート制限の回避。
- **CSVへのエクスポート**: 抽出したアクティビティデータを文字化けの少ないBOM付きCSV形式（`garmin_cycling_activities.csv`）で保存。
- **CLIコンソール表示**: ターミナル上で直近のアクティビティリストや、週次の統計サマリー、簡易アスキーアートグラフを表示。
- **インタラクティブなWebダッシュボード**: `--web`フラグを指定することで、ローカルWebサーバーを起動。埋め込まれたモダンなWeb UI（Plotly.jsを使用）により、ブラウザ上で美麗なダッシュボードを表示・操作可能。

---

## 構成ファイルへのリンク

- [main.go](main.go): アプリケーションのエントリーポイント
- [cmd/](cmd/): CLIコマンドの定義（Cobra）
  - [fetch.go](cmd/fetch.go): Garmin Connectからのデータ取得
  - [list.go](cmd/list.go): アクティビティ履歴のコンソール表示
  - [stats.go](cmd/stats.go): 統計の計算、コンソール表示、Webサーバーの起動
  - [index.html](cmd/index.html): ダッシュボード用Webフロントエンド（埋め込み用）
- [pkg/](pkg/): コアロジック
  - [pmc/pmc.go](pkg/pmc/pmc.go): CTL/ATL/TSBの計算ロジック
  - [data/parser.go](pkg/data/parser.go): CSVデータのロード・保存およびパース処理
- [internal/garmin/](internal/garmin/): Garmin Connect APIクライアント

---

## インストールとビルド

### 前提条件
- Go 1.21 以上がインストールされていること。

### ビルド手順

リポジトリルート（`garmin-cli` ディレクトリ）で以下を実行します。

```bash
# 依存モジュールの解決
go mod tidy

# バイナリのビルド
go build -o garmin-cli main.go
```

ビルドが完了すると、実行ファイル `garmin-cli` （Windowsの場合は `garmin-cli.exe`）が生成されます。

---

## 環境設定（.env）

実行ファイルと同じディレクトリに `.env` ファイルを作成し、以下の項目を設定してください。

```env
GARMIN_USERNAME="your_email@example.com"
GARMIN_PASSWORD="your_password_here"
MAX_ACTIVITY_NUM=50
```

- **`GARMIN_USERNAME`**: Garmin Connectのログイン用メールアドレス。
- **`GARMIN_PASSWORD`**: Garmin Connectのログインパスワード。
- **`MAX_ACTIVITY_NUM`**: （オプション）Garmin Connectから遡って取得するサイクリングアクティビティの最大件数（デフォルト: 50）。

### セッションキャッシュについて
初回ログインに成功すると、認証情報がOS標準の設定ディレクトリ（`GARMIN_CONFIG_DIR`）にキャッシュされます。
- macOS: `~/Library/Application Support/garmin`
- Linux: `~/.config/garmin`
- Windows: `%APPDATA%\garmin`

次回以降はキャッシュされたトークンを再利用するため、不要なログイン認証を防ぐことができます。
キャッシュ先を変更したい場合は、環境変数 `GARMIN_CONFIG_DIR` に任意のディレクトリパスを設定してください。

---

## 使い方

生成された `garmin-cli` コマンドを使用して、データの取得・表示・分析を行います。

### 1. データのダウンロード (`fetch`)

Garmin Connectから最新のアクティビティを取得し、CSVファイル（`garmin_cycling_activities.csv`）に保存します。

```bash
./garmin-cli fetch
```

- 初回実行時やセッションが切れた場合、多要素認証（MFA）のワンタイムコード入力を求められます。メールなどで受信したコードをターミナルに入力してください。
- ロードバイク（`road_biking`）とインドアサイクリング（`indoor_cycling`）のアクティビティのみがフィルタリングされ保存されます。

### 2. アクティビティ履歴の表示 (`list`)

ローカルに保存されているCSVからアクティビティの一覧を表示します。

```bash
./garmin-cli list
```

**主なオプション:**
- `-n, --limit <件数>`: 表示するアクティビティの最大件数を指定します（デフォルト: 20）。
- `-t, --type <タイプ>`: アクティビティタイプで絞り込みます（例: `road_biking` / `indoor_cycling`）。

### 3. トレーニング負荷とPMC統計の表示 (`stats`)

トレーニング状態（FTP、CTL、ATL、TSB）のサマリーや、週次の進捗をターミナル上に表示します。

```bash
./garmin-cli stats
```

#### インタラクティブなWebダッシュボードの表示

`-w` または `--web` フラグを付与することで、内蔵Webサーバーを起動し、ブラウザでよりリッチなビジュアルダッシュボードを表示できます。

```bash
./garmin-cli stats --web
# ポートを変更する場合（デフォルトは 8501）
./garmin-cli stats --web --port 8080
```

起動後、自動的にブラウザが開き、グラフ（Plotly）を用いた進捗やパワーゾーン割合、トレーニング負荷の推移などの分析画面が表示されます。

---

## PMC（パフォーマンス管理チャート）モデルの定義

本ツールでは、トレーニングによる身体の変化を把握するため、以下の指標を計算しています。

- **CTL（Chronic Training Load / 体力）**
  - 過去42日間のTSS（Training Stress Score）の指数平滑移動平均です。長期的なトレーニングの積み重ねを表します。
- **ATL（Acute Training Load / 疲労）**
  - 過去7日間のTSSの指数平滑移動平均です。短期的な負荷（疲れ）の蓄積を表します。
- **TSB（Training Stress Balance / 調子）**
  - `前日のCTL - 前日のATL` で計算されます。レースに向けてコンディションが整っているか（テーパリングがうまくいっているか）の指標となります。正の値であれば疲労が抜け、負の値であればトレーニング負荷がかかっている状態を表します。

---

## 免責事項

本ツールは個人利用を目的とした非公式のツールです。Garmin公式が提供・サポートするものではありません。Garmin Connect APIの仕様変更などにより、予告なく動作しなくなる可能性があります。
