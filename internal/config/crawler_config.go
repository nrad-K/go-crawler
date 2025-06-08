package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
)

type CrawlStrategy string

const (
	CrawlByNextLink   CrawlStrategy = "next_link"   // "次へ" ボタンをたどる
	CrawlByTotalCount CrawlStrategy = "total_count" // 件数を取得してページ数を計算
)

// CrawlerConfigはクローラーの動作設定をまとめる構造体です。
type CrawlerConfig struct {
	Strategy        CrawlStrategy     `yaml:"strategy" validate:"required,oneof=next_link total_count"` // クロール戦略（次へボタンをたどるか、総件数からページ数を計算するか）
	BaseURL         string            `yaml:"base_url" validate:"required,url"`                         // クロールを開始するベースURL
	SleepSeconds    int               `yaml:"sleep_seconds" validate:"min=1,max=60"`                    // 各リクエスト間の待機時間（秒）
	TimeoutSeconds  int               `yaml:"timeout_seconds" validate:"min=1,max=300"`                 // リクエストのタイムアウト時間（秒）
	UserAgent       string            `yaml:"user_agent" validate:"required,min=1"`                     // リクエストヘッダーに設定するUser-Agent
	RetryCount      int               `yaml:"retry_count" validate:"min=0,max=5"`                       // リクエストが失敗した際の再試行回数
	OutputDirectory string            `yaml:"output_directory" validate:"required,dirpath"`             // クロール結果を保存するディレクトリ
	Headers         map[string]string `yaml:"headers"`                                                  // リクエストに追加するカスタムヘッダー
	Selector        CrawlerSelector   `yaml:"selector" validate:"required"`                             // クロール対象要素のCSSセレクター設定
	Pagination      PaginationConfig  `yaml:"pagination" validate:"required"`                           // ページネーションに関する設定
}

// CrawlerSelectorはWebページから特定の要素を選択するためのCSSセレクターを定義します。
type CrawlerSelector struct {
	PrefectureLinkSelector string `yaml:"prefecture_link_selector" validate:"required,min=1"` // 都道府県（またはカテゴリ）リンクのCSSセレクター(複数)
	NextPageSelector       string `yaml:"next_page_selector"`                                 // 次のページへのリンクのCSSセレクター（CrawlByNextLink戦略用）(単一)
	TotalCountSelector     string `yaml:"total_count_selector"`                               // 総件数を取得するためのCSSセレクター（CrawlByTotalCount戦略用）(単一)
	JobLinkSelector        string `yaml:"job_link_selector" validate:"required,min=1"`        // 求人（または詳細情報）リンクのCSSセレクター(複数)
}

type PaginationType string

const (
	Query   PaginationType = "query"   // クエリパラメータによるページネーション
	Path    PaginationType = "path"    // パスによるページネーション
	Segment PaginationType = "segment" // URLセグメントによるページネーション
	None    PaginationType = "none"    // ページネーションなし
)

// Strategy Total Countの時にしか必要ない(NextLinkのときはnoneにする)
// PaginationConfigはページネーションの動作に関する設定を定義します。
type PaginationConfig struct {
	Type            PaginationType `yaml:"type" validate:"required,oneof=query path segment none"`                      // ページネーションのタイプ
	ParamIdentifier string         `yaml:"param_identifier" validate:"required_unless=Type none,min=1"`                 // ページネーションを識別するための文字列
	PageFormat      string         `yaml:"page_format" validate:"required_if=Type path required_if=Type segment,min=1"` // ページ番号の書式指定 (path/segmentタイプ用)
	Start           int            `yaml:"start" validate:"min=0"`                                                      // ページネーションの開始番号
	PerPage         int            `yaml:"per_page" validate:"min=1,max=1000"`                                          // 1ページあたりの項目数
}

// バリデーターのインスタンス
var v = validator.New()

// YAMLファイルからCrawlerConfigを読み込む
func LoadCrawlerConfig(path string) (CrawlerConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return CrawlerConfig{}, err
	}

	var cfg CrawlerConfig
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return CrawlerConfig{}, err
	}

	// バリデーション
	if err := v.Struct(cfg); err != nil {
		return CrawlerConfig{}, err
	}

	// カスタムバリデーション
	if cfg.Strategy == CrawlByTotalCount && cfg.Selector.TotalCountSelector == "" {
		return CrawlerConfig{}, fmt.Errorf("total_count_selector required for total_count strategy")
	}
	if cfg.Strategy == CrawlByNextLink && cfg.Selector.NextPageSelector == "" {
		return CrawlerConfig{}, fmt.Errorf("next_page_selector required for next_link strategy")
	}
	if cfg.Pagination.Type != None && cfg.Pagination.ParamIdentifier == "" {
		return CrawlerConfig{}, fmt.Errorf("param_identifier required when pagination type is not none")
	}

	return cfg, nil
}
