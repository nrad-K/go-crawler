package cmd

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
	"github.com/nrad-K/go-crawler/internal/usecase"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/cobra"
)

var (
	generate bool
	execute  bool
)

var crawlerCmd = &cobra.Command{
	Use:   "crawler",
	Short: "求人情報をクロールし、HTMLを保存します",
	Long:  `設定に基づき、求人情報のURLを収集（--generate）し、各URLのHTMLコンテンツを保存（--execute）します。`,
	Run: func(cmd *cobra.Command, args []string) {
		if !generate && !execute {
			cmd.Help()
			return
		}

		ctx := context.Background()

		err := godotenv.Load()
		if err != nil {
			// build 時の時は何もしない
		}

		// 設定ファイル読み込み
		path := "settings/crawler.yaml"
		cfg, err := config.LoadCrawlerConfig(path)
		if err != nil {
			log.Fatalf("設定ファイルの読み込みに失敗: %v", err)
		}

		// logger初期化
		logHandler := slog.NewTextHandler(os.Stdout, nil)
		appLogger := logger.NewAppLogger(slog.New(logHandler))

		// Redisクライアント初期化
		rdb := redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_ADDRESS"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		})
		// Redisへの接続を確認 (ping)
		if err := rdb.Ping(ctx).Err(); err != nil {
			appLogger.Error("Redisへの接続に失敗しました", "error", err)
			os.Exit(1)
		}
		appLogger.Info("Redisへの接続を確認しました")

		// repository初期化
		repo := infra.NewCrawlJobClient(rdb)

		// browser client初期化
		browserClient, err := infra.NewBrowserClient(&cfg)
		if err != nil {
			log.Fatalf("ブラウザクライアントの初期化に失敗: %v", err)
		}
		defer browserClient.Close()

		ucArgs := usecase.CrawlerArgs{
			Cfg:    &cfg,
			Client: browserClient,
			Repo:   repo,
			Logger: appLogger,
		}

		// crawl generate
		if generate {
			generateUC := usecase.NewGenerateCrawlJobUseCase(ucArgs)
			appLogger.Info("クロールジョブの生成を開始します")
			if err := generateUC.GenerateCrawlJob(ctx); err != nil {
				appLogger.Error("クロールジョブの生成中にエラーが発生しました", "error", err)
				os.Exit(1)
			}
			appLogger.Info("クロールジョブの生成が正常に完了しました")
		}

		// crawl execute
		if execute {
			executeUC := usecase.NewExecuteCrawlJobUseCase(ucArgs)
			appLogger.Info("クロールジョブの実行を開始します")
			if err := executeUC.ExecuteCrawlJob(ctx); err != nil {
				appLogger.Error("クロールジョブの実行中にエラーが発生しました", "error", err)
				os.Exit(1)
			}
			appLogger.Info("クロールジョブの実行が正常に完了しました")
		}
	},
}

func init() {
	rootCmd.AddCommand(crawlerCmd)
	crawlerCmd.Flags().BoolVarP(&generate, "generate", "g", false, "クロールジョブを生成します")
	crawlerCmd.Flags().BoolVarP(&execute, "execute", "e", false, "クロールジョブを実行します")
}
