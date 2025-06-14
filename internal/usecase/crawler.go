package usecase

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/domain/repository"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
)

// CrawlerUseCaseは、クローラーの実行ロジックを定義するインターフェースです。
type CrawlerUseCase interface {
	Run(ctx context.Context) error
}

// CrawlerArgsは、クローラーユースケースを構築するための引数を保持します。
type CrawlerArgs struct {
	Cfg    *config.CrawlerConfig
	Client infra.BrowserClient
	Repo   repository.CrawlJobRepository
	Logger logger.AppLogger
}

// PrefectureCrawlerUseCaseは、都道府県ページをクロールし、CrawlJobを作成するユースケースです。
type PrefectureCrawlerUseCase struct {
	cfg    *config.CrawlerConfig
	client infra.BrowserClient
	repo   repository.CrawlJobRepository
	logger logger.AppLogger
}

// NewPrefectureCrawlerUseCaseは、PrefectureCrawlerUseCaseの新しいインスタンスを作成します。
func NewPrefectureCrawlerUseCase(args CrawlerArgs) CrawlerUseCase {
	return &PrefectureCrawlerUseCase{
		cfg:    args.Cfg,
		client: args.Client,
		repo:   args.Repo,
		logger: args.Logger,
	}
}

// Runは、都道府県クローラーのメイン実行ロジックです。
// ベースURLにナビゲートし、都道府県リンクを抽出し、それぞれの都道府県を処理します。
func (u *PrefectureCrawlerUseCase) Run(ctx context.Context) error {
	u.logger.Info("都道府県クローラーを開始します: ベースURL=%s, 戦略=%s", u.cfg.BaseURL, u.cfg.Strategy)

	// ベースURLに遷移
	var prefectureLinks []string

	switch u.cfg.Mode {
	case config.Manual:
		prefectureLinks = u.cfg.Urls
	default:
		if err := u.client.Navigate(ctx, u.cfg.BaseURL); err != nil {
			u.logger.Error("ベースURLへのナビゲーションに失敗しました: %s, エラー: %v", u.cfg.BaseURL, err)
			return fmt.Errorf("ベースURL %s へのナビゲーションに失敗しました: %w", u.cfg.BaseURL, err)
		}
		links, err := u.client.ExtractAttribute(u.cfg.Selector.PrefectureLinkSelector, "href")
		prefectureLinks = links
		if err != nil {
			u.logger.Error("都道府県リンクの抽出に失敗しました: セレクター=%s, エラー: %v", u.cfg.Selector.PrefectureLinkSelector, err)
			return fmt.Errorf("都道府県リンクの抽出に失敗しました: %w", err)
		}
	}

	// 都道府県リンクを抽出
	u.logger.Info("都道府県リンクを%d件見つけました", len(prefectureLinks))

	successCount := 0
	// 都道府県リンクの処理
	for i, prefectureLink := range prefectureLinks {
		// BaseURLを基準にしてリンクを解決
		resolvedLink, err := u.resolveURL(u.cfg.BaseURL, prefectureLink)
		if err != nil {
			u.logger.Error("都道府県リンクの解決に失敗しました: %s, エラー: %v", prefectureLink, err)
			continue
		}

		u.logger.Info("都道府県を処理中: %d/%d, リンク: %s", i+1, len(prefectureLinks), resolvedLink)

		time.Sleep(time.Duration(u.cfg.SleepSeconds) * time.Second)

		ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(u.cfg.TimeoutSeconds)*time.Second)
		defer cancel()

		var procErr error
		for retry := 0; retry < u.cfg.RetryCount; retry++ {
			procErr = u.processPrefecture(ctxTimeout, resolvedLink)
			if procErr == nil {
				break
			}
			u.logger.Warn("リトライ中: %d/%d, エラー: %v", retry+1, u.cfg.RetryCount, procErr)
			time.Sleep(1 * time.Second)
		}

		if procErr != nil {
			u.logger.Error("都道府県の処理に失敗しました: %d, リンク: %s, エラー: %v", i+1, resolvedLink, procErr)
			continue
		}
		successCount++
	}

	u.logger.Info("都道府県クローラーが完了しました: 成功数=%d/%d", successCount, len(prefectureLinks))

	if successCount == 0 {
		return fmt.Errorf("すべての都道府県の処理に失敗しました")
	}

	return nil
}

