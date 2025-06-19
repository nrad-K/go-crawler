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

type CrawlMode string

const (
	Auto   CrawlMode = "auto"
	Manual CrawlMode = "manual"
)

// CrawlerConfigはクローラーの動作設定をまとめる構造体です。
type CrawlerConfig struct {
	Mode                    CrawlMode         `yaml:"mode" validate:"required,oneof=auto manual"`
	Strategy                CrawlStrategy     `yaml:"strategy" validate:"required,oneof=next_link total_count url_list"` // クロール戦略（次へボタンをたどるか、総件数からページ数を計算するか）
	BaseURL                 string            `yaml:"base_url" validate:"url"`                                           // クロールを開始するベースURL
	JobDetailResolveBaseURL string            `yaml:"job_detail_resolve_base_url" validate:"omitempty,url"`              // 求人詳細リンクが相対パスだった場合に使用する明示的な基準URL
	CrawlSleepSeconds       int               `yaml:"crawl_sleep_seconds" validate:"min=1,max=60"`                       // 各リクエスト間の待機時間（秒）
	CrawlTimeoutSeconds     int               `yaml:"crawl_timeout_seconds" validate:"min=1,max=300"`                    // リクエストのタイムアウト時間（秒）
	EnableHeadless          bool              `yaml:"enable_headless"`
	UserAgent               string            `yaml:"user_agent" validate:"required,min=1"` // リクエストヘッダーに設定するUser-Agent
	OutputDir               string            `yaml:"output_dir" validate:"required"`       // クロール結果を保存するディレクトリ
	Headers                 map[string]string `yaml:"headers"`                              // リクエストに追加するカスタムヘッダー
	Selector                CrawlerSelector   `yaml:"selector" validate:"required"`         // クロール対象要素のCSSセレクター設定
	Pagination              PaginationConfig  `yaml:"pagination" validate:"required"`       // ページネーションに関する設定
	Urls                    []string          `yaml:"urls"`                                 // クロール対象のURLリスト（url_list戦略の場合必須）
	WorkerNum               int               `yaml:"worker_num" validate:"min=1,max=10"`   // 並列実行するワーカーの数
}

// CrawlerSelectorはWebページから特定の要素を選択するためのCSSセレクターを定義します。
type CrawlerSelector struct {
	ListLinksSelector   string `yaml:"list_links_selector" validate:"required,min=1"`   // 一覧ページのリンクのCSSセレクター(複数)
	NextPageLocator     string `yaml:"next_page_locator"`                               // 次のページへのリンクのロケータ-,CrawlByNextLink戦略用）(単一)
	TotalCountSelector  string `yaml:"total_count_selector"`                            // 総件数を取得するためのCSSセレクター（CrawlByTotalCount戦略用）(単一)
	TabClickSelector    string `yaml:"tab_click_selector"`                              // 詳細画面でclickした時にtabで遷移させるセレクター
	DetailLinksSelector string `yaml:"detail_links_selector" validate:"required,min=1"` // 求人（または詳細情報）リンクのCSSセレクター(複数)
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
	Type            PaginationType `yaml:"type" validate:"required,oneof=query path segment none"` // ページネーションのタイプ
	ParamIdentifier string         `yaml:"param_identifier"`                                       // ページネーションを識別するための文字列
	PageFormat      string         `yaml:"page_format"`                                            // ページ番号の書式指定 (path/segmentタイプ用)
	Start           int            `yaml:"start" validate:"min=0"`                                 // ページネーションの開始番号
	PerPage         int            `yaml:"per_page" validate:"min=1,max=1000"`                     // 1ページあたりの項目数
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
		return CrawlerConfig{}, fmt.Errorf("total_count戦略にはtotal_count_selectorが必要です")
	}
	if cfg.Strategy == CrawlByNextLink && cfg.Selector.NextPageLocator == "" {
		return CrawlerConfig{}, fmt.Errorf("next_link戦略にはnext_page_selectorが必要です")
	}
	if cfg.Mode == Manual && len(cfg.Urls) == 0 {
		return CrawlerConfig{}, fmt.Errorf("url_list戦略にはurlsが必要です")
	}
	if cfg.Pagination.Type != None && cfg.Pagination.ParamIdentifier == "" {
		return CrawlerConfig{}, fmt.Errorf("ページネーションタイプがnone以外の場合はparam_identifierが必要です")
	}

	return cfg, nil
}
