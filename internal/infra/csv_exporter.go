package infra

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nrad-K/go-crawler/internal/domain/model"
)

// FileExporterは、求人情報をファイルにエクスポートするためのインターフェースです。
type FileExporter interface {
	// Writeは、単一の求人情報を書き込みます。
	Write(jobPosting model.JobPosting) error
	// Closeは、エクスポーターをクローズし、リソースを解放します。
	Close() error
}

// CSVExporterは、求人情報をCSVファイルにエクスポートするFileExporterの実装です。
//
// フィールド:
//
//	file   : 書き込み対象の*os.File
//	writer : CSV書き込みを行う*csv.Writer
type CSVExporter struct {
	file   *os.File
	writer *csv.Writer
}

// formatUintは、*uint型の値をフォーマットします。ポインタがnilの場合は空文字列を返します。
func formatUint(p *uint) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

// formatUint64は、*uint64型の値をフォーマットします。ポインタがnilの場合は空文字列を返します。
func formatUint64(p *uint64) string {
	if p == nil {
		return ""
	}
	return fmt.Sprintf("%d", *p)
}

// NewCSVExporterは、CSVExporterの新しいインスタンスを生成します。
// 指定されたファイルパスにCSVファイルを作成し、ヘッダーを書き込みます。
//
// args:
//
//	filePath : 出力するCSVファイルのパス
//	headers  : CSVファイルのヘッダー行
//
// return:
//
//	*CSVExporter : 生成されたCSVExporterのインスタンス
//	error        : ディレクトリやファイルの作成、ヘッダーの書き込みに失敗した場合のエラー
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

// Writeは、1件の求人情報をCSVファイルに書き込みます。
//
// args:
//
//	job : 書き込む対象のmodel.JobPosting
//
// return:
//
//	error : CSV行の書き込みに失敗した場合のエラー
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
		formatUint(job.Details().Raise()),
		formatUint(job.Details().Bonus()),
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

// Closeは、CSVライターをフラッシュし、ファイルをクローズします。
//
// return:
//
//	error : ファイルのクローズに失敗した場合のエラー
func (c *CSVExporter) Close() error {
	c.writer.Flush()
	return c.file.Close()
}
