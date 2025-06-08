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
		return fmt.Errorf("failed to marshal crawl job: %w", err)
	}

	key, err := r.generateJobKey(job)
	if err != nil {
		return fmt.Errorf("failed to generate job key: %w", err)
	}

	if err := r.redis.Set(ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save crawl job to redis: %w", err)
	}

	return nil
}

func (r *crawlJobClient) Delete(ctx context.Context, job model.CrawlJob) error {
	key, err := r.generateJobKey(job)
	if err != nil {
		return fmt.Errorf("failed to generate job key for deletion: %w", err)
	}
	if err := r.redis.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete pending job from redis: %w", err)
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
		return nil, fmt.Errorf("unsupported job status: %s", status)
	}

	for {
		// SCANコマンドでキーを取得
		var keys []string
		keys, cursor, err = r.redis.Scan(ctx, cursor, pattern, batchSize).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan error: %w", err)
		}

		// 取得したキーからジョブデータを取得
		for _, key := range keys {
			value, err := r.redis.Get(ctx, key).Result()
			if err != nil {
				return nil, fmt.Errorf("redis get error for key %s: %w", key, err)
			}

			var job model.CrawlJob
			if err := json.Unmarshal([]byte(value), &job); err != nil {
				return nil, fmt.Errorf("unmarshal error for key %s: %w", key, err)
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
		return "", fmt.Errorf("unsupported job status for key generation: %s", job.Status)
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
