package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
	"github.com/nrad-K/go-crawler/internal/usecase"
)

func main() {
	// // --------------------------------------------------
	// // クローラーの実行
	// // --------------------------------------------------
	// // 設定の読み込み
	// crawlerCfg, err := config.NewCrawlerConfig()
	// if err != nil {
	// 	log.Fatalf("failed to load crawler config: %v", err)
	// }

	// // ロガーの初期化
	// appLogger, err := logger.NewAppLogger()
	// if err != nil {
	// 	log.Fatalf("failed to initialize logger: %v", err)
	// }
	// defer appLogger.Sync()

	// // DBの初期化
	// db, err := db.NewDB(crawlerCfg.DSN)
	// if err != nil {
	// 	log.Fatalf("failed to initialize db: %v", err)
	// }
	// defer db.Close()

	// // 依存関係の注入
	// client := infra.NewCrawlJobClient(db)
	// browser := infra.NewBrowserClient(appLogger, crawlerCfg.Playwright)
	// crawler := usecase.NewCrawler(client, browser, appLogger, crawlerCfg)

	// // クローラの実行
	// if err := crawler.Run(context.Background()); err != nil {
	// 	log.Fatalf("failed to run crawler: %v", err)
	// }

	// --------------------------------------------------
	// スクレイパーの実行
	// --------------------------------------------------
	// ロガーの初期化

	logHandler := slog.NewTextHandler(os.Stdout, nil)
	appLogger := logger.NewAppLogger(slog.New(logHandler))

	scraperCfg, err := config.LoadScraperConfig()
	if err != nil {
		log.Fatalf("failed to load scraper config: %v", err)
	}

	// 依存関係の注入
	loader := infra.NewHTMLFileLoader()
	document := infra.NewHTMLDocument()
	patterns := infra.CompiledPatterns{
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
	parser := infra.NewJobPostingParser(patterns)

	headers := []string{
		"会社名", "タイトル", "URL",
		"勤務地(都道府県コード)", "勤務地(都道府県)", "勤務地(市区町村)", "勤務地(原文)",
		"本社(都道府県コード)", "本社(都道府県)", "本社(市区町村)", "本社(原文)",
		"雇用形態", "給与(下限)", "給与(上限)", "給与(単位)", "投稿日",
		"職務内容", "昇給", "賞与", "業務内容詳細", "応募要件", "勤務形態", "年間休日", "休日・休暇", "勤務時間", "福利厚生(原文)",
	}

	exporter, err := infra.NewCSVExporter(
		filepath.Join(scraperCfg.OutputDir, "job_postings.csv"),
		headers,
	)
	if err != nil {
		log.Fatalf("failed to create csv exporter: %v", err)
	}

	args := usecase.ScraperArgs{
		Loader:   *loader,
		Document: document,
		Exporter: exporter,
		Cfg:      scraperCfg,
		Parser:   parser,
		Logger:   appLogger,
	}
	scraper := usecase.NewSaveJobPostingFromHTMLUseCase(args)

	// スクレイパーの実行
	if err := scraper.SaveJobPostingCSV(context.Background()); err != nil {
		log.Fatalf("failed to run scraper: %v", err)
	}
}
