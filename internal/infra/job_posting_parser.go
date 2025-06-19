package infra

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nrad-K/go-crawler/internal/domain/model"
)

type JobPostingParser interface {
	ParseJobType(jobTypeStr string) model.JobType
	ParsePostedAt(postedAtStr string) (time.Time, error)
	ParseRaise(raiseStr string) *uint
	ParseBonus(bonusStr string) *uint
	ParseSalaryDetails(salaryStr string) (model.Salary, error)
	ParseHolidayPolicy(policyStr string) model.HolidayPolicy
	ParseWorkplaceType(workplaceTypeStr string) model.WorkplaceType
	ParseBenefits(benefitsStr string) model.Benefits
	ParseOptionalUint(optionalStr string) (*uint, error)
	ParseLocation(location string) model.Location
}

type jobPostingParser struct {
}

func NewJobPostingParser() *jobPostingParser {
	return &jobPostingParser{}
}

// ParseJobType は文字列からmodel.JobTypeに変換します。
func (p *jobPostingParser) ParseJobType(jobTypeStr string) model.JobType {
	if strings.Contains(jobTypeStr, "正社員") {
		return model.FullTime
	}
	if strings.Contains(jobTypeStr, "アルバイト") || strings.Contains(jobTypeStr, "パート") || strings.Contains(jobTypeStr, "バイト") {
		return model.PartTime
	}
	if strings.Contains(jobTypeStr, "契約社員") {
		return model.Contract
	}
	if strings.Contains(jobTypeStr, "派遣社員") {
		return model.Temporary
	}
	if strings.Contains(jobTypeStr, "業務委託") || strings.Contains(jobTypeStr, "フリーランス") {
		return model.Freelance
	}
	if strings.Contains(jobTypeStr, "インターン") {
		return model.Internship
	}
	return model.Unknown
}

// ParsePostedAt は文字列からtime.Timeに変換します。
func (p *jobPostingParser) ParsePostedAt(postedAtStr string) (time.Time, error) {
	formats := []string{
		"2006年01月02日",     // 例: 2023年03月15日
		"2006/01/02",      // 例: 2023/03/15
		"2006-01-02",      // 例: 2023-03-15
		"January 2, 2006", // 例: March 15, 2023
		"Jan 2, 2006",     // 例: Mar 15, 2023
	}

	for _, format := range formats {
		parsedTime, err := time.Parse(format, postedAtStr)
		if err == nil {
			return parsedTime, nil
		}
	}
	return time.Time{}, fmt.Errorf("日付のパースに失敗しました: %s", postedAtStr)
}

// ParseAmount は金額文字列からuint64に変換します。
func (p *jobPostingParser) ParseAmount(amountStr string) (uint64, error) {
	unitMap := map[string]float64{
		"億": 1e8,
		"万": 1e4,
		"千": 1e3,
	}

	for unit, multiplier := range unitMap {
		if strings.Contains(amountStr, unit) {
			re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
			matches := re.FindStringSubmatch(amountStr)
			if len(matches) == 0 {
				return 0, fmt.Errorf("パースする金額がありません: %s", amountStr)
			}
			amount, err := strconv.ParseFloat(matches[1], 64)
			if err != nil {
				return 0, fmt.Errorf("金額の数値変換に失敗しました: %w", err)
			}
			return uint64(amount * multiplier), nil
		}
	}

	// 通常の金額処理（カンマ除去）
	re := regexp.MustCompile(`[^0-9]`)
	cleanStr := re.ReplaceAllString(amountStr, "")
	if cleanStr == "" {
		return 0, fmt.Errorf("パースする金額がありません: %s", amountStr)
	}
	amount, err := strconv.ParseUint(cleanStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("金額の数値変換に失敗しました: %w", err)
	}
	return amount, nil
}

func (p *jobPostingParser) ParseRaise(text string) *uint {
	// 昇給に関するパターンを定義
	raisePatterns := []*regexp.Regexp{
		regexp.MustCompile(`昇給[／/]年(\d+)回`),   // 昇給／年1回
		regexp.MustCompile(`昇給.*年(\d+)回`),     // 昇給...年1回
		regexp.MustCompile(`年(\d+)回.*昇給`),     // 年1回...昇給
		regexp.MustCompile(`昇給.*(\d+)回[／/]年`), // 昇給...1回／年
		regexp.MustCompile(`昇給.*(\d+)回.*年`),   // 昇給...1回...年
	}

	for _, pattern := range raisePatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) > 1 {
			if count, err := strconv.ParseUint(matches[1], 10, 64); err == nil {
				val := uint(count)
				return &val
			}
		}
	}

	// パターンにマッチしないが「昇給」が含まれている場合は1回とみなす
	if strings.Contains(text, "昇給") {
		val := uint(1)
		return &val
	}

	return nil
}

