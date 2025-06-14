package repository

import (
	"context"

	"github.com/nrad-K/go-crawler/internal/domain/model"
)

type JobPostingRepository interface {
	Save(ctx context.Context, job model.JobPosting) error
}
