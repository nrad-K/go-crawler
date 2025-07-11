package infra

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/nrad-K/go-crawler/internal/domain/model"
	"golang.org/x/text/width"
)

// JobPostingParserは、求人情報の様々な要素を文字列から解析するためのインターフェースです。
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

// CompiledPatternsは、解析処理で使用されるコンパイル済みの正規表現を保持します。
// これにより、パースのたびに正規表現をコンパイルするオーバーヘッドを削減します。
type CompiledPatterns struct {
	RaisePatterns       []*regexp.Regexp
	BonusPatterns       []*regexp.Regexp
	AmountPattern       *regexp.Regexp
	SalaryRangePattern  *regexp.Regexp
	SalarySinglePattern *regexp.Regexp
	LocationPattern     *regexp.Regexp
}

// jobPostingParserは、JobPostingParserインターフェースの実装です。
//
// フィールド:
//
//	patterns: コンパイル済みの正規表現パターン
type jobPostingParser struct {
	patterns CompiledPatterns
}

// NewJobPostingParserは、jobPostingParserの新しいインスタンスを生成します。
//
// args:
//
//	patterns: 解析に使用するコンパイル済み正規表現
//
// return:
//
//	*jobPostingParser: 新しいパーサーのインスタンス
func NewJobPostingParser(patterns CompiledPatterns) *jobPostingParser {
	return &jobPostingParser{
		patterns: patterns,
	}
}

// ParseJobTypeは、与えられた雇用形態の文字列を解析し、対応するmodel.JobType定数を返します。
//
// args:
//
//	jobTypeStr: 解析対象の雇用形態の文字列 (例: "正社員", "アルバイト")
//
// return:
//
//	model.JobType: 解析結果の雇用形態
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

// ParsePostedAtは、様々な形式の投稿日の文字列を解析し、time.Timeオブジェクトに変換します。
//
// args:
//
//	postedAtStr: 解析対象の日付文字列 (例: "2023年03月15日", "2023/03/15")
//
// return:
//
//	time.Time: 解析された時刻
//	error    : いずれの形式にもマッチしない場合のエラー
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

// ParseAmountは、"100万円"や"500,000"のような金額を表す文字列から、数値を抽出しuint64型で返します。
//
// args:
//
//	amountStr: 解析対象の金額文字列
//
// return:
//
//	uint64: 解析された金額
//	error : 解析に失敗した場合のエラー
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

// ParseRaiseは、昇給情報を含むテキストから年間の昇給回数を抽出します。
//
// args:
//
//	text: 昇給情報を含む文字列
//
// return:
//
//	*uint: 抽出された昇給回数。見つからない場合はnil。
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

// ParseBonusは、賞与情報を含むテキストから年間の賞与回数を抽出します。
//
// args:
//
//	text: 賞与情報を含む文字列
//
// return:
//
//	*uint: 抽出された賞与回数。見つからない場合はnil。
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

// ParseSalaryDetailsは、給与情報の文字列を解析し、給与の範囲、単位などを含むmodel.Salaryオブジェクトを返します。
//
// args:
//
//	salaryStr: 解析対象の給与情報文字列 (例: "月給25万円～", "年収400万円～800万円")
//
// return:
//
//	model.Salary: 解析された給与情報
//	error       : 解析に失敗した場合のエラー
func (p *jobPostingParser) ParseSalaryDetails(salaryStr string) (model.Salary, error) {
	salaryStr = p.normalizeString(salaryStr)
	if salaryStr == "" {
		minAmount := model.NewAmount(0)
		maxAmount := model.NewNullAmount()
		return model.NewSalary(minAmount, maxAmount, model.UnknownSalaryType), fmt.Errorf("給与文字列が空です")
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

		pMinAmount, err := p.ParseAmount(minStr)
		if err != nil {
			minAmount := model.NewAmount(0)
			maxAmount := model.NewNullAmount()
			return model.NewSalary(minAmount, maxAmount, model.UnknownSalaryType), fmt.Errorf("給与の下限値のパースに失敗しました: %w", err)
		}

		pMaxAmount, err := p.ParseAmount(maxStr)
		if err != nil {
			minAmount := model.NewAmount(0)
			maxAmount := model.NewNullAmount()
			return model.NewSalary(minAmount, maxAmount, model.UnknownSalaryType), fmt.Errorf("給与の上限値のパースに失敗しました: %w", err)
		}

		minAmount := model.NewAmount(pMinAmount)
		maxAmount := model.NewAmount(pMaxAmount)

		return model.NewSalary(minAmount, maxAmount, unit), nil
	}

	// reSingle := regexp.MustCompile(`(\d+(?:\.\d+)?[万億千]?)`)
	// 単一表現の処理
	if singleMatch := p.patterns.SalarySinglePattern.FindStringSubmatch(salaryStr); len(singleMatch) >= 2 {
		amount, err := p.ParseAmount(singleMatch[1])
		maxAmount := model.NewNullAmount()
		if err != nil {
			minAmount := model.NewAmount(0)
			return model.NewSalary(minAmount, maxAmount, model.UnknownSalaryType), fmt.Errorf("給与のパースに失敗しました: %w", err)
		}

		minAmount := model.NewAmount(amount)
		return model.NewSalary(minAmount, maxAmount, unit), nil
	}

	minAmount := model.NewAmount(0)
	maxAmount := model.NewNullAmount()
	return model.NewSalary(minAmount, maxAmount, model.UnknownSalaryType), fmt.Errorf("給与の金額を抽出できませんでした: %s", salaryStr)
}

