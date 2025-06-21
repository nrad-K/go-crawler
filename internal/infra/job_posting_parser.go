package infra

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

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
	ParseLocation(location string) (model.Location, error)
}

type CompiledPatterns struct {
	RaisePatterns       []*regexp.Regexp
	BonusPatterns       []*regexp.Regexp
	AmountPattern       *regexp.Regexp
	SalaryRangePattern  *regexp.Regexp
	SalarySinglePattern *regexp.Regexp
	LocationPattern     *regexp.Regexp
}

type jobPostingParser struct {
	patterns CompiledPatterns
}

func NewJobPostingParser(patterns CompiledPatterns) *jobPostingParser {
	return &jobPostingParser{
		patterns: patterns,
	}
}

// ParseJobType は文字列からmodel.JobTypeに変換します。
func (p *jobPostingParser) ParseJobType(jobTypeStr string) model.JobType {
	jobTypeStr = p.normalizeString(jobTypeStr)
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
	postedAtStr = p.normalizeString(postedAtStr)
	formats := []string{
		"2006年01月02日",     // 例: 2023年03月15日
		"2006/01/02",      // 例: 2023/03/15
		"2006-01-02",      // 例: 2023-03-15
		"2006.01.02",      // 例: 2025.06.17
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
	amountStr = p.normalizeString(amountStr)
	if amountStr == "" {
		return 0, fmt.Errorf("金額文字列が空です")
	}

	unitMap := map[string]float64{
		"億": 1e8,
		"万": 1e4,
		"千": 1e3,
	}

	for unit, multiplier := range unitMap {
		if strings.Contains(amountStr, unit) {
			// re := regexp.MustCompile(`(\d+(?:\.\d+)?)`)
			matches := p.patterns.AmountPattern.FindStringSubmatch(amountStr)
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
	text = p.normalizeString(text)
	for _, pattern := range p.patterns.RaisePatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) <= 1 {
			continue
		}

		if count, err := strconv.ParseUint(matches[1], 10, 64); err == nil {
			val := uint(count)
			return &val
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
	text = p.normalizeString(text)

	for _, pattern := range p.patterns.BonusPatterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) <= 1 {
			continue
		}

		if count, err := strconv.ParseUint(matches[1], 10, 64); err == nil {
			val := uint(count)
			return &val
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
	salaryStr = p.normalizeString(salaryStr)
	if salaryStr == "" {
		return model.NewSalary(0, nil, model.UnknownSalaryType), fmt.Errorf("給与文字列が空です")
	}

	unit := p.ParseSalaryType(salaryStr)

	// 範囲表現の処理
	if matches := p.patterns.SalaryRangePattern.FindStringSubmatch(salaryStr); len(matches) >= 3 {
		minStr := matches[1]
		maxStr := matches[2]

		// 下限に単位がなく上限にある場合、上限の単位を下限に付与する
		// 例: 400〜500万円 -> 400万円〜500万円
		unitRegex := regexp.MustCompile(`(万|千|億)`)
		minUnitMatch := unitRegex.FindString(minStr)
		maxUnitMatch := unitRegex.FindString(maxStr)

		if minUnitMatch == "" && maxUnitMatch != "" {
			minStr += maxUnitMatch
		}

		minAmount, err := p.ParseAmount(minStr)
		if err != nil {
			return model.NewSalary(0, nil, model.UnknownSalaryType), fmt.Errorf("給与の下限値のパースに失敗しました: %w", err)
		}

		maxAmount, err := p.ParseAmount(maxStr)
		if err != nil {
			return model.NewSalary(0, nil, model.UnknownSalaryType), fmt.Errorf("給与の上限値のパースに失敗しました: %w", err)
		}

		return model.NewSalary(minAmount, &maxAmount, unit), nil
	}

	// reSingle := regexp.MustCompile(`(\d+(?:\.\d+)?[万億千]?)`)
	// 単一表現の処理
	if singleMatch := p.patterns.SalarySinglePattern.FindStringSubmatch(salaryStr); len(singleMatch) >= 2 {
		amount, err := p.ParseAmount(singleMatch[1])
		if err != nil {
			return model.NewSalary(0, nil, model.UnknownSalaryType), fmt.Errorf("給与のパースに失敗しました: %w", err)
		}

		return model.NewSalary(amount, nil, unit), nil
	}

	return model.NewSalary(0, nil, model.UnknownSalaryType), fmt.Errorf("給与の金額を抽出できませんでした: %s", salaryStr)
}

// ParseSalaryUnitAndCurrency は給与情報からUnitとIsFixedを抽出します。
func (p *jobPostingParser) ParseSalaryType(salaryStr string) model.SalaryType {
	switch {
	case strings.Contains(salaryStr, "年収"), strings.Contains(salaryStr, "年給"):
		return model.Yearly
	case strings.Contains(salaryStr, "月給"):
		return model.Monthly
	case strings.Contains(salaryStr, "日給"):
		return model.Daily
	case strings.Contains(salaryStr, "時給"):
		return model.Hourly
	default:
		return model.UnknownSalaryType
	}
}

// ParseOptionalUint はオプションの数値を抽出し、*uint型で返します。
// 文字列が空の場合やパースできない場合はnilを返します。
func (p *jobPostingParser) ParseOptionalUint(optionalStr string) (*uint, error) {
	optionalStr = p.normalizeString(optionalStr)
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
	policyStr = p.normalizeString(policyStr)
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
	workplaceTypeStr = p.normalizeString(workplaceTypeStr)
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
	var benefits model.BenefitsArgs
	benefits.RawBenefits = benefitsStr // 元の文字列を保存
	normalizedBenefitsStr := p.normalizeString(benefitsStr)

	// キーワードに基づいて各フィールドを設定
	if strings.Contains(normalizedBenefitsStr, "社会保険完備") {
		benefits.SocialInsurance = true
	}
	if strings.Contains(normalizedBenefitsStr, "交通費支給") {
		benefits.TransportAllowance = true
	}
	if strings.Contains(normalizedBenefitsStr, "住宅手当") {
		benefits.HousingAllowance = true
	}
	if strings.Contains(normalizedBenefitsStr, "社宅・寮") {
		benefits.CompanyHousing = true
	}
	if strings.Contains(normalizedBenefitsStr, "家賃補助") {
		benefits.RentSubsidy = true
	}
	if strings.Contains(normalizedBenefitsStr, "食事手当") {
		benefits.MealAllowance = true
	}
	if strings.Contains(normalizedBenefitsStr, "社員食堂") {
		benefits.CafeteriaProvided = true
	}
	if strings.Contains(normalizedBenefitsStr, "研修制度") {
		benefits.TrainingSupport = true
	}
	if strings.Contains(normalizedBenefitsStr, "資格取得支援") {
		benefits.CertificationSupport = true
	}
	if strings.Contains(normalizedBenefitsStr, "有給休暇") {
		benefits.PaidLeave = true
	}
	if strings.Contains(normalizedBenefitsStr, "特別休暇") {
		benefits.SpecialLeave = true
	}
	if strings.Contains(normalizedBenefitsStr, "フレックスタイム") {
		benefits.FlexTime = true
	}
	if strings.Contains(normalizedBenefitsStr, "時短勤務") {
		benefits.ShortWorkingHours = true
	}
	if strings.Contains(normalizedBenefitsStr, "育児支援") {
		benefits.ChildcareSupport = true
	}
	if strings.Contains(normalizedBenefitsStr, "産前産後休暇") {
		benefits.MaternityLeave = true
	}
	if strings.Contains(normalizedBenefitsStr, "育児休暇") {
		benefits.ParentalLeave = true
	}
	if strings.Contains(normalizedBenefitsStr, "介護支援") {
		benefits.ElderCareSupport = true
	}
	if strings.Contains(normalizedBenefitsStr, "退職金制度") {
		benefits.RetirementPlan = true
	}
	return model.NewBenefits(benefits)
}

var (
	// 全角記号を半角に変換するためのリプレーサー
	symbolReplacer = strings.NewReplacer(
		"～", "~",
		"／", "/",
		"（", "(",
		"）", ")",
		"！", "!",
		"？", "?",
		"：", ":",
		"　", " ", // 全角スペース
	)

	// 都道府県名と PrefectureCode の対応表
	prefMap = map[string]model.PrefectureCode{
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
)

func (p *jobPostingParser) ParseLocation(locationStr string) (model.Location, error) {
	locationStr = p.normalizeString(locationStr)
	if locationStr == "" {
		return model.Location{}, fmt.Errorf("位置情報文字列が空です")
	}

	var name string
	var code model.PrefectureCode

	// 都道府県名の特定
	for k, v := range prefMap {
		// "東京都" -> "東京" のように末尾の文字を削除
		shortName := k
		if strings.HasSuffix(k, "都") || strings.HasSuffix(k, "府") || strings.HasSuffix(k, "県") {
			shortName = string([]rune(k)[:len([]rune(k))-1])
		}

		if strings.Contains(locationStr, k) || strings.Contains(locationStr, shortName) {
			name = k
			code = v
			break
		}
	}

	if name == "" {
		return model.Location{}, fmt.Errorf("都道府県名が特定できませんでした: %s", locationStr)
	}

	var city string
	// 市区町村の抽出（例: 東京都渋谷区 → 渋谷区）
	// re := regexp.MustCompile(`(?:都|道|府|県)(.+?[市区町村])`)
	match := p.patterns.LocationPattern.FindStringSubmatch(locationStr)
	if len(match) >= 2 {
		city = match[1]
	}

	return model.NewLocation(code, name, city, locationStr), nil
}

// 文字列の正規化を行うヘルパー関数
func (p *jobPostingParser) normalizeString(s string) string {
	// 全角記号を半角に変換
	s = symbolReplacer.Replace(s)

	// 全角スペースも含めてトリム
	s = strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsSpace(r)
	})

	// 全角数字を半角に変換し、制御文字を削除
	s = strings.Map(func(r rune) rune {
		if r >= '０' && r <= '９' {
			return r - '０' + '0'
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)

	return s
}
