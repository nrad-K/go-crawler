package model

import (
	"net/url"

	"github.com/google/uuid"
)

type CrawlJobStatus string

const (
	CrawlJobStatusPending CrawlJobStatus = "PENDING"
	CrawlJobStatusSuccess CrawlJobStatus = "SUCCESS"
	CrawlJobStatusFailed  CrawlJobStatus = "FAILED"
)

type CrawlJob struct {
	ID     uuid.UUID
	URL    url.URL
	Status CrawlJobStatus
}
