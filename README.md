# go-crawler

求人情報サイトのクローリングとスクレイピングを行うツールです。
go-crawlerは、求人情報のURLを収集するクローラー機能と、ダウンロード済みのHTMLファイルから詳細情報を抽出するスクレイパー機能を提供します。

## クイックスタート

1. **環境変数の設定**

`.env`ファイルを作成し、以下の環境変数を設定します。

```env
REDIS_ADDRESS=localhost:6379
REDIS_PASSWORD=
```

2. **Redisの起動**

Docker Composeを使用してRedisコンテナを起動します。

```bash
docker-compose up -d
```

3. **ビルドと実行**

`make build`コマンドでアプリケーションをビルドします。

```bash
make build
```

`make all`コマンドで、全ての処理（クローリングとスクレイピング）を一括で実行できます。

```bash
make all
```

## 推奨環境

- Go: 1.24.2
- Docker
- Docker Compose

## 必要なもの

このツールは内部で Playwright を使用しています。
以下のコマンドを実行して、Playwright とその依存関係をインストールしてください。

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.0 install --with-deps
```

参考: [playwright-community/playwright-go](https://github.com/playwright-community/playwright-go)

## 使い方

### `crawler`

求人情報をクロールし、HTMLを保存します。
設定に基づき、求人情報のURLを収集（`--generate`）し、各URLのHTMLコンテンツを保存（`--execute`）します。

#### フラグ

- `--generate`, `-g`: クロールジョブを生成します。
- `--execute`, `-e`: 生成されたクロールジョブを実行し、HTMLをダウンロードします。

#### 実行例

クロールジョブの生成:

```bash
./go-crawler crawler --generate
```

クロールジョブの実行:

```bash
./go-crawler crawler --execute
```

### `scrape`

ローカルに保存されたHTMLファイルを解析し、設定されたセレクターに基づいて求人情報を抽出し、結果をCSVファイルに保存します。

#### 実行例

```bash
./go-crawler scrape
```

## 設定

クローリングとスクレイピングの挙動は、以下のYAMLファイルで設定します。

- `settings/crawler.yaml`: クローラーの設定ファイル
- `settings/scraper.yaml`: スクレイパーの設定ファイル
