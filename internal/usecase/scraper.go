package usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
)

// ScraperArgsは、スクレイパーユースケースを構築するための引数を保持します。
//
// フィールド:
//
//	Loader   : HTMLファイルのローダー
//	Document : HTMLドキュメントのパーサー
//	Exporter : ファイルエクスポーター
//	Cfg      : スクレイパーの設定情報
//	Parser   : 求人情報のパーサー
//	Logger   : ロガー
type ScraperArgs struct {
	Loader   infra.HTMLFileLoader
	Document infra.HTMLDocument
	Exporter infra.FileExporter
	Cfg      config.ScraperConfig
	Parser   infra.JobPostingParser
	Logger   logger.AppLogger
}

// saveJobPostingFromHTMLUseCaseは、HTMLファイルから求人情報を抽出し、保存するユースケースです。
type saveJobPostingFromHTMLUseCase struct {
	loader   infra.HTMLFileLoader
	document infra.HTMLDocument
	exporter infra.FileExporter
	cfg      config.ScraperConfig
	parser   infra.JobPostingParser
	logger   logger.AppLogger
}

// NewSaveJobPostingFromHTMLUseCaseは、saveJobPostingFromHTMLUseCaseの新しいインスタンスを生成します。
//
// args:
//
//	args : ScraperArgs構造体（ローダー、パーサー、エクスポーター、設定、ロガーなど）
//
// return:
//
//	*saveJobPostingFromHTMLUseCase : 生成されたユースケースインスタンス
func NewSaveJobPostingFromHTMLUseCase(args ScraperArgs) *saveJobPostingFromHTMLUseCase {
	return &saveJobPostingFromHTMLUseCase{
		loader:   args.Loader,
		document: args.Document,
		exporter: args.Exporter,
		cfg:      args.Cfg,
		parser:   args.Parser,
		logger:   args.Logger,
	}
}

// SaveJobPostingCSVは、指定されたディレクトリからHTMLファイルを読み込み、
// 求人情報を抽出してCSVファイルに保存するメインの処理です。
//
// args:
//
//	ctx : コンテキスト
//
// return:
//
//	error : 処理中に発生したエラー
func (u *saveJobPostingFromHTMLUseCase) SaveJobPostingCSV(ctx context.Context) error {
	u.logger.Info("HTMLファイルパスの一覧を取得します...")
	dirpaths, err := u.loader.ListHTMLFilePaths(u.cfg.HtmlDir)
	if err != nil {
		u.logger.Error("HTMLファイルの一覧取得に失敗しました", "error", err)
		return fmt.Errorf("HTMLファイルの一覧取得に失敗しました: %w", err)
	}

	jobs := make(chan string, len(dirpaths))
	jobPosting := make(chan model.JobPosting, len(dirpaths))
	var wg sync.WaitGroup
	maxWorkers := 3

	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u.worker(ctx, jobs, jobPosting)
		}()
	}

	for _, path := range dirpaths {
		jobs <- path
	}
	close(jobs)

	wg.Wait()
	close(jobPosting)

	writtenCount := 0
	for post := range jobPosting {
		if err := u.exporter.Write(post); err != nil {
			u.logger.Error("求人情報の書き込みに失敗しました", "error", err)
			continue
		}
		writtenCount++
		if writtenCount%100 == 0 {
			u.logger.Info("求人情報を書き込みました。", "count", writtenCount)
		}
	}

	if err := u.exporter.Close(); err != nil {
		u.logger.Error("exporterのクローズに失敗しました", "error", err)
		return fmt.Errorf("exporterのクローズに失敗しました: %w", err)
	}

	u.logger.Info("スクレイピング処理が完了しました。", "total_count", writtenCount)
	return nil
}

// workerは、ファイルパスを受け取って処理し、結果をチャネルに送信するワーカー関数です。
//
// args:
//
//	ctx     : コンテキスト
//	jobs    : 処理対象のファイルパスを受信するチャネル
//	results : 処理結果の求人情報を送信するチャネル
func (u *saveJobPostingFromHTMLUseCase) worker(ctx context.Context, jobs <-chan string, results chan<- model.JobPosting) {
	for path := range jobs {
		select {

		case <-ctx.Done():
			return

		default:
			extractJobPosting, err := u.processFile(path)
			if err != nil {
				u.logger.Error("求人情報の処理に失敗しました", "path", path, "error", err)
				continue
			}

			select {
			case results <- extractJobPosting:
			case <-ctx.Done():
				return
			}
		}
	}
}

