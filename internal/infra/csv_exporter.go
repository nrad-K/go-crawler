package infra

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nrad-K/go-crawler/internal/domain/model"
)

type FileExporter interface {
	Write(jobPosting model.JobPosting) error
	Close() error
}

type CSVExporter struct {
	file   *os.File
	writer *csv.Writer
}

func formatUint(p *uint) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

func formatUint64(p *uint64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

func NewCSVExporter(filePath string, headers []string) (*CSVExporter, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("出力ディレクトリの作成に失敗しました: %w", err)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("CSVファイルの作成に失敗しました: %w", err)
	}

	writer := csv.NewWriter(file)

	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("CSVヘッダーの書き込みに失敗しました: %w", err)
	}

	return &CSVExporter{
		file:   file,
		writer: writer,
	}, nil
}

func (c *CSVExporter) Write(job model.JobPosting) error {

	row := []string{
		job.CompanyName(),
		job.Title(),
		job.SummaryURL(),
		string(job.Location().PrefectureCode()),
		job.Location().PrefectureName(),
		job.Location().City(),
		job.Location().Raw(),
		string(job.Headquarters().PrefectureCode()),
		job.Headquarters().PrefectureName(),
		job.Headquarters().City(),
		job.Headquarters().Raw(),
		string(job.JobType()),
		fmt.Sprintf("%d", job.Salary().MinAmount()),
		formatUint64(job.Salary().MaxAmount()),
		string(job.Salary().Unit()),
		job.PostedAt().Format("2006-01-02"),
		job.Details().JobName(),
		fmt.Sprintf("%d", job.Details().Raise()),
		fmt.Sprintf("%d", job.Details().Bonus()),
		job.Details().Description(),
		job.Details().Requirements(),
		string(job.Details().WorkplaceType()),
		formatUint(job.Details().HolidaysPerYear()),
		string(job.Details().HolidayPolicy()),
		job.Details().WorkHours(),
		job.Details().Benefits().RawBenefits(),
	}

	return c.writer.Write(row)
}

func (c *CSVExporter) Close() error {
	c.writer.Flush()
	return c.file.Close()
}