func (p *jobPostingParser) ParseBonus(text string) *uint {
	// 賞与に関するパターンを定義
	bonusPatterns := []*regexp.Regexp{
		regexp.MustCompile(`賞与[／/]年(\d+)回`),   // 賞与／年2回
		regexp.MustCompile(`賞与.*年(\d+)回`),     // 賞与...年2回
		regexp.MustCompile(`年(\d+)回.*賞与`),     // 年2回...賞与
		regexp.MustCompile(`賞与.*(\d+)回[／/]年`), // 賞与...2回／年
		regexp.MustCompile(`賞与.*(\d+)回.*年`),   // 賞与...2回...年
		regexp.MustCompile(`ボーナス[／/]年(\d+)回`), // ボーナス／年2回
		regexp.MustCompile(`ボーナス.*年(\d+)回`),   // ボーナス...年2回
	}

	for _, pattern := range bonusPatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) > 1 {
			if count, err := strconv.ParseUint(matches[1], 10, 64); err == nil {
				val := uint(count)
				return &val
			}
		}
	}

	// パターンにマッチしないが「賞与」「ボーナス」が含まれている場合は1回とみなす
	if strings.Contains(text, "賞与") || strings.Contains(text, "ボーナス") {
		val := uint(1)
		return &val
	}

	return nil
}

// ParseSalaryDetails は給与情報を解析し、範囲や単位、固定・変動を返す
func (p *jobPostingParser) ParseSalaryDetails(salaryStr string) (model.Salary, error) {
	unit := p.ParseSalaryType(salaryStr)
	isFixed := true
	minAmount, maxAmount := uint64(0), uint64(0)

	// 範囲表現の正規表現（例：100万〜200万、100万から、以上、以下 等）
	re := regexp.MustCompile(`(\d+(?:\.\d+)?[万億千]?)(?:円)?(?:〜|～|\-|ー|から|以上)?.{0,10}?(\d+(?:\.\d+)?[万億千]?)(?:円)?(?:まで|以下)?`)
	matches := re.FindStringSubmatch(salaryStr)

	if len(matches) >= 3 {
		// 範囲指定あり
		var err error
		minAmount, err = p.ParseAmount(matches[1])
		if err != nil {
			return model.Salary{
				MinAmount: 0,
				MaxAmount: 0,
				Unit:      unit,
				IsFixed:   isFixed,
			}, fmt.Errorf("給与の下限値のパースに失敗しました: %w", err)
		}
		maxAmount, err = p.ParseAmount(matches[2])
		if err != nil {
			return model.Salary{
				MinAmount: 0,
				MaxAmount: 0,
				Unit:      unit,
				IsFixed:   isFixed,
			}, fmt.Errorf("給与の上限値のパースに失敗しました: %w", err)
		}
		isFixed = false
		return model.Salary{
			MinAmount: minAmount,
			MaxAmount: maxAmount,
			Unit:      unit,
			IsFixed:   isFixed,
		}, nil
	}

	// 範囲表現がなければ単一金額を抽出
	reSingle := regexp.MustCompile(`(\d+(?:\.\d+)?[万億千]?)`)
	singleMatch := reSingle.FindStringSubmatch(salaryStr)
	if len(singleMatch) < 2 {
		return model.Salary{
			MinAmount: 0,
			MaxAmount: 0,
			Unit:      unit,
			IsFixed:   isFixed,
		}, fmt.Errorf("給与の金額を抽出できませんでした: %s", salaryStr)
	}

	var err error
	minAmount, err = p.ParseAmount(singleMatch[1])
	if err != nil {
		return model.Salary{
			MinAmount: 0,
			MaxAmount: 0,
			Unit:      unit,
			IsFixed:   isFixed,
		}, fmt.Errorf("給与の金額のパースに失敗しました: %w", err)
	}
	maxAmount = minAmount

	// 「〜」「以上」などで上限なし判定
	rangeIndicators := []string{"〜", "～", "以上", "から", "-", "ー"}
	for _, indicator := range rangeIndicators {
		if strings.Contains(salaryStr, indicator) {
			isFixed = false
			maxAmount = 0
			break
		}
	}

	return model.Salary{
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		Unit:      unit,
		IsFixed:   isFixed,
	}, nil
}

