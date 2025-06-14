package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
	"github.com/nrad-K/go-crawler/internal/usecase"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// 設定ファイル読み込み
	cfg, err := config.LoadCrawlerConfig("settings/crawler.yaml")
	if err != nil {
		log.Fatalf("設定ファイルの読み込みに失敗: %v", err)
	}

	// logger初期化
	logHandler := slog.NewTextHandler(os.Stdout, nil)
	appLogger := logger.NewAppLogger(slog.New(logHandler))

	// Redisクライアント初期化
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// repository初期化
	repo := infra.NewCrawlJobClient(rdb)

	// browser client初期化
	browserClient, err := infra.NewBrowserClient(&cfg)
	if err != nil {
		log.Fatalf("ブラウザクライアントの初期化に失敗: %v", err)
	}
	defer browserClient.Close()

	// usecase初期化
	crawler := usecase.NewCrawlJobExecutorUseCase(usecase.CrawlerArgs{
		Cfg:    &cfg,
		Client: browserClient,
		Repo:   repo,
		Logger: appLogger,
	})

	// 実行
	appLogger.Info("クローラーを実行します")
	if err := crawler.Run(ctx); err != nil {
		appLogger.Error("クローラー実行中にエラー: %v", err)
		os.Exit(1)
	}
	appLogger.Info("クローラーが正常に完了しました")
}
