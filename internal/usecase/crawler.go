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
	u.logger.Info("Starting prefecture crawler with base_url=%s strategy=%s", u.cfg.BaseURL, u.cfg.Strategy)

	// ベースURLに遷移
	if err := u.client.Navigate(ctx, u.cfg.BaseURL); err != nil {
		u.logger.Error("Failed to navigate to base URL %s: %v", u.cfg.BaseURL, err)
		return fmt.Errorf("failed to navigate to %s: %w", u.cfg.BaseURL, err)
	}

	// 都道府県リンクを抽出
	prefectureLinks, err := u.client.ExtractAttribute(u.cfg.Selector.PrefectureLinkSelector, "href")
	if err != nil {
		u.logger.Error("Failed to extract prefecture links with selector %s: %v", u.cfg.Selector.PrefectureLinkSelector, err)
		return fmt.Errorf("failed to extract prefecture links: %w", err)
	}

	u.logger.Info("Found %d prefecture links", len(prefectureLinks))

	successCount := 0
	// 都道府県リンクの処理
	for i, prefectureLink := range prefectureLinks {
		// BaseURLを基準にしてリンクを解決
		resolvedLink, err := u.resolveURL(u.cfg.BaseURL, prefectureLink)
		if err != nil {
			u.logger.Error("Failed to resolve prefecture link %s: %v", prefectureLink, err)
			continue
		}

		u.logger.Info("Processing prefecture %d/%d: %s", i+1, len(prefectureLinks), resolvedLink)

		time.Sleep(time.Duration(u.cfg.SleepSeconds) * time.Second)

		ctxTimeout, cancel := context.WithTimeout(ctx, time.Duration(u.cfg.TimeoutSeconds)*time.Second)
		defer cancel()

		var procErr error
		for retry := 0; retry < u.cfg.RetryCount; retry++ {
			procErr = u.processPrefecture(ctxTimeout, resolvedLink)
			if procErr == nil {
				break
			}
			u.logger.Warn("Retry %d/%d: %v", retry+1, u.cfg.RetryCount, procErr)
			time.Sleep(1 * time.Second)
		}

		if procErr != nil {
			u.logger.Error("Failed to process prefecture %d %s: %v", i+1, resolvedLink, procErr)
			continue
		}
		successCount++
	}

	u.logger.Info("Prefecture crawler completed: %d/%d successful", successCount, len(prefectureLinks))

	if successCount == 0 {
		return fmt.Errorf("all prefecture processing failed")
	}

	return nil
}

// resolveURLは、与えられたURLをベースURLに対して解決し、絶対URLを返します。
// targetURLが既に絶対URLであればそれを返し、相対URLであればベースURLに解決します。
func (u *PrefectureCrawlerUseCase) resolveURL(baseURL, targetURL string) (string, error) {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse target URL %s: %w", targetURL, err)
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL %s: %w", u.cfg.BaseURL, err)
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
		return fmt.Errorf("failed to navigate to prefecture page %s: %w", prefectureLink, err)
	}

	jobCount, err := u.createCrawlJobsByStrategy(ctx)
	if err != nil {
		return fmt.Errorf("failed to create crawl jobs for %s: %w", prefectureLink, err)
	}

	u.logger.Info("Created %d crawl jobs for prefecture: %s", jobCount, prefectureLink)

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
		return 0, fmt.Errorf("unsupported crawl strategy: %s", u.cfg.Strategy)
	}
}