// ParseSalaryUnitAndCurrency は給与情報からUnitとIsFixedを抽出します。
func (p *jobPostingParser) ParseSalaryType(salaryStr string) model.SalaryType {
	var unit model.SalaryType = model.UnknownSalaryType

	switch {
	case strings.Contains(salaryStr, "年収"), strings.Contains(salaryStr, "年給"):
		unit = model.Yearly
	case strings.Contains(salaryStr, "月給"):
		unit = model.Monthly
	case strings.Contains(salaryStr, "日給"):
		unit = model.Daily
	case strings.Contains(salaryStr, "時給"):
		unit = model.Hourly
	default:
		unit = p.InferSalaryUnitFromAmount(salaryStr)
	}

	return unit
}

func (p *jobPostingParser) InferSalaryUnitFromAmount(salaryStr string) model.SalaryType {
	re := regexp.MustCompile(`(\d+)\s*万`)
	matches := re.FindAllStringSubmatch(salaryStr, -1)

	for _, m := range matches {
		amount, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		switch {
		case amount >= 100:
			return model.Yearly
		case amount >= 20:
			return model.Monthly
		}
	}
	return model.UnknownSalaryType
}

// ParseOptionalUint はオプションの数値を抽出し、*uint型で返します。
// 文字列が空の場合やパースできない場合はnilを返します。
func (p *jobPostingParser) ParseOptionalUint(optionalStr string) (*uint, error) {
	if optionalStr == "" {
		return nil, nil // 値がない場合はnilを返す
	}

	re := regexp.MustCompile(`[^0-9]`)
	cleanStr := re.ReplaceAllString(optionalStr, "")

	if cleanStr == "" {
		return nil, nil // クリーンアップ後に空になった場合もnilを返す
	}

	parsedVal, err := strconv.ParseUint(cleanStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("オプションの数値のパースに失敗しました: %w", err)
	}

	// uint64からuintへ変換。Goのuintはシステム依存のサイズだが、ここでは十分なサイズを想定。
	val := uint(parsedVal)
	return &val, nil
}

// ParseHolidayPolicy は文字列からmodel.HolidayPolicyに変換します。
func (p *jobPostingParser) ParseHolidayPolicy(policyStr string) model.HolidayPolicy {
	if strings.Contains(policyStr, "完全週休二日制") {
		return model.CompleteTwoDaysAWeek
	}
	if strings.Contains(policyStr, "週休二日制") {
		return model.TwoDaysAWeek
	}
	if strings.Contains(policyStr, "週休制") {
		return model.OneDayAWeek
	}
	if strings.Contains(policyStr, "シフト制") {
		return model.ShiftSystem
	}
	return model.UnknownHoliday
}

// ParseWorkplaceType は文字列からmodel.WorkplaceTypeに変換します。
func (p *jobPostingParser) ParseWorkplaceType(workplaceTypeStr string) model.WorkplaceType {
	if strings.Contains(workplaceTypeStr, "出社") {
		return model.Onsite
	}
	if strings.Contains(workplaceTypeStr, "在宅") || strings.Contains(workplaceTypeStr, "リモート") || strings.Contains(workplaceTypeStr, "フルリモート") {
		return model.Remote
	}
	if strings.Contains(workplaceTypeStr, "ハイブリッド") {
		return model.Hybrid
	}
	return model.UnknownWorkplace
}

