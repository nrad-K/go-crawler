package model

import (
	"time"

	"github.com/google/uuid"
)

type SalaryType int

const (
	Hourly SalaryType = iota
	Daily
	Monthly
	Yearly
)

func (st SalaryType) String() string {
	switch st {
	case Hourly:
		return "時給"
	case Daily:
		return "日給"
	case Monthly:
		return "月給"
	case Yearly:
		return "年給"
	default:
		return "不明"
	}
}

type Currency string

const (
	JPY Currency = "JPY"
	USD Currency = "USD"
	// 必要に応じて他の通貨を追加
)

type Salary struct {
	MinAmount uint64     `json:"min_amount"`
	MaxAmount uint64     `json:"max_amount"`
	Unit      SalaryType `json:"unit"`
	Currency  Currency   `json:"currency"`
	IsFixed   bool       `json:"is_fixed"`
}

func (s Salary) IsValid() bool {
	return s.MinAmount <= s.MaxAmount
}

type JobType int

const (
	Unknown    JobType = iota
	FullTime           // 正社員
	PartTime           // アルバイト・パート
	Contract           // 契約社員
	Temporary          // 派遣社員
	Freelance          // 業務委託
	Internship         // インターン
	Other              // その他
)

func (jt JobType) String() string {
	switch jt {
	case FullTime:
		return "正社員"
	case PartTime:
		return "アルバイト・パート"
	case Contract:
		return "契約社員"
	case Temporary:
		return "派遣社員"
	case Freelance:
		return "業務委託"
	case Internship:
		return "インターン"
	case Other:
		return "その他"
	default:
		return "不明"
	}
}

type JobPosting struct {
	ID          uuid.UUID        `json:"id"`
	Title       string           `json:"title"`
	CompanyName string           `json:"company_name"`
	Location    Location         `json:"location"`
	SummaryURL  string           `json:"summary_url"`
	JobType     JobType          `json:"job_type"`
	Salary      Salary           `json:"salary"`
	PostedAt    time.Time        `json:"posted_at"`
	Details     JobPostingDetail `json:"details"`
}

type Location struct {
	PrefectureCode string `json:"prefecture_code"`
	PrefectureName string `json:"prefecture_name"`
	Municipality   string `json:"municipality"` // 市区町村名だけで簡略化
}

type JobPostingDetail struct {
	JobName         string `json:"job_name"`
	HolidayPolicy   string `json:"holiday_policy"`
	Raise           *uint  `json:"raise,omitempty"` // nil = 昇給なしまたは未定
	Bonus           *uint  `json:"bonus,omitempty"` // nil = 賞与なしまたは未定
	Description     string `json:"description"`
	Requirements    string `json:"requirements"`
	HolidaysPerYear *uint  `json:"holidays_per_year,omitempty"` // nil = 記載なし
	WorkHours       string `json:"work_hours"`
	Benefits        string `json:"benefits"`
}