// resolveURLは、与えられたURLをベースURLに対して解決し、絶対URLを返します。
// targetURLが既に絶対URLであればそれを返し、相対URLであればベースURLに解決します。
func (u *PrefectureCrawlerUseCase) resolveURL(baseURL, targetURL string) (string, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("ターゲットURL %s のパースに失敗しました: %w", targetURL, err)
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("ベースURL %s のパースに失敗しました: %w", u.cfg.BaseURL, err)
	}

	if parsed.IsAbs() {
		return parsed.String(), nil
	}

	resolved := parsedBase.ResolveReference(parsed)
	return resolved.String(), nil
}

// processPrefectureは、指定された都道府県リンクのページにナビゲートし、
// 設定された戦略に基づいてクロールジョブを作成します。
func (u *PrefectureCrawlerUseCase) processPrefecture(ctx context.Context, prefectureLink string) error {
	// prefectureLinkは絶対URLの文字列として渡される
	if err := u.client.Navigate(ctx, prefectureLink); err != nil {
		return fmt.Errorf("都道府県ページ %s へのナビゲートに失敗しました: %w", prefectureLink, err)
	}

	jobCount, err := u.createCrawlJobsByStrategy(ctx)
	if err != nil {
		return fmt.Errorf("都道府県 %s のクロールジョブ作成に失敗しました: %w", prefectureLink, err)
	}

	u.logger.Info("求人情報を%d件作成しました: 都道府県リンク: %s", jobCount, prefectureLink)

	return nil
}

// createCrawlJobsByStrategyは、設定されたクロール戦略に基づいてクロールジョブを作成します。
func (u *PrefectureCrawlerUseCase) createCrawlJobsByStrategy(ctx context.Context) (int, error) {
	switch u.cfg.Strategy {
	case config.CrawlByNextLink:
		return u.createJobsByNextLink(ctx)
	case config.CrawlByTotalCount:
		return u.createJobsByTotalCount(ctx)
	// yaml loadの際にvalidation あるので基本的にはdefaultにならない
	default:
		return 0, fmt.Errorf("サポートされていないクロール戦略です: %s", u.cfg.Strategy)
	}
}

// createJobsByNextLinkは、次へのリンクを辿る戦略でクロールジョブを作成します。
// ページネーションセレクタが存在する限り、詳細リンクを抽出し、ジョブを作成し、次のページへ遷移します。
func (u *PrefectureCrawlerUseCase) createJobsByNextLink(ctx context.Context) (int, error) {
	jobCount := 0
	pageNum := 1

	for {
		u.logger.Info("ページ%dを処理中", pageNum)

		// 現在のURLを取得
		currentURL, err := u.client.GetCurrentURL(ctx)
		if err != nil {
			u.logger.Error("ページ%dで現在のURLの取得に失敗しました: エラー: %v", pageNum, err)
			return jobCount, fmt.Errorf("ページ%dで現在のURLの取得に失敗しました: %w", pageNum, err)
		}

		links, err := u.client.ExtractAttribute(u.cfg.Selector.JobLinkSelector, "href")
		if err != nil {
			u.logger.Error("ページ%dで詳細リンクの抽出に失敗しました: エラー: %v", pageNum, err)
			return jobCount, fmt.Errorf("ページ%dで詳細リンクの抽出に失敗しました: %w", pageNum, err)
		}

		pageJobCount := 0
		// 求人詳細リンクの処理
		for _, link := range links {
			// 現在のURLを基準にしてリンクを解決
			resolvedURL, err := u.resolveURL(currentURL.String(), link)
			if err != nil {
				u.logger.Warn("ページ%dでURL %sの解決に失敗しました: エラー: %v", pageNum, link, err)
				continue
			}

			u.logger.Info("求人詳細リンクが見つかりました: %s", resolvedURL)

			if err := u.createCrawlJobByURL(ctx, resolvedURL); err != nil {
				u.logger.Warn("ページ%dでURL %sのクロールジョブの作成に失敗しました: エラー: %v", pageNum, resolvedURL, err)
				continue
			}
			pageJobCount++
		}

		jobCount += pageJobCount
		u.logger.Info("ページ%dで%d件のジョブを作成しました", pageNum, pageJobCount)

		// 次のページボタンが存在するか確認
		if !u.client.Exists(u.cfg.Selector.NextPageSelector) {
			u.logger.Info("ページ%dに次のページボタンが見つかりませんでした。ページネーションを停止します。", pageNum)
			break // 次のページがないためループを終了
		}

		// 次のページボタンをクリック
		if err := u.client.Click(ctx, u.cfg.Selector.NextPageSelector); err != nil {
			u.logger.Error("ページ%dで次のページボタンのクリックに失敗しました: エラー: %v", pageNum, err)
			break // クリックに失敗したためループを終了
		}
		pageNum++
	}

	return jobCount, nil
}

