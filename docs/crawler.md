# クローラー仕様書

## 概要

このドキュメントは、ウェブページを巡回し、後のスクレイピングのためにHTMLコンテンツを保存するクローラーの仕様について説明します。
クローラーには `auto` と `manual` の2つの主要な動作モードがあります。

## 設定 (`settings/crawler.yaml`)

クローラーの動作は `settings/crawler.yaml` ファイルによって制御されます。利用可能な設定オプションは以下の通りです。

### 一般設定

- `mode` (string): クローラーの動作モード。`auto`または`manual`を指定します。
  - `auto`: `base_url`で指定されたページから`list_links_selector`を使って一覧ページ（例: カテゴリページ）へのリンクを自動検出し、クロールを開始します。`一覧ページ > ページネーションページ > 詳細ページ` のような階層を持つサイトに適しています。
  - `manual`: `urls`で指定されたURLリストを直接クロールの起点（一覧ページ）とします。特定の一覧ページからクロールを開始する場合に使用します。
- `base_url` (string): クロールを開始する基準URL（`auto`モードで使用）。
- `job_detail_resolve_base_url` (string): 求人詳細リンクが相対パスの場合に使用する明示的な基準URL。
- `user_agent` (string): HTTPリクエストに使用するUser-Agent文字列。
- `crawl_sleep_seconds` (integer): 各リクエスト間の待機時間（秒）。
- `crawl_timeout_seconds` (integer): リクエストのタイムアウト時間（秒）。
- `enable_headless` (boolean): ヘッドレスブラウザモードを有効または無効にします。
- `retry_count` (integer): 失敗したリクエストを再試行する回数。
- `output_dir` (string): クロール結果（HTMLファイル）を保存するディレクトリ。
- `worker_num` (integer): クロール用の並行ワーカー数。
- `headers` (map): リクエストに追加するカスタムヘッダーのマップ。

### クロール戦略

- `strategy` (string): 一覧ページ内でのページネーション戦略。
  - `next_link`: 「次へ」ボタンをたどってページを移動します。
  - `total_count`: 総アイテム数に基づいてページ数を計算します。

### CSSセレクター

- `selector`: 操作対象の要素のCSSセレクターのマップ。
  - `list_links_selector` (string): 一覧ページへのリンク（例：カテゴリや都道府県）のCSSセレクター（`auto`モードで使用）。
  - `next_page_locator` (string): 「次のページへ」のリンクのCSSセレクター（`next_link` 戦略で使用）。
  - `total_count_selector` (string): 総アイテム数を含む要素のCSSセレクター（`total_count` 戦略で使用）。
  - `detail_links_selector` (string): 詳細ページへのリンク（例：求人情報）のCSSセレクター。
  - `tab_click_selector` (string): 詳細ページでコンテンツを切り替えるためにクリックするタブ要素のCSSセレクター。

### ページネーション設定

- `pagination`: ページネーションの処理に関する設定。
  - `type` (string): ページネーションのタイプ。`query`、`path`、`segment`、または `none` を指定できます。
  - `param_identifier` (string): ページネーションの識別子（例：クエリパラメータ "page" やパスセグメント）。
  - `page_format` (string): ページ番号の書式指定子（例：`%d`, `%02d`）。
  - `start` (integer): 開始ページ番号。
  - `per_page` (integer): 1ページあたりのアイテム数。

### 対象URL

- `urls` (list of strings): クロールする特定のURLのリスト（`manual`モードで使用）。
