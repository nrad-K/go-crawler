package infra

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/nrad-K/go-crawler/internal/db"
	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/domain/repository"
)

type JobPostingQuery interface {
	CreateJobPosting(ctx context.Context, job db.CreateJobPostingParams) error
	GetJobPostingByID(ctx context.Context, id uuid.UUID) (db.JobPosting, error)
	CreateCompany(ctx context.Context, arg db.CreateCompanyParams) (db.Company, error)
	GetCompanyByName(ctx context.Context, name string) (db.Company, error)
	CreateLocation(ctx context.Context, arg db.CreateLocationParams) (db.Location, error)
	GetLocationByPrefectureAndMunicipality(ctx context.Context, arg db.GetLocationByPrefectureAndMunicipalityParams) (db.Location, error)
	CreateJobBenefit(ctx context.Context, arg db.CreateJobBenefitsParams) error
}

type jobPositingClient struct {
	db JobPostingQuery
}

func NewJobPostingClient(db JobPostingQuery) repository.JobPostingRepository {
	return &jobPositingClient{db: db}
}

func (j *jobPositingClient) Save(ctx context.Context, job model.JobPosting) error {
	// 会社の情報を保存または取得
	company, err := j.db.GetCompanyByName(ctx, job.CompanyName)
	if err != nil {
		if err == sql.ErrNoRows {
			company, err = j.db.CreateCompany(ctx, db.CreateCompanyParams{
				Name:                       job.CompanyName,
				HeadquartersPrefectureCode: string(job.Headquarters.PrefectureCode),
				HeadquartersPrefectureName: job.Headquarters.PrefectureName,
				HeadquartersMunicipality:   job.Headquarters.City,
				HeadquartersRaw:            job.Headquarters.Raw,
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// ロケーションの情報を保存または取得
	location, err := j.db.GetLocationByPrefectureAndMunicipality(ctx, db.GetLocationByPrefectureAndMunicipalityParams{
		PrefectureCode: string(job.Location.PrefectureCode),
		Municipality:   job.Location.City,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			location, err = j.db.CreateLocation(ctx, db.CreateLocationParams{
				PrefectureCode: string(job.Location.PrefectureCode),
				PrefectureName: job.Location.PrefectureName,
				Municipality:   job.Location.City,
				RawLocation:    job.Location.Raw,
			})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	arg := db.CreateJobPostingParams{
		CompanyID:       company.ID,
		LocationID:      location.ID,
		Title:           job.Title,
		JobName:         job.Details.JobName,
		SummaryUrl:      job.SummaryURL,
		JobType:         toDBJobType(job.JobType),
		SalaryMinAmount: int64(job.Salary.MinAmount),
		SalaryMaxAmount: int64(job.Salary.MaxAmount),
		SalaryUnit:      toDBSalaryType(job.Salary.Unit),
		SalaryIsFixed:   job.Salary.IsFixed,
		Raise:           toNullInt32(job.Details.Raise),
		Bonus:           toNullInt32(job.Details.Bonus),
		Description:     job.Details.Description,
		Requirements:    job.Details.Requirements,
		WorkplaceType:   toDBWorkplaceType(job.Details.WorkplaceType),
		WorkHours:       job.Details.WorkHours,
		HolidayPolicy:   toDBHolidayPolicy(job.Details.HolidayPolicy),
		HolidaysPerYear: toNullInt32(job.Details.HolidaysPerYear),
		PostedAt:        job.PostedAt,
	}

	err = j.db.CreateJobPosting(ctx, arg)
	if err != nil {
		return err
	}

	// JobBenefit の保存
	benefitArg := db.CreateJobBenefitsParams{
		JobPostingID:         job.ID,
		SocialInsurance:      job.Details.Benefits.SocialInsurance,
		TransportAllowance:   job.Details.Benefits.TransportAllowance,
		HousingAllowance:     job.Details.Benefits.HousingAllowance,
		CompanyHousing:       job.Details.Benefits.CompanyHousing,
		RentSubsidy:          job.Details.Benefits.RentSubsidy,
		MealAllowance:        job.Details.Benefits.MealAllowance,
		CafeteriaProvided:    job.Details.Benefits.CafeteriaProvided,
		TrainingSupport:      job.Details.Benefits.TrainingSupport,
		CertificationSupport: job.Details.Benefits.CertificationSupport,
		PaidLeave:            job.Details.Benefits.PaidLeave,
		SpecialLeave:         job.Details.Benefits.SpecialLeave,
		FlexTime:             job.Details.Benefits.FlexTime,
		ShortWorkingHours:    job.Details.Benefits.ShortWorkingHours,
		ChildcareSupport:     job.Details.Benefits.ChildcareSupport,
		MaternityLeave:       job.Details.Benefits.MaternityLeave,
		ParentalLeave:        job.Details.Benefits.ParentalLeave,
		ElderCareSupport:     job.Details.Benefits.ElderCareSupport,
		RetirementPlan:       job.Details.Benefits.RetirementPlan,
		RawBenefits:          job.Details.Benefits.RawBenefits,
	}
	return j.db.CreateJobBenefit(ctx, benefitArg)

}

func toNullInt32(u *uint) sql.NullInt32 {
	if u == nil {
		return sql.NullInt32{Valid: false}
	}
	return sql.NullInt32{Int32: int32(*u), Valid: true}
}

func toDBJobType(jt model.JobType) db.JobType {
	switch jt {
	case model.FullTime:
		return db.JobTypeFullTime
	case model.PartTime:
		return db.JobTypePartTime
	case model.Contract:
		return db.JobTypeContract
	case model.Temporary:
		return db.JobTypeTemporary
	case model.Freelance:
		return db.JobTypeFreelance
	case model.Internship:
		return db.JobTypeInternship
	case model.Other:
		return db.JobTypeOther
	default:
		return db.JobTypeUnknown
	}
}

func toDBSalaryType(st model.SalaryType) db.SalaryType {
	switch st {
	case model.Hourly:
		return db.SalaryTypeHourly
	case model.Daily:
		return db.SalaryTypeDaily
	case model.Monthly:
		return db.SalaryTypeMonthly
	case model.Yearly:
		return db.SalaryTypeYearly
	default:
		return db.SalaryTypeYearly
	}
}

func toModelHolidayPolicy(hp db.HolidayPolicy) model.HolidayPolicy {
	switch hp {
	case db.HolidayPolicyCompleteTwoDaysAWeek:
		return model.CompleteTwoDaysAWeek
	case db.HolidayPolicyTwoDaysAWeek:
		return model.TwoDaysAWeek
	case db.HolidayPolicyOneDayAWeek:
		return model.OneDayAWeek
	case db.HolidayPolicyShiftSystem:
		return model.ShiftSystem
	default:
		return model.UnknownHoliday
	}
}

func toDBHolidayPolicy(hp model.HolidayPolicy) db.HolidayPolicy {
	switch hp {
	case model.CompleteTwoDaysAWeek:
		return db.HolidayPolicyCompleteTwoDaysAWeek
	case model.TwoDaysAWeek:
		return db.HolidayPolicyTwoDaysAWeek
	case model.OneDayAWeek:
		return db.HolidayPolicyOneDayAWeek
	case model.ShiftSystem:
		return db.HolidayPolicyShiftSystem
	default:
		return db.HolidayPolicyUnknownHoliday
	}
}

func toModelWorkplaceType(wt db.WorkplaceType) model.WorkplaceType {
	switch wt {
	case db.WorkplaceTypeOnsite:
		return model.Onsite
	case db.WorkplaceTypeRemote:
		return model.Remote
	case db.WorkplaceTypeHybrid:
		return model.Hybrid
	case db.WorkplaceTypeFullRemote:
		return model.FullRemote
	default:
		return model.UnknownWorkplace
	}
}

func toDBWorkplaceType(wt model.WorkplaceType) db.WorkplaceType {
	switch wt {
	case model.Onsite:
		return db.WorkplaceTypeOnsite
	case model.Remote:
		return db.WorkplaceTypeRemote
	case model.Hybrid:
		return db.WorkplaceTypeHybrid
	case model.FullRemote:
		return db.WorkplaceTypeFullRemote
	default:
		return db.WorkplaceTypeUnknownWorkplace
	}
}
