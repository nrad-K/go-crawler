package infra

import (
	"github.com/nrad-K/go-crawler/internal/domain/model"
)

type CrawlJobRecord struct {
	ID     string `json:"id"`
	URL    string `json:"url"`
	Status string `json:"status"`
}

func (c *CrawlJobRecord) ToDomain() (model.CrawlJob, error) {
	crawlJob, err := model.Reconstruct(c.ID, c.URL, c.Status)
	if err != nil {
		return model.CrawlJob{}, err
	}

	return crawlJob, nil
}

func ToRecord(crawlJob model.CrawlJob) CrawlJobRecord {
	return CrawlJobRecord{
		ID:     crawlJob.ID(),
		URL:    crawlJob.URL(),
		Status: string(crawlJob.Status()),
	}
}
