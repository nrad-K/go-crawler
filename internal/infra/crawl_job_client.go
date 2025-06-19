package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/domain/repository"
	"github.com/redis/go-redis/v9"
)

// crawlJobClientは、Redisを用いたCrawlJobRepositoryの実装です。
type crawlJobClient struct {
	redis *redis.Client
}

// NewCrawlJobClientは、crawlJobClientの新しいインスタンスを作成します。
//
// args:
//
//	rds: Redisクライアント
//
// return:
//
//	repository.CrawlJobRepository: 生成されたリポジトリ実装
func NewCrawlJobClient(rds *redis.Client) repository.CrawlJobRepository {
	return &crawlJobClient{
		redis: rds,
	}
}

// Saveは、CrawlJobをRedisに保存します。
//
// args:
//
//	ctx: コンテキスト
//	job: 保存するCrawlJob
//
// return:
//
//	error: 保存に失敗した場合のエラー
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

// Deleteは、指定したCrawlJobをRedisから削除します。
//
// args:
//
//	ctx: コンテキスト
//	job: 削除対象のCrawlJob
//
// return:
//
//	error: 削除に失敗した場合のエラー
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

// FindListByStatusは、指定したステータスのCrawlJobをRedisから取得します。
//
// args:
//
//	ctx: コンテキスト
//	size: 取得する件数
//	status: 対象のジョブステータス
//
// return:
//
//	[]model.CrawlJob: 取得したジョブのリスト
//	error: 取得に失敗した場合のエラー
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

// Existsは、指定したCrawlJobがRedisに存在するか確認します。
//
// args:
//
//	ctx: コンテキスト
//	job: 存在確認するCrawlJob
//
// return:
//
//	bool: 存在する場合はtrue
//	error: 確認に失敗した場合のエラー
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

// generateJobKeyは、ジョブのステータスに応じたRedisキーを生成します。
//
// args:
//
//	job: 対象のCrawlJob
//
// return:
//
//	string: 生成されたキー
//	error: 生成に失敗した場合のエラー
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

// generateSuccessJobKeyは、成功ジョブ用のRedisキーを生成します。
//
// args:
//
//	url: 対象URL
//
// return:
//
//	string: 生成されたキー
func (r *crawlJobClient) generateSuccessJobKey(url string) string {
	return fmt.Sprintf("success_job: %s", url)
}

// generateFailedJobKeyは、失敗ジョブ用のRedisキーを生成します。
//
// args:
//
//	url: 対象URL
//
// return:
//
//	string: 生成されたキー
func (r *crawlJobClient) generateFailedJobKey(url string) string {
	return fmt.Sprintf("failed_job: %s", url)
}

// generatePendingJobKeyは、保留ジョブ用のRedisキーを生成します。
//
// args:
//
//	url: 対象URL
//
// return:
//
//	string: 生成されたキー
func (r *crawlJobClient) generatePendingJobKey(url string) string {
	return fmt.Sprintf("pending_job:%s", url)
}
