package model

import (
	"errors"
	"net/url"

	"github.com/google/uuid"
)

type CrawlJobStatus string

const (
	CrawlJobStatusPending CrawlJobStatus = "PENDING"
	CrawlJobStatusSuccess CrawlJobStatus = "SUCCESS"
	CrawlJobStatusFailed  CrawlJobStatus = "FAILED"
)

type CrawlJobStream struct {
	Job CrawlJob
	Err error
}

type CrawlJob struct {
	id     uuid.UUID
	url    url.URL
	status CrawlJobStatus
}

func NewCrawlJob(rawURL string) (CrawlJob, error) {
	parseURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return CrawlJob{}, errors.New("不正なURLです")
	}

	return CrawlJob{
		id:     uuid.New(),
		url:    *parseURL,
		status: CrawlJobStatusPending,
	}, nil
}

func Reconstruct(id, rawURL, status string) (CrawlJob, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return CrawlJob{}, errors.New("不正なIDです")
	}

	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return CrawlJob{}, errors.New("不正なURLです")
	}

	var st CrawlJobStatus
	switch status {
	case string(CrawlJobStatusPending):
		st = CrawlJobStatusPending
	case string(CrawlJobStatusSuccess):
		st = CrawlJobStatusSuccess
	case string(CrawlJobStatusFailed):
		st = CrawlJobStatusFailed
	default:
		return CrawlJob{}, errors.New("無効なステータスです")
	}

	return CrawlJob{
		id:     uid,
		url:    *parsedURL,
		status: st,
	}, nil

}

func (c *CrawlJob) ChangeStatus(newStatus CrawlJobStatus) (CrawlJob, error) {
	switch newStatus {

	case CrawlJobStatusPending, CrawlJobStatusSuccess, CrawlJobStatusFailed:
		c.status = newStatus
		return CrawlJob{
			id:     c.id,
			url:    c.url,
			status: newStatus,
		}, nil

	default:
		return CrawlJob{}, errors.New("無効なステータスです")
	}
}

func (c *CrawlJob) ID() string {
	return c.id.String()
}

func (c *CrawlJob) URL() string {
	return c.url.String()
}

func (c *CrawlJob) Status() CrawlJobStatus {
	return c.status
}
