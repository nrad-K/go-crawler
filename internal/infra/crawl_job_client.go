package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nrad-K/go-crawler/internal/domain/model"
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
func NewCrawlJobClient(rds *redis.Client) *crawlJobClient {
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
	// ジョブをJSONにマーシャルする
	record := ToRecord(job)

	data, err := json.Marshal(record)
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

// FindListByStatusStreamは、指定したステータスのCrawlJobをRedisからストリーム形式で取得します。
//
// args:
//
//	ctx: コンテキスト
//	size: 1回のSCANで取得するキーの数
//	status: 対象のジョブステータス
//
// return:
//
//	<-chan model.CrawlJobStream: 取得したジョブのストリーム
func (r *crawlJobClient) FindListByStatusStream(ctx context.Context, size int, status model.CrawlJobStatus) <-chan model.CrawlJobStream {
	batchSize := int64(size)
	resultCh := make(chan model.CrawlJobStream, batchSize)

	go func() {
		defer close(resultCh)

		var cursor uint64 = 0
		pattern, err := r.getJobKeyPattern(status)
		if err != nil {
			resultCh <- model.CrawlJobStream{
				Err: fmt.Errorf("ジョブキーのパターンの取得に失敗しました: %w", err),
			}
			return
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// SCANでキーを取得
			keys, nextCursor, err := r.redis.Scan(ctx, cursor, pattern, batchSize).Result()
			if err != nil {
				resultCh <- model.CrawlJobStream{
					Err: fmt.Errorf("Redis SCANエラー: %w", err),
				}
				return
			}

			for _, key := range keys {
				select {
				case <-ctx.Done():
					return
				default:
				}

				value, err := r.redis.Get(ctx, key).Result()
				if err != nil {
					resultCh <- model.CrawlJobStream{
						Err: fmt.Errorf("キー %s のRedis取得エラー: %w", key, err),
					}
					continue
				}

				jobRecord := CrawlJobRecord{}
				err = json.Unmarshal([]byte(value), &jobRecord)
				if err != nil {
					resultCh <- model.CrawlJobStream{
						Err: fmt.Errorf("キー %s のJSONデシリアライズに失敗しました: %w", key, err),
					}
					continue
				}

				job, err := jobRecord.ToDomain()
				if err != nil {
					resultCh <- model.CrawlJobStream{
						Err: fmt.Errorf("ジョブデータのドメイン変換に失敗しました（キー: %s, エラー: %v）", key, err),
					}
					continue
				}

				resultCh <- model.CrawlJobStream{
					Job: job,
					Err: nil,
				}
			}

			// カーソルが0になったら終了
			if nextCursor == 0 {
				break
			}
			cursor = nextCursor
		}
	}()

	return resultCh
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

// getJobKeyPatternは、指定されたジョブステータスに対応するRedisキーのパターンを生成します。
//
// args:
//
//	status: パターンを生成する対象のジョブステータス
//
// return:
//
//	string: 生成されたキーパターン
//	error: サポートされていないステータスが指定された場合のエラー
func (r *crawlJobClient) getJobKeyPattern(status model.CrawlJobStatus) (string, error) {
	pattern := ""
	switch status {
	case model.CrawlJobStatusSuccess:
		pattern = "success_job:*"
	case model.CrawlJobStatusFailed:
		pattern = "failed_job:*"
	case model.CrawlJobStatusPending:
		pattern = "pending_job:*"
	default:
		return pattern, fmt.Errorf("サポートされていないジョブステータスです: %s", status)
	}

	return pattern, nil
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

	switch job.Status() {

	case model.CrawlJobStatusPending:
		key = r.generatePendingJobKey(job.URL())

	case model.CrawlJobStatusSuccess:
		key = r.generateSuccessJobKey(job.URL())

	case model.CrawlJobStatusFailed:
		key = r.generateFailedJobKey(job.URL())

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