// ParseBenefits は福利厚生の文字列をパースしてmodel.Benefits構造体に変換します。
func (p *jobPostingParser) ParseBenefits(benefitsStr string) model.Benefits {
	var benefits model.Benefits
	benefits.RawBenefits = benefitsStr // 元の文字列を保存

	// キーワードに基づいて各フィールドを設定
	if strings.Contains(benefitsStr, "社会保険完備") {
		benefits.SocialInsurance = true
	}
	if strings.Contains(benefitsStr, "交通費支給") {
		benefits.TransportAllowance = true
	}
	if strings.Contains(benefitsStr, "住宅手当") {
		benefits.HousingAllowance = true
	}
	if strings.Contains(benefitsStr, "社宅・寮") {
		benefits.CompanyHousing = true
	}
	if strings.Contains(benefitsStr, "家賃補助") {
		benefits.RentSubsidy = true
	}
	if strings.Contains(benefitsStr, "食事手当") {
		benefits.MealAllowance = true
	}
	if strings.Contains(benefitsStr, "社員食堂") {
		benefits.CafeteriaProvided = true
	}
	if strings.Contains(benefitsStr, "研修制度") {
		benefits.TrainingSupport = true
	}
	if strings.Contains(benefitsStr, "資格取得支援") {
		benefits.CertificationSupport = true
	}
	if strings.Contains(benefitsStr, "有給休暇") {
		benefits.PaidLeave = true
	}
	if strings.Contains(benefitsStr, "特別休暇") {
		benefits.SpecialLeave = true
	}
	if strings.Contains(benefitsStr, "フレックスタイム") {
		benefits.FlexTime = true
	}
	if strings.Contains(benefitsStr, "時短勤務") {
		benefits.ShortWorkingHours = true
	}
	if strings.Contains(benefitsStr, "育児支援") {
		benefits.ChildcareSupport = true
	}
	if strings.Contains(benefitsStr, "産前産後休暇") {
		benefits.MaternityLeave = true
	}
	if strings.Contains(benefitsStr, "育児休暇") {
		benefits.ParentalLeave = true
	}
	if strings.Contains(benefitsStr, "介護支援") {
		benefits.ElderCareSupport = true
	}
	if strings.Contains(benefitsStr, "退職金制度") {
		benefits.RetirementPlan = true
	}
	return benefits
}

func (p *jobPostingParser) ParseLocation(locationStr string) model.Location {
	var location model.Location
	location.Raw = locationStr

	// 都道府県名と PrefectureCode の対応表
	prefMap := map[string]model.PrefectureCode{
		"北海道":  model.Hokkaido,
		"青森県":  model.Aomori,
		"岩手県":  model.Iwate,
		"宮城県":  model.Miyagi,
		"秋田県":  model.Akita,
		"山形県":  model.Yamagata,
		"福島県":  model.Fukushima,
		"茨城県":  model.Ibaraki,
		"栃木県":  model.Tochigi,
		"群馬県":  model.Gunma,
		"埼玉県":  model.Saitama,
		"千葉県":  model.Chiba,
		"東京都":  model.Tokyo,
		"神奈川県": model.Kanagawa,
		"新潟県":  model.Niigata,
		"富山県":  model.Toyama,
		"石川県":  model.Ishikawa,
		"福井県":  model.Fukui,
		"山梨県":  model.Yamanashi,
		"長野県":  model.Nagano,
		"岐阜県":  model.Gifu,
		"静岡県":  model.Shizuoka,
		"愛知県":  model.Aichi,
		"三重県":  model.Mie,
		"滋賀県":  model.Shiga,
		"京都府":  model.Kyoto,
		"大阪府":  model.Osaka,
		"兵庫県":  model.Hyogo,
		"奈良県":  model.Nara,
		"和歌山県": model.Wakayama,
		"鳥取県":  model.Tottori,
		"島根県":  model.Shimane,
		"岡山県":  model.Okayama,
		"広島県":  model.Hiroshima,
		"山口県":  model.Yamaguchi,
		"徳島県":  model.Tokushima,
		"香川県":  model.Kagawa,
		"愛媛県":  model.Ehime,
		"高知県":  model.Kochi,
		"福岡県":  model.Fukuoka,
		"佐賀県":  model.Saga,
		"長崎県":  model.Nagasaki,
		"熊本県":  model.Kumamoto,
		"大分県":  model.Oita,
		"宮崎県":  model.Miyazaki,
		"鹿児島県": model.Kagoshima,
		"沖縄県":  model.Okinawa,
	}

	// 都道府県名の特定
	for name, code := range prefMap {
		if strings.Contains(locationStr, name) {
			location.PrefectureName = name
			location.PrefectureCode = code
			break
		}
	}

	// 市区町村の抽出（例: 東京都渋谷区 → 渋谷区）
	re := regexp.MustCompile(`(?:都|道|府|県)(.+?[市区町村])`)
	match := re.FindStringSubmatch(locationStr)
	if len(match) >= 2 {
		location.City = match[1]
	}

	return location
}
