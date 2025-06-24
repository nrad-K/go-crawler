package config

import (
	"fmt"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
)

// SelectorConfigはCSSセレクターを定義します。
type SelectorConfig struct {
	Selector string `yaml:"selector" validate:"required,min=1"`
	Attr     string `yaml:"attr"`
	Regex    string `yaml:"regex"`
}

// SalaryConfigは給与情報のセレクターと正規表現を定義します。
type SalaryConfig struct {
	Selector string `yaml:"selector" validate:"required,min=1"`
}

// DetailsConfigは求人詳細情報のセレクターを定義します。
type DetailsConfig struct {
	JobName         SelectorConfig `yaml:"job_name" validate:"required"`
	Raise           SelectorConfig `yaml:"raise" validate:"required"`
	Bonus           SelectorConfig `yaml:"bonus" validate:"required"`
	Description     SelectorConfig `yaml:"description" validate:"required"`
	Requirements    SelectorConfig `yaml:"requirements" validate:"required"`
	WorkplaceType   SelectorConfig `yaml:"workplace_type" validate:"required"`
	HolidaysPerYear SelectorConfig `yaml:"holidays_per_year" validate:"required"`
	HolidayPolicy   SelectorConfig `yaml:"holiday_policy" validate:"required"`
	WorkHours       SelectorConfig `yaml:"work_hours" validate:"required"`
	Benefits        SelectorConfig `yaml:"benefits" validate:"required"`
}

// ScraperConfigはスクレイパーの動作設定をまとめる構造体です。
type ScraperConfig struct {
	BaseURL      string         `yaml:"base_url" validate:"required,url,min=1"`
	HtmlDir      string         `yaml:"html_dir" validate:"required,min=1"`
	OutputDir    string         `yaml:"output_dir" validate:"required,min=1"`
	MaxWorkers   int            `yaml:"max_workers" validate:"required,gt=0,max=10"`
	FileName     string         `yaml:"file_name" validate:"required,min=1,max=20"`
	Title        SelectorConfig `yaml:"title" validate:"required"`
	CompanyName  SelectorConfig `yaml:"company_name" validate:"required"`
	SummaryURL   SelectorConfig `yaml:"summary_url" validate:"required"`
	Location     SelectorConfig `yaml:"location" validate:"required"`
	Headquarters SelectorConfig `yaml:"headquarters" validate:"required"`
	JobType      SelectorConfig `yaml:"job_type" validate:"required"`
	Salary       SalaryConfig   `yaml:"salary" validate:"required"`
	PostedAt     SelectorConfig `yaml:"posted_at" validate:"required"`
	Details      DetailsConfig  `yaml:"details" validate:"required"`
}

// バリデーターのインスタンス
var validate = validator.New()

// YAMLファイルからScraperConfigを読み込む
func LoadScraperConfig(path string) (ScraperConfig, error) {
	f, err := os.ReadFile(path)
	if err != nil {
		return ScraperConfig{}, fmt.Errorf("設定ファイルを読み込めませんでした: %w", err)
	}

	var cfg ScraperConfig
	if err := yaml.Unmarshal(f, &cfg); err != nil {
		return ScraperConfig{}, fmt.Errorf("YAMLの解析に失敗しました: %w", err)
	}

	// バリデーション
	if err := validate.Struct(cfg); err != nil {
		return ScraperConfig{}, fmt.Errorf("設定のバリデーションに失敗しました: %w", err)
	}

	return cfg, nil
}