// createJobsByNextLinkは、次へのリンクを辿る戦略でクロールジョブを作成します。
// ページネーションセレクタが存在する限り、詳細リンクを抽出し、ジョブを作成し、次のページへ遷移します。
func (u *PrefectureCrawlerUseCase) createJobsByNextLink(ctx context.Context) (int, error) {
	jobCount := 0
	pageNum := 1

	for {
		u.logger.Info("Processing page %d", pageNum)

		// 現在のURLを取得
		currentURL, err := u.client.GetCurrentURL(ctx)
		if err != nil {
			u.logger.Error("Failed to get current URL on page %d: %v", pageNum, err)
			return jobCount, fmt.Errorf("failed to get current URL on page %d: %w", pageNum, err)
		}

		links, err := u.client.ExtractAttribute(u.cfg.Selector.JobLinkSelector, "href")
		if err != nil {
			u.logger.Error("Failed to extract detail links on page %d: %v", pageNum, err)
			return jobCount, fmt.Errorf("failed to extract detail links on page %d: %w", pageNum, err)
		}

		pageJobCount := 0
		// 求人詳細リンクの処理
		for _, link := range links {
			// 現在のURLを基準にしてリンクを解決
			resolvedURL, err := u.resolveURL(currentURL.String(), link)
			if err != nil {
				u.logger.Warn("Failed to resolve URL %s on page %d: %v", link, pageNum, err)
				continue
			}

			u.logger.Info("Found job detail link %s", resolvedURL)

			if err := u.createCrawlJobByURL(ctx, resolvedURL); err != nil {
				u.logger.Warn("Failed to create crawl job for URL %s on page %d: %v", resolvedURL, pageNum, err)
				continue
			}
			pageJobCount++
		}

		jobCount += pageJobCount
		u.logger.Info("Created %d jobs for page %d", pageJobCount, pageNum)

		// 次のページボタンが存在するか確認
		if !u.client.Exists(u.cfg.Selector.NextPageSelector) {
			u.logger.Info("No next page button found on page %d, stopping pagination.", pageNum)
			break // 次のページがないためループを終了
		}

		// 次のページボタンをクリック
		if err := u.client.Click(ctx, u.cfg.Selector.NextPageSelector); err != nil {
			u.logger.Error("Failed to click next page on page %d: %v", pageNum, err)
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
		return 0, fmt.Errorf("failed to extract total count text: %w", err)
	}

	if len(texts) != 1 {
		return 0, fmt.Errorf("failed to extract total count text: %w", err)
	}

	totalCount, err := u.extractTotalCount(texts[0])
	if err != nil {
		return 0, fmt.Errorf("failed to extract total count: %w", err)
	}

	u.logger.Info("Extracted total count: %d from text: %s", totalCount, texts[0])

	pageSize := u.cfg.Pagination.PerPage
	pageCount := (totalCount + pageSize - 1) / pageSize // 切り上げ計算

	topListURL, err := u.client.GetCurrentURL(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get current URL: %w", err)
	}

	// 最初のページを正規化したURLを構築 (dynamicなpathやqueryの箇所を排除した形)
	baseURL := u.normalizeToPageOneURL(topListURL.String())
	jobCount := 0
	for page := u.cfg.Pagination.Start; page <= pageCount; page++ {
		pageURL, err := u.buildPaginatedURL(baseURL, page)
		if err != nil {
			u.logger.Error("Failed to build paginated URL for page %d from base %s: %v", page, baseURL, err)
			continue
		}

		resolvedURL, err := u.resolveURL(u.cfg.BaseURL, pageURL)
		if err != nil {
			u.logger.Warn("Failed to resolve paginated URL %s: %v", pageURL, err)
			continue
		}

		if err := u.createCrawlJobByURL(ctx, resolvedURL); err != nil {
			u.logger.Warn("Failed to create crawl job for page %d URL %s: %v", page, resolvedURL, err)
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
		return 0, fmt.Errorf("no number found in total count text: %s", text)
	}

	// 抽出した文字列からカンマを除去
	cleanedMatch := strings.ReplaceAll(match, ",", "")

	totalCount, err := strconv.Atoi(cleanedMatch)
	if err != nil {
		return 0, fmt.Errorf("failed to convert total count to int: %w, text: %s", err, cleanedMatch)
	}

	return totalCount, nil
}

// createCrawlJobByURLは、指定されたURLからCrawlJobを作成し、リポジトリに保存します。
func (u *PrefectureCrawlerUseCase) createCrawlJobByURL(ctx context.Context, link string) error {
	uParsed, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("failed to parse URL %s: %w", link, err)
	}

	job := model.CrawlJob{
		ID:     uuid.New(),
		Status: model.CrawlJobStatusPending,
		URL:    *uParsed,
	}

	// Redisのキー生成ロジックにより、同じURLに対しては同じキーが生成されるため、
	if err := u.repo.Save(ctx, job); err != nil {
		return fmt.Errorf("failed to save crawl job: %w", err)
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
		return "", fmt.Errorf("unsupported pagination type: %s", u.cfg.Pagination.Type)
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

// CrawlJobExecutorUseCaseは、CrawlJobを実行し、HTMLを保存するユースケースです。
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
	u.logger.Info("Starting crawl job executor")

	// PENDING状態のCrawlJobをループで取得し続ける
	for {
		size := 100 // 一度に取得するジョブの数
		jobs, err := u.repo.FindListByStatus(ctx, size, model.CrawlJobStatusPending)
		if err != nil {
			u.logger.Error("Failed to find pending crawl jobs: %v", err)
			return fmt.Errorf("failed to find pending crawl jobs: %w", err)
		}

		if len(jobs) == 0 {
			u.logger.Info("No pending crawl jobs found, stopping executor")
			break // 処理すべきジョブがなければループを終了
		}

		u.logger.Info("Found %d pending crawl jobs", len(jobs))

		successCount := 0
		// pending job　を処理
		for i, job := range jobs {
			u.logger.Info("Processing crawl job %d/%d: %s %s", i+1, len(jobs), job.ID.String(), job.URL.String())

			if err := u.processCrawlJob(ctx, job); err != nil {
				u.logger.Error("Failed to process crawl job %s %s: %v", job.ID.String(), job.URL.String(), err)

				newJob := model.CrawlJob{
					ID:     job.ID,
					URL:    job.URL,
					Status: model.CrawlJobStatusFailed,
				}

				// ジョブのステータスをFAILEDに更新
				if err := u.repo.Save(ctx, newJob); err != nil {
					// ここで保存に失敗した場合、ログを出力して処理を続行するか、中断する
					u.logger.Error("Failed to save job status to FAILED for %s: %v", job.ID.String(), err)
					break
				}

				continue
			}

			successCount++
		}
		u.logger.Info("Crawl job executor completed: %d/%d successful in this batch", successCount, len(jobs))

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
		return fmt.Errorf("failed to navigate to URL %s: %w", job.URL.String(), err)
	}

	// HTMLを取得
	html, err := u.client.GetHTML(ctx)
	if err != nil {
		return fmt.Errorf("failed to get HTML: %w", err)
	}

	// HTMLを保存（実装は要件に応じて）
	if err := u.client.SaveHTML(ctx, job.ID.String(), html); err != nil {
		return fmt.Errorf("failed to save HTML: %w", err)
	}

	// TODO: transaction
	// Note: ジョブの削除とステータス更新はアトミックに行われるべきです。
	// 現在は、削除が成功してもステータス更新が失敗する可能性があるため、トランザクション管理を検討してください。
	if err := u.repo.Delete(ctx, job); err != nil {
		return fmt.Errorf("failed to delete processed crawl job from redis: %w", err)
	}

	newJob := model.CrawlJob{
		ID:     job.ID,
		URL:    job.URL,
		Status: model.CrawlJobStatusSuccess,
	}

	// ジョブのステータスをSUCCESSに更新
	if err := u.repo.Save(ctx, newJob); err != nil {
		return fmt.Errorf("failed to update job status to SUCCESS: %w", err)
	}

	u.logger.Info("Successfully processed crawl job %s %s (HTML size: %d bytes)", job.ID.String(), job.URL.String(), len(html))

	return nil
}
