package infra

import (
	"context"
	"database/sql"

	"github.com/nrad-K/go-crawler/internal/db"
	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/domain/repository"
)

type JobPostingQuery interface {
	CreateJobPosting(ctx context.Context, job db.CreateJobPostingParams) error
	GetJobPostingByID(ctx context.Context, id string) (db.JobPosting, error)
}

type jobPositingClient struct {
	db JobPostingQuery
}

func NewJobPostingClient() repository.JobPostingRepository {
	return &jobPositingClient{}
}

func (j *jobPositingClient) Save(ctx context.Context, job model.JobPosting) error {
	arg := db.CreateJobPostingParams{
		ID:              job.ID,
		Title:           job.Title,
		CompanyName:     job.CompanyName,
		PrefectureCode:  job.Location.PrefectureCode,
		PrefectureName:  job.Location.PrefectureName,
		Municipality:    job.Location.Municipality,
		SummaryUrl:      job.SummaryURL,
		JobType:         db.JobType(job.JobType.String()),
		SalaryMinAmount: int64(job.Salary.MinAmount),
		SalaryMaxAmount: int64(job.Salary.MaxAmount),
		SalaryUnit:      db.SalaryType(job.Salary.Unit.String()),
		SalaryCurrency:  db.Currency(job.Salary.Currency),
		SalaryIsFixed:   job.Salary.IsFixed,
		PostedAt:        job.PostedAt,
		JobName:         job.Details.JobName,
		HolidayPolicy:   job.Details.HolidayPolicy,
		Raise:           toNullInt32(job.Details.Raise),
		Bonus:           toNullInt32(job.Details.Bonus),
		Description:     job.Details.Description,
		Requirements:    job.Details.Requirements,
		HolidaysPerYear: toNullInt32(job.Details.HolidaysPerYear),
		WorkHours:       job.Details.WorkHours,
		Benefits:        job.Details.Benefits,
	}

	return j.db.CreateJobPosting(ctx, arg)

}

func (j *jobPositingClient) FindByID(ctx context.Context, id string) (model.JobPosting, error) {
	job, err := j.db.GetJobPostingByID(ctx, id)
	if err != nil {
		return model.JobPosting{}, err
	}

	return model.JobPosting{
		ID:          job.ID,
		Title:       job.Title,
		CompanyName: job.CompanyName,
		Location: model.Location{
			PrefectureCode: job.PrefectureCode,
			PrefectureName: job.PrefectureName,
			Municipality:   job.Municipality,
		},
		SummaryURL: job.SummaryUrl,
		JobType:    toModelJobType(job.JobType),
		Salary: model.Salary{
			MinAmount: uint64(job.SalaryMinAmount),
			MaxAmount: uint64(job.SalaryMaxAmount),
			Unit:      toModelSalaryType(job.SalaryUnit),
			Currency:  model.Currency(job.SalaryCurrency),
			IsFixed:   job.SalaryIsFixed,
		},
		PostedAt: job.PostedAt,
		Details: model.JobPostingDetail{
			JobName:         job.JobName,
			HolidayPolicy:   job.HolidayPolicy,
			Raise:           fromNullInt32(job.Raise),
			Bonus:           fromNullInt32(job.Bonus),
			Description:     job.Description,
			Requirements:    job.Requirements,
			HolidaysPerYear: fromNullInt32(job.HolidaysPerYear),
			WorkHours:       job.WorkHours,
			Benefits:        job.Benefits,
		},
	}, nil
}

func toNullInt32(u *uint) sql.NullInt32 {
	if u == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(*u), Valid: true}
}

func fromNullInt32(n sql.NullInt32) *uint {
	if !n.Valid {
		return nil
	}
	val := uint(n.Int32)
	return &val
}

func toModelJobType(jt db.JobType) model.JobType {
	switch jt {
	case db.JobTypeFullTime:
		return model.FullTime
	case db.JobTypePartTime:
		return model.PartTime
	case db.JobTypeContract:
		return model.Contract
	case db.JobTypeTemporary:
		return model.Temporary
	case db.JobTypeFreelance:
		return model.Freelance
	case db.JobTypeInternship:
		return model.Internship
	case db.JobTypeOther:
		return model.Other
	default:
		return model.Unknown
	}
}

func toModelSalaryType(st db.SalaryType) model.SalaryType {
	switch st {
	case db.SalaryTypeHourly:
		return model.Hourly
	case db.SalaryTypeDaily:
		return model.Daily
	case db.SalaryTypeMonthly:
		return model.Monthly
	case db.SalaryTypeYearly:
		return model.Yearly
	default:
		return model.Yearly
	}
}