// createJobsByTotalCountは、総件数からページ数を計算し、ページネーションURLを構築してクロールジョブを作成します。
func (u *PrefectureCrawlerUseCase) createJobsByTotalCount(ctx context.Context) (int, error) {
	texts, err := u.client.ExtractText(u.cfg.Selector.TotalCountSelector)
	if err != nil {
		return 0, fmt.Errorf("合計件数テキストの抽出に失敗しました: %w", err)
	}

	if len(texts) != 1 {
		return 0, fmt.Errorf("合計件数テキストの抽出に失敗しました: %w", err)
	}

	totalCount, err := u.extractTotalCount(texts[0])
	if err != nil {
		return 0, fmt.Errorf("合計件数の抽出に失敗しました: %w", err)
	}

	u.logger.Info("総件数を抽出しました: %d (テキスト: %s)", totalCount, texts[0])

	pageSize := u.cfg.Pagination.PerPage
	pageCount := (totalCount + pageSize - 1) / pageSize // 切り上げ計算

	topListURL, err := u.client.GetCurrentURL(ctx)
	if err != nil {
		return 0, fmt.Errorf("現在のURLの取得に失敗しました: %w", err)
	}

	// 最初のページを正規化したURLを構築 (dynamicなpathやqueryの箇所を排除した形)
	baseURL := u.normalizeToPageOneURL(topListURL.String())
	jobCount := 0
	for page := u.cfg.Pagination.Start; page <= pageCount; page++ {
		pageURL, err := u.buildPaginatedURL(baseURL, page)
		if err != nil {
			u.logger.Error("ページ%dのページネーションURL構築に失敗しました: ベース=%s, エラー: %v", page, baseURL, err)
			continue
		}

		resolvedURL, err := u.resolveURL(u.cfg.BaseURL, pageURL)
		if err != nil {
			u.logger.Warn("ページネーションURL %s の解決に失敗しました: エラー: %v", pageURL, err)
			continue
		}

		if err := u.createCrawlJobByURL(ctx, resolvedURL); err != nil {
			u.logger.Warn("ページ%dのURL %s のクロールジョブ作成に失敗しました: エラー: %v", page, resolvedURL, err)
			continue
		}
		jobCount++
	}
	return jobCount, nil
}

// extractTotalCountは、テキストから合計件数を表す数値を正規表現で抽出し、カンマを除去して返します。
func (u *PrefectureCrawlerUseCase) extractTotalCount(text string) (int, error) {
	// 数字とカンマにマッチする正規表現。例: "1,234件" から "1,234" を抽出。
	re := regexp.MustCompile(`[0-9,]+`)
	match := re.FindString(text)
	if match == "" {
		return 0, fmt.Errorf("合計件数テキストから数値が見つかりませんでした: %s", text)
	}

	// 抽出した文字列からカンマを除去
	cleanedMatch := strings.ReplaceAll(match, ",", "")

	totalCount, err := strconv.Atoi(cleanedMatch)
	if err != nil {
		return 0, fmt.Errorf("合計件数の整数変換に失敗しました: %w, テキスト: %s", err, cleanedMatch)
	}

	return totalCount, nil
}

// createCrawlJobByURLは、指定されたURLからCrawlJobを作成し、リポジトリに保存します。
func (u *PrefectureCrawlerUseCase) createCrawlJobByURL(ctx context.Context, link string) error {
	uParsed, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("URL %s のパースに失敗しました: %w", link, err)
	}

	job := model.CrawlJob{
		ID:     uuid.New(),
		Status: model.CrawlJobStatusPending,
		URL:    *uParsed,
	}

	// Redisのキー生成ロジックにより、同じURLに対しては同じキーが生成されるため、
	if err := u.repo.Save(ctx, job); err != nil {
		return fmt.Errorf("クロールジョブの保存に失敗しました: %w", err)
	}

	return nil
}

