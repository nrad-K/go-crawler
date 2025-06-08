package repository

import (
	"context"

	"github.com/nrad-K/go-crawler/internal/domain/model"
)

type CrawlJobRepository interface {
	Save(ctx context.Context, job model.CrawlJob) error
	Delete(ctx context.Context, job model.CrawlJob) error
	FindListByStatus(ctx context.Context, size int, status model.CrawlJobStatus) ([]model.CrawlJob, error)
}
