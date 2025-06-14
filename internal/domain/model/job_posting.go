package model

import (
	"time"

	"github.com/google/uuid"
)

type SalaryType string

const (
	Hourly            SalaryType = "時給"
	Daily             SalaryType = "日給"
	Monthly           SalaryType = "月給"
	Yearly            SalaryType = "年給"
	UnknownSalaryType SalaryType = "不明"
)

type Salary struct {
	MinAmount uint64     `json:"min_amount"`
	MaxAmount uint64     `json:"max_amount"`
	Unit      SalaryType `json:"unit"`
	IsFixed   bool       `json:"is_fixed"`
}

type JobType string

const (
	FullTime   JobType = "正社員"
	PartTime   JobType = "アルバイト・パート"
	Contract   JobType = "契約社員"
	Temporary  JobType = "派遣社員"
	Freelance  JobType = "業務委託"
	Internship JobType = "インターン"
	Other      JobType = "その他"
	Unknown    JobType = "不明"
)

type JobPosting struct {
	ID           uuid.UUID        `json:"id"`
	Title        string           `json:"title"`
	CompanyName  string           `json:"company_name"`
	SummaryURL   string           `json:"summary_url"`
	Location     Location         `json:"location"`
	Headquarters Location         `json:"headquarters"`
	JobType      JobType          `json:"job_type"`
	Salary       Salary           `json:"salary"`
	PostedAt     time.Time        `json:"posted_at"`
	Details      JobPostingDetail `json:"details"`
}

type Location struct {
	PrefectureCode PrefectureCode `json:"prefecture_code"`
	PrefectureName string         `json:"prefecture_name"`
	City           string         `json:"city"`
	Raw            string         `json:"raw"`
}

type HolidayPolicy string

const (
	CompleteTwoDaysAWeek HolidayPolicy = "完全週休二日制"
	TwoDaysAWeek         HolidayPolicy = "週休二日制"
	OneDayAWeek          HolidayPolicy = "週休制"
	ShiftSystem          HolidayPolicy = "シフト制"
	UnknownHoliday       HolidayPolicy = "不明"
)

type WorkplaceType string

const (
	Onsite           WorkplaceType = "出社"
	Remote           WorkplaceType = "在宅"
	Hybrid           WorkplaceType = "ハイブリッド"
	FullRemote       WorkplaceType = "フルリモート"
	UnknownWorkplace WorkplaceType = "不明"
)

type JobPostingDetail struct {
	JobName         string        `json:"job_name"`
	Raise           *uint         `json:"raise,omitempty"` // nil = 昇給なしまたは未定
	Bonus           *uint         `json:"bonus,omitempty"` // nil = 賞与なしまたは未定
	Description     string        `json:"description"`
	Requirements    string        `json:"requirements"`
	WorkplaceType   WorkplaceType `json:"work_place_type"`
	HolidaysPerYear *uint         `json:"holidays_per_year,omitempty"` // nil = 記載なし
	HolidayPolicy   HolidayPolicy `json:"holiday_policy"`
	WorkHours       string        `json:"work_hours"`
	Benefits        Benefits      `json:"benefits"`
}

// 福利厚生をbooleanフィールドで表現
type Benefits struct {
	// 保険関連
	SocialInsurance bool `json:"social_insurance"` // 社会保険完備

	// 交通・通勤
	TransportAllowance bool `json:"transport_allowance"` // 交通費支給

	// 住宅関連
	HousingAllowance bool `json:"housing_allowance"` // 住宅手当
	CompanyHousing   bool `json:"company_housing"`   // 社宅・寮
	RentSubsidy      bool `json:"rent_subsidy"`      // 家賃補助

	// 食事・生活
	MealAllowance     bool `json:"meal_allowance"`     // 食事手当
	CafeteriaProvided bool `json:"cafeteria_provided"` // 社員食堂

	// 教育・研修
	TrainingSupport      bool `json:"training_support"`      // 研修制度
	CertificationSupport bool `json:"certification_support"` // 資格取得支援

	// 休暇・時間
	PaidLeave         bool `json:"paid_leave"`          // 有給休暇
	SpecialLeave      bool `json:"special_leave"`       // 特別休暇
	FlexTime          bool `json:"flex_time"`           // フレックスタイム
	ShortWorkingHours bool `json:"short_working_hours"` // 時短勤務

	// ライフサポート
	ChildcareSupport bool `json:"childcare_support"`  // 育児支援
	MaternityLeave   bool `json:"maternity_leave"`    // 産前産後休暇
	ParentalLeave    bool `json:"parental_leave"`     // 育児休暇
	ElderCareSupport bool `json:"elder_care_support"` // 介護支援

	// その他
	RetirementPlan bool `json:"retirement_plan"` // 退職金制度

	// 原文も保持
	RawBenefits string `json:"raw_benefits"` // 元の文字列
}

type PrefectureCode string

const (
	Hokkaido  PrefectureCode = "01"
	Aomori    PrefectureCode = "02"
	Iwate     PrefectureCode = "03"
	Miyagi    PrefectureCode = "04"
	Akita     PrefectureCode = "05"
	Yamagata  PrefectureCode = "06"
	Fukushima PrefectureCode = "07"
	Ibaraki   PrefectureCode = "08"
	Tochigi   PrefectureCode = "09"
	Gunma     PrefectureCode = "10"
	Saitama   PrefectureCode = "11"
	Chiba     PrefectureCode = "12"
	Tokyo     PrefectureCode = "13"
	Kanagawa  PrefectureCode = "14"
	Niigata   PrefectureCode = "15"
	Toyama    PrefectureCode = "16"
	Ishikawa  PrefectureCode = "17"
	Fukui     PrefectureCode = "18"
	Yamanashi PrefectureCode = "19"
	Nagano    PrefectureCode = "20"
	Gifu      PrefectureCode = "21"
	Shizuoka  PrefectureCode = "22"
	Aichi     PrefectureCode = "23"
	Mie       PrefectureCode = "24"
	Shiga     PrefectureCode = "25"
	Kyoto     PrefectureCode = "26"
	Osaka     PrefectureCode = "27"
	Hyogo     PrefectureCode = "28"
	Nara      PrefectureCode = "29"
	Wakayama  PrefectureCode = "30"
	Tottori   PrefectureCode = "31"
	Shimane   PrefectureCode = "32"
	Okayama   PrefectureCode = "33"
	Hiroshima PrefectureCode = "34"
	Yamaguchi PrefectureCode = "35"
	Tokushima PrefectureCode = "36"
	Kagawa    PrefectureCode = "37"
	Ehime     PrefectureCode = "38"
	Kochi     PrefectureCode = "39"
	Fukuoka   PrefectureCode = "40"
	Saga      PrefectureCode = "41"
	Nagasaki  PrefectureCode = "42"
	Kumamoto  PrefectureCode = "43"
	Oita      PrefectureCode = "44"
	Miyazaki  PrefectureCode = "45"
	Kagoshima PrefectureCode = "46"
	Okinawa   PrefectureCode = "47"
)