// buildPaginatedURLは、ベースURLとページ番号に基づいてページネーションされたURLを構築します。
// 設定されたページネーションタイプ（クエリパラメータ、パス、セグメント）に応じてURLを生成します。
func (u *PrefectureCrawlerUseCase) buildPaginatedURL(baseURL string, page int) (string, error) {
	uParsed, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	switch u.cfg.Pagination.Type {
	case config.Query:
		// 例: /jobs?page=3 のようなクエリパラメータに対応
		q := uParsed.Query()
		q.Set(u.cfg.Pagination.ParamIdentifier, strconv.Itoa(page))
		uParsed.RawQuery = q.Encode()
		return uParsed.String(), nil

	case config.Path:
		// 例: /jobs/page/3 のようなパス構成に対応
		// path.Join を使用して安全にパスを構築します。
		pageStr := fmt.Sprintf(u.cfg.Pagination.PageFormat, page)
		uParsed.Path = path.Join(uParsed.Path, u.cfg.Pagination.ParamIdentifier, pageStr)
		return uParsed.String(), nil

	case config.Segment:
		// 例: /p1 や /page3 のようにパス末尾にセグメントで埋め込む
		// `path.Join` を使うと `/base/param/1` のようになるため、直接文字列結合します。
		// ただし、baseURLのパスが既に末尾にスラッシュを持っている場合は、
		// 重複を避けるためにスラッシュを除去します。
		trimmedPath := strings.TrimSuffix(uParsed.Path, "/")
		pageStr := fmt.Sprintf(u.cfg.Pagination.PageFormat, page)
		uParsed.Path = fmt.Sprintf("%s/%s%s", trimmedPath, u.cfg.Pagination.ParamIdentifier, pageStr)
		return uParsed.String(), nil

	case config.None:
		// ページネーションなし（通常1ページのみ対象）
		return baseURL, nil

	default:
		return "", fmt.Errorf("サポートされていないページネーションタイプです: %s", u.cfg.Pagination.Type)
	}
}

// normalizeToPageOneURLは、現在のURLをページネーションの最初のページ（またはページネーションなし）のURLに正規化します。
// クエリパラメータやパスセグメントのページ番号を除去します。
func (u *PrefectureCrawlerUseCase) normalizeToPageOneURL(rawURL string) string {
	uParsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // パース失敗時はそのまま返す
	}

	switch u.cfg.Pagination.Type {
	case config.Path:
		// 例: /list/page/1 -> /list/
		// パスが `/ParamIdentifier/数値` の形式で終わる場合に、その部分を削除し、基準のパスに戻します。
		// PageFormatを考慮すると正規表現が複雑になるため、ParamIdentifierのみでマッチング
		re := regexp.MustCompile(`/` + regexp.QuoteMeta(u.cfg.Pagination.ParamIdentifier) + `/\d+$`)
		uParsed.Path = re.ReplaceAllString(uParsed.Path, "/")
		return uParsed.String()

	case config.Segment:
		// 例: /list/page1 -> /list/
		// パスが `/ParamIdentifier数値` の形式で終わる場合に、その部分を削除し、基準のパスに戻します。
		// PageFormatを考慮すると正規表現が複雑になるため、ParamIdentifierのみでマッチング
		re := regexp.MustCompile(`/` + regexp.QuoteMeta(u.cfg.Pagination.ParamIdentifier) + `\d+$`)
		uParsed.Path = re.ReplaceAllString(uParsed.Path, "/")
		return uParsed.String()

	case config.Query:
		// 例: ?p=1 -> クエリから除去
		q := uParsed.Query()
		q.Del(u.cfg.Pagination.ParamIdentifier)
		uParsed.RawQuery = q.Encode()
		return uParsed.String()

	default:
		// それ以外は変更せず返す
		return rawURL
	}
}

// CrawlJobExecutorUseCaseは、RedisからCrawlJobを消費し、ブラウザで実行するユースケースです。
type CrawlJobExecutorUseCase struct {
	cfg    *config.CrawlerConfig
	client infra.BrowserClient
	repo   repository.CrawlJobRepository
	logger logger.AppLogger
}

// NewCrawlJobExecutorUseCaseは、CrawlJobExecutorUseCaseの新しいインスタンスを作成します。
func NewCrawlJobExecutorUseCase(args CrawlerArgs) CrawlerUseCase {
	return &CrawlJobExecutorUseCase{
		cfg:    args.Cfg,
		client: args.Client,
		repo:   args.Repo,
		logger: args.Logger,
	}
}

