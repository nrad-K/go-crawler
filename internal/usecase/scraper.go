package usecase

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/domain/repository"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
)

type ScraperUseCase interface {
	Execute(ctx context.Context) error
}

type ScraperArgs struct {
	loader   infra.HTMLFileLoader
	document infra.HTMLDocument
	repo     repository.JobPostingRepository
	cfg      config.ScraperConfig
	parser   infra.JobPostingParser
	logger   logger.AppLogger
}

type saveJobPostingFromHTMLUseCase struct {
	loader   infra.HTMLFileLoader
	document infra.HTMLDocument
	repo     repository.JobPostingRepository
	cfg      config.ScraperConfig
	parser   infra.JobPostingParser
	logger   logger.AppLogger
}

func NewSaveJobPostingFromHTMLUseCase(args ScraperArgs) ScraperUseCase {
	return &saveJobPostingFromHTMLUseCase{
		loader:   args.loader,
		document: args.document,
		repo:     args.repo,
		cfg:      args.cfg,
		parser:   args.parser,
		logger:   args.logger,
	}
}

func (u *saveJobPostingFromHTMLUseCase) Execute(ctx context.Context) error {
	u.logger.Info("HTMLファイルパスの一覧を取得します...")
	dirpaths, err := u.loader.ListHTMLFilePaths(u.cfg.DirPath)
	if err != nil {
		u.logger.Error("HTMLファイルの一覧取得に失敗しました: %v", err)
		return fmt.Errorf("HTMLファイルの一覧取得に失敗しました: %w", err)
	}

	for _, path := range dirpaths {
		u.logger.Info("HTMLファイルを処理中... Path: %s", path)
		htmlContent, err := u.loader.LoadHTMLFile(path)
		if err != nil {
			u.logger.Error("HTMLファイルの読み込みに失敗しました. Path: %s, Error: %v", path, err)
			return fmt.Errorf("HTMLファイルの読み込みに失敗しました: %w", err)
		}

		var jobPosting model.JobPosting
		jobPosting.ID = uuid.New()

		if err := u.extractJobPosting(htmlContent, &jobPosting); err != nil {
			u.logger.Error("求人情報の抽出に失敗しました. Path: %s, Error: %v", path, err)
			return err
		}

		if err := u.repo.Save(ctx, jobPosting); err != nil {
			u.logger.Error("求人情報のDB保存に失敗しました. ID: %s, Error: %v", jobPosting.ID, err)
			return fmt.Errorf("求人情報のDB保存に失敗しました: %w", err)
		}
		u.logger.Info("求人情報を保存しました. ID: %s, CompanyName: %s", jobPosting.ID, jobPosting.CompanyName)
	}

	u.logger.Info("スクレイピング処理が完了しました。")
	return nil
}