// processFileは、単一のHTMLファイルを処理し、求人情報を抽出します。
//
// args:
//
//	path : 処理対象のHTMLファイルのパス
//
// return:
//
//	model.JobPosting : 抽出された求人情報
//	error            : ファイルの読み込みや処理中に発生したエラー
func (u *saveJobPostingFromHTMLUseCase) processFile(path string) (model.JobPosting, error) {
	htmlContent, err := u.loader.LoadHTMLFile(path)
	if err != nil {
		return model.JobPosting{}, fmt.Errorf("HTMLファイルの読み込みに失敗しました: %w", err)
	}

	extractJobPosting := u.extractJobPosting(htmlContent)
	return extractJobPosting, nil
}

// extractJobPostingは、HTMLコンテンツから求人情報の詳細を抽出し、JobPostingオブジェクトを生成します。
//
// args:
//
//	htmlContent : 解析対象のHTMLコンテンツ
//
// return:
//
//	model.JobPosting : 抽出された情報を持つJobPostingオブジェクト
func (u *saveJobPostingFromHTMLUseCase) extractJobPosting(htmlContent string) model.JobPosting {
	var args model.JobPostingArgs
	// タイトルを抽出
	extractedTitles, err := u.extractValues(htmlContent, u.cfg.Title)
	if err != nil {
		u.logger.Warn("タイトルの抽出に失敗しました", "error", err)
	}
	if len(extractedTitles) > 0 {
		args.Title = extractedTitles[0]
	}

	// Locationを抽出
	extractedLocation, err := u.extractValues(htmlContent, u.cfg.Location)
	if err != nil {
		u.logger.Warn("勤務地の抽出に失敗しました", "error", err)
	}
	if len(extractedLocation) > 0 {
		location, err := u.parser.ParseLocation(extractedLocation[0])
		if err != nil {
			u.logger.Warn("勤務地のパースに失敗しました", "error", err)
		}

		args.Location = location
	}

	// Headquarters（本社所在地）の抽出
	extractedHeadquarters, err := u.extractValues(htmlContent, u.cfg.Headquarters)
	if err != nil {
		u.logger.Warn("本社所在地の抽出に失敗しました", "error", err)
	}
	if len(extractedHeadquarters) > 0 {
		headquarters, err := u.parser.ParseLocation(extractedHeadquarters[0])
		if err != nil {
			u.logger.Warn("本社所在地のパースに失敗しました", "error", err)
		}

		args.Headquarters = headquarters
	}

	// 会社名を抽出
	extractedCompanyNames, err := u.extractValues(htmlContent, u.cfg.CompanyName)
	if err != nil {
		u.logger.Warn("会社名の抽出に失敗しました", "error", err)
	}
	if len(extractedCompanyNames) > 0 {
		args.CompanyName = extractedCompanyNames[0]
	}

	// 概要URLを抽出
	extractedSummaryURLs, err := u.extractValues(htmlContent, u.cfg.SummaryURL)
	if err != nil {
		u.logger.Warn("概要URLの抽出に失敗しました", "error", err)
	}
	if len(extractedSummaryURLs) > 0 {
		args.SummaryURL = extractedSummaryURLs[0]
	}

	// JobTypeを抽出
	extractedJobTypesStr, err := u.extractValues(htmlContent, u.cfg.JobType)
	if err != nil {
		u.logger.Warn("JobTypeの抽出に失敗しました", "error", err)
	}
	if len(extractedJobTypesStr) > 0 {
		args.JobType = u.parser.ParseJobType(extractedJobTypesStr[0])
	}

	// Salaryを抽出
	var salaryStr string
	extractedSalaryStrs, err := u.document.ExtractText(htmlContent, u.cfg.Salary.Selector)
	if err != nil {
		u.logger.Warn("給与情報の抽出に失敗しました", "error", err)
	}
	if len(extractedSalaryStrs) > 0 {
		salaryStr = extractedSalaryStrs[0]
	}

	salary, err := u.parser.ParseSalaryDetails(salaryStr)
	// 空文字列のパースエラーはログに出さない
	if err != nil && salaryStr != "" {
		u.logger.Warn("給与情報のパースに失敗しました", "error", err)
	}
	args.Salary = salary

	// PostedAtを抽出
	extractedPostedAtStr, err := u.extractValues(htmlContent, u.cfg.PostedAt)
	if err != nil {
		u.logger.Warn("PostedAtの抽出に失敗しました", "error", err)
	}
	if len(extractedPostedAtStr) > 0 {
		parsedTime, err := u.parser.ParsePostedAt(extractedPostedAtStr[0])
		if err != nil {
			u.logger.Warn("PostedAtのパースに失敗しました", "error", err)
		}
		args.PostedAt = parsedTime
	}

	// Detailsを抽出
	var details model.JobPostingDetailArgs

	// JobName
	extractedJobName, err := u.extractValues(htmlContent, u.cfg.Details.JobName)
	if err != nil {
		u.logger.Warn("職種名の抽出に失敗しました", "error", err)
	}
	if len(extractedJobName) > 0 {
		details.JobName = extractedJobName[0]
	}

	// Description
	extractedDescription, err := u.extractValues(htmlContent, u.cfg.Details.Description)
	if err != nil {
		u.logger.Warn("募集要項の抽出に失敗しました", "error", err)
	}
	if len(extractedDescription) > 0 {
		details.Description = extractedDescription[0]
	}

	// Requirements
	extractedRequirements, err := u.extractValues(htmlContent, u.cfg.Details.Requirements)
	if err != nil {
		u.logger.Warn("応募資格・条件の抽出に失敗しました", "error", err)
	}
	if len(extractedRequirements) > 0 {
		details.Requirements = extractedRequirements[0]
	}

	// WorkHours
	extractedWorkHours, err := u.extractValues(htmlContent, u.cfg.Details.WorkHours)
	if err != nil {
		u.logger.Warn("勤務時間の抽出に失敗しました", "error", err)
	}
	if len(extractedWorkHours) > 0 {
		details.WorkHours = extractedWorkHours[0]
	}

	// WorkplaceType
	extractedWorkplaceType, err := u.extractValues(htmlContent, u.cfg.Details.WorkplaceType)
	if err != nil {
		u.logger.Warn("勤務地タイプ情報の抽出に失敗しました", "error", err)
	}
	if len(extractedWorkplaceType) > 0 {
		details.WorkplaceType = u.parser.ParseWorkplaceType(extractedWorkplaceType[0])
	}

	// Benefits
	extractedBenefits, err := u.extractValues(htmlContent, u.cfg.Details.Benefits)
	if err != nil {
		u.logger.Warn("福利厚生の抽出に失敗しました", "error", err)
	}
	if len(extractedBenefits) > 0 {
		details.Benefits = u.parser.ParseBenefits(extractedBenefits[0])
	}

	// Raise
	extractedRaise, err := u.extractValues(htmlContent, u.cfg.Details.Raise)
	if err != nil {
		u.logger.Warn("昇給情報の抽出に失敗しました", "error", err)
	}
	if len(extractedRaise) > 0 {
		parsedRaise := u.parser.ParseRaise(extractedRaise[0])
		details.Raise = parsedRaise
	}

	// Bonus
	extractedBonus, err := u.extractValues(htmlContent, u.cfg.Details.Bonus)
	if err != nil {
		u.logger.Warn("賞与情報の抽出に失敗しました", "error", err)
	}
	if len(extractedBonus) > 0 {
		parsedBonus := u.parser.ParseBonus(extractedBonus[0])
		details.Bonus = parsedBonus
	}

	// HolidaysPerYear
	extractedHolidaysPerYear, err := u.extractValues(htmlContent, u.cfg.Details.HolidaysPerYear)
	if err != nil {
		u.logger.Warn("年間休日数の抽出に失敗しました", "error", err)
	}
	if len(extractedHolidaysPerYear) > 0 {
		parsedHolidaysPerYear, err := u.parser.ParseOptionalUint(extractedHolidaysPerYear[0])
		if err != nil {
			u.logger.Warn("年間休日数のパースに失敗しました", "error", err)
		}
		details.HolidaysPerYear = parsedHolidaysPerYear
	}

	// HolidayPolicy
	extractedHolidayPolicy, err := u.extractValues(htmlContent, u.cfg.Details.HolidayPolicy)
	if err != nil {
		u.logger.Warn("休日休暇ポリシーの抽出に失敗しました", "error", err)
	}
	if len(extractedHolidayPolicy) > 0 {
		details.HolidayPolicy = u.parser.ParseHolidayPolicy(extractedHolidayPolicy[0])
	}
	extractDetails := model.NewJobPostingDetail(details)
	args.Details = extractDetails

	// JobPostingを生成して返す
	return model.NewJobPosting(args)
}

// extractValuesは、SelectorConfigに基づいてHTMLから値を抽出します。
// 属性、正規表現、またはテキストの抽出をセレクター設定に応じて行います。
//
// args:
//
//	htmlContent : 解析対象のHTMLコンテンツ
//	cfg         : 使用するセレクター設定
//
// return:
//
//	[]string : 抽出された文字列のスライス
//	error    : 抽出処理中に発生したエラー
func (u *saveJobPostingFromHTMLUseCase) extractValues(htmlContent string, cfg config.SelectorConfig) ([]string, error) {
	var extracted []string
	var err error

	if cfg.Attr != "" {
		extracted, err = u.document.ExtractAttribute(htmlContent, cfg.Selector, cfg.Attr)
		return extracted, err
	}

	if cfg.Regex != "" {
		extracted, err = u.document.ExtractTextByRegex(htmlContent, cfg.Selector, cfg.Regex)
		return extracted, err
	}

	extracted, err = u.document.ExtractText(htmlContent, cfg.Selector)
	return extracted, err
}
