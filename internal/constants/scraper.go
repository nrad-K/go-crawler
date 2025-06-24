package constants

import (
	"regexp"

	"github.com/nrad-K/go-crawler/internal/infra"
)

// GetScraperCompiledPatternsは、スクレイパーで使用するコンパイル済みの正規表現パターンを返します。
func GetScraperCompiledPatterns() infra.CompiledPatterns {
	return infra.CompiledPatterns{
		RaisePatterns: []*regexp.Regexp{
			regexp.MustCompile(`昇給[／/]年(\d+)回`),
			regexp.MustCompile(`昇給.*年(\d+)回`),
			regexp.MustCompile(`年(\d+)回.*昇給`),
			regexp.MustCompile(`昇給.*(\d+)回[／/]年`),
			regexp.MustCompile(`昇給.*(\d+)回.*年`),
		},
		BonusPatterns: []*regexp.Regexp{
			regexp.MustCompile(`賞与[／/]年(\d+)回`),
			regexp.MustCompile(`賞与.*年(\d+)回`),
			regexp.MustCompile(`年(\d+)回.*賞与`),
			regexp.MustCompile(`賞与.*(\d+)回[／/]年`),
			regexp.MustCompile(`賞与.*(\d+)回.*年`),
			regexp.MustCompile(`ボーナス[／/]年(\d+)回`),
			regexp.MustCompile(`ボーナス.*年(\d+)回`),
		},
		AmountPattern:       regexp.MustCompile(`(\d+(?:\.\d+)?)`),
		SalaryRangePattern:  regexp.MustCompile(`([\d.,]+(?:万|千|億)?円?)\s*[~～]\s*([\d.,]+(?:万|千|億)?円?)`),
		SalarySinglePattern: regexp.MustCompile(`(\d+(?:\.\d+)?[万億千]?)`),
		LocationPattern:     regexp.MustCompile(`(?:都|道|府|県)(.+?[市区町村])`),
	}
}

// GetScraperCSVHeadersは、スクレイパーが出力するCSVファイルのヘッダーを返します。
func GetScraperCSVHeaders() []string {
	return []string{
		"会社名", "タイトル", "URL",
		"勤務地(都道府県コード)", "勤務地(都道府県)", "勤務地(市区町村)", "勤務地(原文)",
		"本社(都道府県コード)", "本社(都道府県)", "本社(市区町村)", "本社(原文)",
		"雇用形態", "給与(下限)", "給与(上限)", "給与(単位)", "投稿日",
		"職務内容", "昇給", "賞与", "業務内容詳細", "応募要件", "勤務形態", "年間休日", "休日・休暇", "勤務時間", "福利厚生(原文)",
	}
}

const (
	LogBatchCount = 100
)