func (u *saveJobPostingFromHTMLUseCase) extractJobPosting(htmlContent string, jobPosting *model.JobPosting) error {
	// Locationを抽出
	extractedLocation, err := u.extractValues(htmlContent, u.cfg.Location)
	if err != nil {
		return fmt.Errorf("勤務地の抽出に失敗しました: %w", err)
	}
	if len(extractedLocation) > 0 {
		jobPosting.Location = u.parser.ParseLocation(extractedLocation[0])
	}

	// Headquarters
	extractedHeadquarters, err := u.extractValues(htmlContent, u.cfg.Headquarters)
	if err != nil {
		return fmt.Errorf("本社所在地の抽出に失敗しました: %w", err)
	}
	if len(extractedHeadquarters) > 0 {
		jobPosting.Headquarters = u.parser.ParseLocation(extractedHeadquarters[0])
	}

	// 会社名を抽出
	extractedCompanyNames, err := u.extractValues(htmlContent, u.cfg.CompanyName)
	if err != nil {
		return fmt.Errorf("会社名の抽出に失敗しました: %w", err)
	}
	if len(extractedCompanyNames) > 0 {
		jobPosting.CompanyName = extractedCompanyNames[0]
	}

	// 概要URLを抽出
	extractedSummaryURLs, err := u.extractValues(htmlContent, u.cfg.SummaryURL)
	if err != nil {
		return fmt.Errorf("概要URLの抽出に失敗しました: %w", err)
	}
	if len(extractedSummaryURLs) > 0 {
		jobPosting.SummaryURL = extractedSummaryURLs[0]
	}

	// JobTypeを抽出
	extractedJobTypesStr, err := u.extractValues(htmlContent, u.cfg.JobType)
	if err != nil {
		return fmt.Errorf("JobTypeの抽出に失敗しました: %w", err)
	}
	if len(extractedJobTypesStr) > 0 {
		jobPosting.JobType = u.parser.ParseJobType(extractedJobTypesStr[0])
	}

	// Salaryを抽出
	extractedSalaryStr, err := u.document.ExtractText(htmlContent, u.cfg.Salary.Selector)
	if err != nil {
		return fmt.Errorf("給与情報の抽出に失敗しました: %w", err)
	}
	if len(extractedSalaryStr) > 0 {
		salary, err := u.parser.ParseSalaryDetails(extractedSalaryStr[0])
		if err != nil {
			return fmt.Errorf("給与情報のパースに失敗しました: %w", err)
		}
		jobPosting.Salary = salary
	}

	// PostedAtを抽出
	extractedPostedAtStr, err := u.extractValues(htmlContent, u.cfg.PostedAt)
	if err != nil {
		return fmt.Errorf("PostedAtの抽出に失敗しました: %w", err)
	}
	if len(extractedPostedAtStr) > 0 {
		parsedTime, err := u.parser.ParsePostedAt(extractedPostedAtStr[0])
		if err != nil {
			return fmt.Errorf("PostedAtのパースに失敗しました: %w", err)
		}
		jobPosting.PostedAt = parsedTime
	}

	// Detailsを抽出
	var details model.JobPostingDetail

	// JobName
	extractedJobName, err := u.extractValues(htmlContent, u.cfg.Details.JobName)
	if err != nil {
		return fmt.Errorf("職種名の抽出に失敗しました: %w", err)
	}
	if len(extractedJobName) > 0 {
		details.JobName = extractedJobName[0]
	}

	// Description
	extractedDescription, err := u.extractValues(htmlContent, u.cfg.Details.Description)
	if err != nil {
		return fmt.Errorf("募集要項の抽出に失敗しました: %w", err)
	}
	if len(extractedDescription) > 0 {
		details.Description = extractedDescription[0]
	}

	// Requirements
	extractedRequirements, err := u.extractValues(htmlContent, u.cfg.Details.Requirements)
	if err != nil {
		return fmt.Errorf("応募資格・条件の抽出に失敗しました: %w", err)
	}
	if len(extractedRequirements) > 0 {
		details.Requirements = extractedRequirements[0]
	}

	// WorkHours
	extractedWorkHours, err := u.extractValues(htmlContent, u.cfg.Details.WorkHours)
	if err != nil {
		return fmt.Errorf("勤務時間の抽出に失敗しました: %w", err)
	}
	if len(extractedWorkHours) > 0 {
		details.WorkHours = extractedWorkHours[0]
	}

	// WorkplaceType
	extractedWorkplaceType, err := u.extractValues(htmlContent, u.cfg.Details.WorkplaceType)
	if err != nil {
		return fmt.Errorf("勤務地タイプ情報の抽出に失敗しました: %w", err)
	}
	if len(extractedWorkplaceType) > 0 {
		details.WorkplaceType = u.parser.ParseWorkplaceType(extractedWorkplaceType[0])
	}

	// Benefits
	extractedBenefits, err := u.extractValues(htmlContent, u.cfg.Details.Benefits)
	if err != nil {
		return fmt.Errorf("福利厚生の抽出に失敗しました: %w", err)
	}
	if len(extractedBenefits) > 0 {
		details.Benefits = u.parser.ParseBenefits(extractedBenefits[0])
	}

	// Raise
	extractedRaise, err := u.extractValues(htmlContent, u.cfg.Details.Raise)
	if err != nil {
		return fmt.Errorf("昇給情報の抽出に失敗しました: %w", err)
	}
	if len(extractedRaise) > 0 {
		parsedRaise := u.parser.ParseRaise(extractedRaise[0])
		details.Raise = parsedRaise
	}

	// Bonus
	extractedBonus, err := u.extractValues(htmlContent, u.cfg.Details.Bonus)
	if err != nil {
		return fmt.Errorf("賞与情報の抽出に失敗しました: %w", err)
	}
	if len(extractedBonus) > 0 {
		parsedBonus := u.parser.ParseBonus(extractedBonus[0])
		details.Bonus = parsedBonus
	}

	// HolidaysPerYear
	extractedHolidaysPerYear, err := u.extractValues(htmlContent, u.cfg.Details.HolidaysPerYear)
	if err != nil {
		return fmt.Errorf("年間休日数の抽出に失敗しました: %w", err)
	}
	if len(extractedHolidaysPerYear) > 0 {
		parsedHolidaysPerYear, err := u.parser.ParseOptionalUint(extractedHolidaysPerYear[0])
		if err != nil {
			return fmt.Errorf("年間休日数のパースに失敗しました: %w", err)
		}
		details.HolidaysPerYear = parsedHolidaysPerYear
	}

	// HolidayPolicy
	extractedHolidayPolicy, err := u.extractValues(htmlContent, u.cfg.Details.HolidayPolicy)
	if err != nil {
		return fmt.Errorf("休日休暇ポリシーの抽出に失敗しました: %w", err)
	}
	if len(extractedHolidayPolicy) > 0 {
		details.HolidayPolicy = u.parser.ParseHolidayPolicy(extractedHolidayPolicy[0])
	}
	jobPosting.Details = details

	return nil
}

// extractValues はSelectorConfigに基づいてHTMLから値を抽出します。
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