// ParseSalaryTypeは、給与情報の文字列から給与の単位（年収、月給など）を特定します。
//
// args:
//
//	salaryStr: 解析対象の給与情報文字列
//
// return:
//
//	model.SalaryType: 特定された給与単位
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

// ParseOptionalUintは、オプションの数値を含む文字列（例: 年間休日数）を解析し、*uint型で返します。
// 文字列が空の場合や数値に変換できない場合はnilを返します。
//
// args:
//
//	optionalStr: 解析対象の文字列
//
// return:
//
//	*uint: 解析された数値。解析できない場合はnil。
//	error: 数値への変換に失敗した場合のエラー
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

// ParseHolidayPolicyは、休日・休暇に関する文字列を解析し、対応するmodel.HolidayPolicyを返します。
//
// args:
//
//	policyStr: 解析対象の休日・休暇の文字列
//
// return:
//
//	model.HolidayPolicy: 解析された休日ポリシー
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

// ParseWorkplaceTypeは、勤務形態に関する文字列を解析し、対応するmodel.WorkplaceTypeを返します。
//
// args:
//
//	workplaceTypeStr: 解析対象の勤務形態の文字列
//
// return:
//
//	model.WorkplaceType: 解析された勤務形態
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

// ParseBenefitsは、福利厚生に関する文字列を解析し、キーワードに基づいてmodel.Benefits構造体に変換します。
//
// args:
//
//	benefitsStr: 解析対象の福利厚生の文字列
//
// return:
//
//	model.Benefits: 解析された福利厚生情報
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

// ParseLocationは、所在地の文字列を解析し、都道府県コード、市区町村などを含むmodel.Locationオブジェクトを返します。
//
// args:
//
//	locationStr: 解析対象の所在地の文字列
//
// return:
//
//	model.Location: 解析された所在地情報
//	error         : 都道府県名の特定に失敗した場合などのエラー
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
	match := p.patterns.LocationPattern.FindStringSubmatch(locationStr)
	if len(match) >= 2 {
		city = p.trimPunctuation(match[1])
	}

	return model.NewLocation(code, name, city, locationStr), nil
}

// normalizeStringは、文字列の正規化（全角記号・数字の半角化、トリムなど）を行います。
//
// args:
//
//	s: 正規化対象の文字列
//
// return:
//
//	string: 正規化後の文字列
func (p *jobPostingParser) normalizeString(s string) string {
	// 全角英数字・記号・カタカナなどを半角に変換
	s = width.Narrow.String(s)

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

// trimPunctuationは、文字列の先頭と末尾から句読点や記号を削除します。
//
// args:
//
//	s: 処理対象の文字列
//
// return:
//
//	string: 句読点と記号がトリムされた文字列
func (p *jobPostingParser) trimPunctuation(s string) string {
	return strings.TrimFunc(s, func(r rune) bool {
		return unicode.IsPunct(r) || unicode.IsSymbol(r)
	})
}
