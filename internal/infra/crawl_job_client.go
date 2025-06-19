package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/domain/repository"
	"github.com/redis/go-redis/v9"
)

type crawlJobClient struct {
	redis *redis.Client
}

func NewCrawlJobClient(rds *redis.Client) repository.CrawlJobRepository {
	return &crawlJobClient{
		redis: rds,
	}
}

func (r *crawlJobClient) Save(ctx context.Context, job model.CrawlJob) error {
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("クローリングジョブのマーシャルに失敗しました: %w", err)
	}

	key, err := r.generateJobKey(job)
	if err != nil {
		return fmt.Errorf("ジョブキーの生成に失敗しました: %w", err)
	}

	if err := r.redis.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("クローリングジョブをRedisに保存できませんでした: %w", err)
	}

	return nil
}

func (r *crawlJobClient) Delete(ctx context.Context, job model.CrawlJob) error {
	key, err := r.generateJobKey(job)
	if err != nil {
		return fmt.Errorf("削除用のジョブキーの生成に失敗しました: %w", err)
	}
	if err := r.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("保留中のジョブをRedisから削除できませんでした: %w", err)
	}
	return nil
}

func (r *crawlJobClient) FindListByStatus(ctx context.Context, size int, status model.CrawlJobStatus) ([]model.CrawlJob, error) {
	var jobs []model.CrawlJob
	var cursor uint64
	var err error

	// バッチサイズを設定
	batchSize := int64(size)

	pattern := ""
	switch status {
	case model.CrawlJobStatusSuccess:
		pattern = "success_job:*"
	case model.CrawlJobStatusFailed:
		pattern = "failed_job:*"
	case model.CrawlJobStatusPending:
		pattern = "pending_job:*"
	default:
		return nil, fmt.Errorf("サポートされていないジョブステータスです: %s", status)
	}

	for {
		// SCANコマンドでキーを取得
		var keys []string
		keys, cursor, err = r.redis.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return nil, fmt.Errorf("redisスキャンエラー: %w", err)
		}

		// 取得したキーからジョブデータを取得
		for _, key := range keys {
			value, err := r.redis.Get(ctx, key).Result()
			if err != nil {
				return nil, fmt.Errorf("キー %s のRedis取得エラー: %w", key, err)
			}

			var job model.CrawlJob
			if err := json.Unmarshal([]byte(value), &job); err != nil {
				return nil, fmt.Errorf("キー %s のアンマーシャルエラー: %w", key, err)
			}
			jobs = append(jobs, job)
		}

		// カーソルが0になったら終了
		if cursor == 0 {
			break
		}
	}

	return jobs, nil
}

func (r *crawlJobClient) Exists(ctx context.Context, job model.CrawlJob) (bool, error) {
	key, err := r.generateJobKey(job)
	if err != nil {
		return false, fmt.Errorf("ジョブキーの生成に失敗しました: %w", err)
	}
	exists, err := r.redis.Exists(ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("redisの存在確認に失敗しました: %w", err)
	}
	return exists > 0, nil
}

func (r *crawlJobClient) generateJobKey(job model.CrawlJob) (string, error) {
	var key string
	switch job.Status {
	case model.CrawlJobStatusPending:
		key = r.generatePendingJobKey(job.URL.String())
	case model.CrawlJobStatusSuccess:
		key = r.generateSuccessJobKey(job.URL.String())
	case model.CrawlJobStatusFailed:
		key = r.generateFailedJobKey(job.URL.String())
	default:
		return "", fmt.Errorf("キー生成にサポートされていないジョブステータスです: %s", job.Status)
	}

	return key, nil
}

func (r *crawlJobClient) generateSuccessJobKey(url string) string {
	return fmt.Sprintf("success_job: %s", url)
}

func (r *crawlJobClient) generateFailedJobKey(url string) string {
	return fmt.Sprintf("failed_job: %s", url)
}

func (r *crawlJobClient) generatePendingJobKey(url string) string {
	return fmt.Sprintf("pending_job:%s", url)
}
