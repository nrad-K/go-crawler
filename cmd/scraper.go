package cmd

import (
	"context"
	"log"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/constants"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
	"github.com/nrad-K/go-crawler/internal/usecase"
	"github.com/spf13/cobra"
)

var scraperCmd = &cobra.Command{
	Use:   "scrape",
	Short: "HTMLファイルから求人情報をスクレイピングします",
	Long:  `ローカルに保存されたHTMLファイルを解析し、設定されたセレクターに基づいて求人情報を抽出し、結果をCSVファイルに保存します`,
	Run: func(cmd *cobra.Command, args []string) {
		logHandler := slog.NewTextHandler(os.Stdout, nil)
		appLogger := logger.NewAppLogger(slog.New(logHandler))

		path := "settings/scraper.yaml"
		scraperCfg, err := config.LoadScraperConfig(path)
		if err != nil {
			log.Fatalf("スクレイプの設定ファイルを読み込めませんでした: %v", err)
		}

		patterns := constants.GetScraperCompiledPatterns()
		headers := constants.GetScraperCSVHeaders()

		loader := infra.NewHTMLFileLoader()
		document := infra.NewHTMLDocument()
		parser := infra.NewJobPostingParser(patterns)
		exporter, err := infra.NewCSVExporter(
			filepath.Join(scraperCfg.OutputDir, scraperCfg.FileName),
			headers,
		)

		if err != nil {
			log.Fatalf("CSVエクスポーターの初期化に失敗しました: %v", err)
		}

		scraperArgs := usecase.ScraperArgs{
			Loader:   *loader,
			Document: document,
			Exporter: exporter,
			Cfg:      scraperCfg,
			Parser:   parser,
			Logger:   appLogger,
		}
		scraper := usecase.NewSaveJobPostingFromHTMLUseCase(scraperArgs)
		if err := scraper.SaveJobPostingCSV(context.Background()); err != nil {
			log.Fatalf("スクレイプに失敗しました: %v", err)
		}
	}}

func init() {
	rootCmd.AddCommand(scraperCmd)
}