// Runは、CrawlJobExecutorUseCaseのメイン実行ロジックです。
// PENDING状態のCrawlJobを定期的に取得し、処理します。
func (u *CrawlJobExecutorUseCase) Run(ctx context.Context) error {
	u.logger.Info("クロールジョブ実行クローラーを開始します")

	// PENDING状態のCrawlJobをループで取得し続ける
	for {
		size := 100 // 一度に取得するジョブの数
		jobs, err := u.repo.FindListByStatus(ctx, size, model.CrawlJobStatusPending)
		if err != nil {
			u.logger.Error("保留中のクロールジョブの検索に失敗しました: %v", err)
			return fmt.Errorf("保留中のクロールジョブの検索に失敗しました: %w", err)
		}

		if len(jobs) == 0 {
			u.logger.Info("保留中のクロールジョブが見つかりませんでした。エグゼキューターを停止します。")
			break // 処理すべきジョブがなければループを終了
		}

		u.logger.Info("保留中のクロールジョブを%d件見つけました", len(jobs))

		successCount := 0
		// pending job　を処理
		for i, job := range jobs {
			u.logger.Info("クロールジョブを処理中: %d/%d, ID: %s, URL: %s", i+1, len(jobs), job.ID.String(), job.URL.String())

			if err := u.processCrawlJob(ctx, job); err != nil {
				u.logger.Error("クロールジョブの処理に失敗しました: ID: %s, URL: %s, エラー: %v", job.ID.String(), job.URL.String(), err)

				newJob := model.CrawlJob{
					ID:     job.ID,
					URL:    job.URL,
					Status: model.CrawlJobStatusFailed,
				}

				// ジョブのステータスをFAILEDに更新
				if err := u.repo.Save(ctx, newJob); err != nil {
					// ここで保存に失敗した場合、ログを出力して処理を続行するか、中断する
					u.logger.Error("ジョブのステータスをFAILEDに保存できませんでした: ID: %s, エラー: %v", job.ID.String(), err)
					break
				}

				continue
			}

			successCount++
		}
		u.logger.Info("クロールジョブエグゼキューターが完了しました: このバッチで成功したジョブ数=%d/%d", successCount, len(jobs))

		// 短い間隔を置いて次のバッチをフェッチ
		time.Sleep(1 * time.Second) // 必要に応じて調整
	}

	return nil
}

// processCrawlJobは、単一のCrawlJobを処理します。
// URLにナビゲートし、HTMLを取得して保存し、ジョブのステータスをSUCCESSに更新します。
func (u *CrawlJobExecutorUseCase) processCrawlJob(ctx context.Context, job model.CrawlJob) error {
	// URLに遷移
	if err := u.client.Navigate(ctx, job.URL.String()); err != nil {
		return fmt.Errorf("URL %s へのナビゲートに失敗しました: %w", job.URL.String(), err)
	}

	// HTMLを取得
	html, err := u.client.GetHTML(ctx)
	if err != nil {
		return fmt.Errorf("HTMLの取得に失敗しました: %w", err)
	}

	// HTMLを保存
	if err := u.client.SaveHTML(ctx, job.ID.String()+".html", html); err != nil {
		return fmt.Errorf("HTMLの保存に失敗しました: %w", err)
	}

	// Note: ジョブの削除とステータス更新はアトミックに行われるべきです。
	// 現在は、削除が成功してもステータス更新が失敗する可能性があるため、トランザクション管理を検討してください。
	if err := u.repo.Delete(ctx, job); err != nil {
		return fmt.Errorf("処理済みクロールジョブのRedisからの削除に失敗しました: %w", err)
	}

	newJob := model.CrawlJob{
		ID:     job.ID,
		URL:    job.URL,
		Status: model.CrawlJobStatusSuccess,
	}

	// ジョブのステータスをSUCCESSに更新
	if err := u.repo.Save(ctx, newJob); err != nil {
		return fmt.Errorf("ジョブのステータスをSUCCESSに更新できませんでした: %w", err)
	}

	u.logger.Info("クロールジョブの処理に成功しました: ID: %s, URL: %s (HTMLサイズ: %dバイト)", job.ID.String(), job.URL.String(), len(html))

	return nil
}
