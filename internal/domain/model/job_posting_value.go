package model

import "fmt"

type Amount struct {
	value uint64
	valid bool
}

func (a *Amount) Format() string {
	if !a.valid {
		return ""
	}
	return fmt.Sprintf("%d", a.value)
}

func NewAmount(value uint64) Amount {
	return Amount{
		value: uint64(value),
		valid: true,
	}
}

func NewNullAmount() Amount {
	return Amount{
		value: 0,
		valid: false,
	}
}

type Salary struct {
	minAmount Amount
	maxAmount Amount
	unit      SalaryType
}

func NewSalary(minAmount Amount, maxAmount Amount, salaryType SalaryType) Salary {
	return Salary{
		minAmount: minAmount,
		maxAmount: maxAmount,
		unit:      salaryType,
	}
}

func (s Salary) MinAmount() Amount {
	return s.minAmount
}

func (s Salary) MaxAmount() Amount {
	return s.maxAmount
}

func (s Salary) Unit() SalaryType {
	return s.unit
}

type Location struct {
	prefectureCode PrefectureCode
	prefectureName string
	city           string
	raw            string
}

func NewLocation(code PrefectureCode, name, city, raw string) Location {
	return Location{
		prefectureCode: code,
		prefectureName: name,
		city:           city,
		raw:            raw,
	}
}

func (l Location) PrefectureCode() PrefectureCode {
	return l.prefectureCode
}

func (l Location) PrefectureName() string {
	return l.prefectureName
}

func (l Location) City() string {
	return l.city
}

func (l Location) Raw() string {
	return l.raw
}

// 福利厚生の引数が多いため、構造体にまとめて渡す形に変更
type Benefits struct {
	// 保険関連
	socialInsurance bool

	// 交通・通勤
	transportAllowance bool

	// 住宅関連
	housingAllowance bool
	companyHousing   bool
	rentSubsidy      bool

	// 食事・生活
	mealAllowance     bool
	cafeteriaProvided bool

	// 教育・研修
	trainingSupport      bool
	certificationSupport bool

	// 休暇・時間
	paidLeave         bool
	specialLeave      bool
	flexTime          bool
	shortWorkingHours bool

	// ライフサポート
	childcareSupport bool
	maternityLeave   bool
	parentalLeave    bool
	elderCareSupport bool

	// その他
	retirementPlan bool

	// 原文も保持
	rawBenefits string
}

type BenefitsArgs struct {
	SocialInsurance      bool
	TransportAllowance   bool
	HousingAllowance     bool
	CompanyHousing       bool
	RentSubsidy          bool
	MealAllowance        bool
	CafeteriaProvided    bool
	TrainingSupport      bool
	CertificationSupport bool
	PaidLeave            bool
	SpecialLeave         bool
	FlexTime             bool
	ShortWorkingHours    bool
	ChildcareSupport     bool
	MaternityLeave       bool
	ParentalLeave        bool
	ElderCareSupport     bool
	RetirementPlan       bool
	RawBenefits          string
}

func NewBenefits(args BenefitsArgs) Benefits {
	return Benefits{
		socialInsurance:      args.SocialInsurance,
		transportAllowance:   args.TransportAllowance,
		housingAllowance:     args.HousingAllowance,
		companyHousing:       args.CompanyHousing,
		rentSubsidy:          args.RentSubsidy,
		mealAllowance:        args.MealAllowance,
		cafeteriaProvided:    args.CafeteriaProvided,
		trainingSupport:      args.TrainingSupport,
		certificationSupport: args.CertificationSupport,
		paidLeave:            args.PaidLeave,
		specialLeave:         args.SpecialLeave,
		flexTime:             args.FlexTime,
		shortWorkingHours:    args.ShortWorkingHours,
		childcareSupport:     args.ChildcareSupport,
		maternityLeave:       args.MaternityLeave,
		parentalLeave:        args.ParentalLeave,
		elderCareSupport:     args.ElderCareSupport,
		retirementPlan:       args.RetirementPlan,
		rawBenefits:          args.RawBenefits,
	}
}

func (b Benefits) RawBenefits() string {
	return b.rawBenefits
}

type JobPostingDetailArgs struct {
	JobName         string
	Raise           *uint
	Bonus           *uint
	Description     string
	Requirements    string
	WorkplaceType   WorkplaceType
	HolidaysPerYear *uint
	HolidayPolicy   HolidayPolicy
	WorkHours       string
	Benefits        Benefits
}

type JobPostingDetail struct {
	jobName         string
	raise           *uint
	bonus           *uint
	description     string
	requirements    string
	workplaceType   WorkplaceType
	holidaysPerYear *uint
	holidayPolicy   HolidayPolicy
	workHours       string
	benefits        Benefits
}

func (d JobPostingDetail) JobName() string {
	return d.jobName
}

func (d JobPostingDetail) Raise() *uint {
	return d.raise
}

func (d JobPostingDetail) Bonus() *uint {
	return d.bonus
}

func (d JobPostingDetail) Description() string {
	return d.description
}

func (d JobPostingDetail) Requirements() string {
	return d.requirements
}

func (d JobPostingDetail) WorkplaceType() WorkplaceType {
	return d.workplaceType
}

func (d JobPostingDetail) HolidaysPerYear() *uint {
	return d.holidaysPerYear
}

func (d JobPostingDetail) HolidayPolicy() HolidayPolicy {
	return d.holidayPolicy
}

func (d JobPostingDetail) WorkHours() string {
	return d.workHours
}

func (d JobPostingDetail) Benefits() Benefits {
	return d.benefits
}

func NewJobPostingDetail(args JobPostingDetailArgs) JobPostingDetail {
	return JobPostingDetail{
		jobName:         args.JobName,
		raise:           args.Raise,
		bonus:           args.Bonus,
		description:     args.Description,
		requirements:    args.Requirements,
		workplaceType:   args.WorkplaceType,
		holidaysPerYear: args.HolidaysPerYear,
		holidayPolicy:   args.HolidayPolicy,
		workHours:       args.WorkHours,
		benefits:        args.Benefits,
	}
}
