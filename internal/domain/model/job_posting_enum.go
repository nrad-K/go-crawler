package model

type SalaryType string

const (
	Hourly            SalaryType = "時給"
	Daily             SalaryType = "日給"
	Monthly           SalaryType = "月給"
	Yearly            SalaryType = "年給"
	UnknownSalaryType SalaryType = "不明"
)

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
