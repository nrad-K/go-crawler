package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/nrad-K/go-crawler/internal/domain/model"
	"github.com/nrad-K/go-crawler/internal/domain/repository"
	"github.com/nrad-K/go-crawler/internal/infra"
	"github.com/nrad-K/go-crawler/internal/logger"
	"golang.org/x/sync/errgroup"
)

// CrawlerUseCaseは、クローラーの実行ロジックを定義するインターフェースです。

// CrawlerArgsは、クローラーユースケースを構築するための引数を保持します。
type CrawlerArgs struct {
	Cfg    *config.CrawlerConfig
	Client infra.BrowserClient
	Repo   repository.CrawlJobRepository
	Logger logger.AppLogger
}

type generateCrawlJobUseCase struct {
	cfg    *config.CrawlerConfig
	client infra.BrowserClient
	repo   repository.CrawlJobRepository
	logger logger.AppLogger
}

func NewGenerateCrawlJobUseCase(args CrawlerArgs) *generateCrawlJobUseCase {
	return &generateCrawlJobUseCase{
		cfg:    args.Cfg,
		client: args.Client,
		repo:   args.Repo,
		logger: args.Logger,
	}
}

const (
	maxListLinks = 100
	batchSize    = 100
)

// Runは、クローラーのメイン実行ロジックです。
func (u *generateCrawlJobUseCase) GenerateCrawlJob(ctx context.Context) error {
	u.logger.Info("クローラーの実行を開始します: ベースURL=%s, Strategy=%s", u.cfg.BaseURL, u.cfg.Strategy)

	// ベースURLに遷移
	listLinks := u.listLinksByMode()

	if len(listLinks) == 0 {
		u.logger.Error("一覧ページのリンクが見つかりませんでした")
		return fmt.Errorf("一覧ページのリンクが見つかりませんでした")
	}

	// 一覧ページのリンクを抽出
	u.logger.Info("一覧ページのリンクを%d件見つけました", len(listLinks))

	// 一覧リンクの処理
	for i, link := range listLinks {
		// BaseURLを基準にしてリンクを解決
		resolvedLink, err := u.resolveURL(u.cfg.BaseURL, link)
		if err != nil {
			u.logger.Error("ぺージネーションページのリンクの解決に失敗しました: %s, エラー: %v", link, err)
			continue
		}

		u.logger.Info("一覧ページのリンクを処理中: %d/%d, リンク: %s", i+1, len(listLinks), resolvedLink)

		if err := u.processListLink(ctx, resolvedLink); err != nil {
			u.logger.Error("一覧ページのリンクの処理に失敗しました: %d, リンク: %s, エラー: %v", i+1, resolvedLink, err)
			continue
		}

		time.Sleep(time.Duration(u.cfg.CrawlSleepSeconds) * time.Second)
	}

	u.logger.Info("クローラーの実行が完了しました: 件数=%d", len(listLinks))
	return nil
}

func (u *generateCrawlJobUseCase) listLinksByMode() []string {
	listLinks := make([]string, 0, 100)

	switch u.cfg.Mode {
	case config.Manual:
		listLinks = u.cfg.Urls
	case config.Auto:
		if err := u.client.Navigate(u.cfg.BaseURL); err != nil {
			u.logger.Error("べースURLへのナビゲーションに失敗しました: %s, エラー: %v", u.cfg.BaseURL, err)
			return listLinks
		}

		links, err := u.client.ExtractAttribute(u.cfg.Selector.ListLinksSelector, "href")
		if err != nil {
			u.logger.Error("一覧ページのリンクの抽出に失敗しました: セレクター=%s, エラー: %v", u.cfg.Selector.ListLinksSelector, err)
			return listLinks
		}

		listLinks = links
	default:
		u.logger.Error("サポートされていないモードです")
		return listLinks
	}

	u.logger.Info("listLinksByMode: %d件のリンクを取得: %v", len(listLinks), listLinks)
	return listLinks
}

// resolveURLは、与えられたURLをベースURLに対して解決し、絶対URLを返します。
// targetURLが既に絶対URLであればそれを返し、相対URLであればベースURLに解決します。
func (u *generateCrawlJobUseCase) resolveURL(baseURL, targetURL string) (string, error) {
	parsedTarget, err := url.Parse(targetURL)
	if err != nil {
		return "", fmt.Errorf("ターゲットURL %s のパースに失敗しました: %w", targetURL, err)
	}

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("ベースURL %s のパースに失敗しました: %w", u.cfg.BaseURL, err)
	}

	if parsedTarget.IsAbs() {
		return parsedTarget.String(), nil
	}

	resolved := parsedBase.ResolveReference(parsedTarget)
	return resolved.String(), nil
}

// 設定された戦略に基づいてクロールジョブを作成します。
func (u *generateCrawlJobUseCase) processListLink(ctx context.Context, link string) error {
	if err := u.client.Navigate(link); err != nil {
		return fmt.Errorf("ぺージネーションページ %s へのナビゲートに失敗しました: %w", link, err)
	}

	jobCount, err := u.createCrawlJobsByStrategy(ctx)
	if err != nil {
		return fmt.Errorf("%s のクロールジョブ作成に失敗しました: %w", link, err)
	}

	u.logger.Info("クロールジョブを%d件作成しました", jobCount)

	return nil
}

// createCrawlJobsByStrategyは、設定されたStrategyに基づいてクロールジョブを作成します。
func (u *generateCrawlJobUseCase) createCrawlJobsByStrategy(ctx context.Context) (int, error) {
	switch u.cfg.Strategy {

	case config.CrawlByNextLink:
		return u.createJobsByNextLink(ctx)

	case config.CrawlByTotalCount:
		return u.createJobsByTotalCount(ctx)

	default:
		return 0, fmt.Errorf("サポートされていないStrategyです: %s", u.cfg.Strategy)
	}
}

// createJobsByNextLinkは、次へのリンクを辿る戦略でクロールジョブを作成します。
// ページネーションセレクタが存在する限り、詳細リンクを抽出し、ジョブを作成し、次のページへ遷移します。
func (u *generateCrawlJobUseCase) createJobsByNextLink(ctx context.Context) (int, error) {
	jobCount := 0
	pageNum := 1

	for {
		u.logger.Info("ページ%dを処理中", pageNum)

		currentURL, err := u.client.CurrentURL()
		if err != nil {
			u.logger.Error("ページ%dで現在のURLの取得に失敗しました: エラー: %v", pageNum, err)
			return jobCount, fmt.Errorf("ページ%dで現在のURLの取得に失敗しました: %w", pageNum, err)
		}

		links, err := u.client.ExtractAttribute(u.cfg.Selector.DetailLinksSelector, "href")
		if err != nil {
			u.logger.Error("ページ%dで詳細ページのリンクの抽出に失敗しました: エラー: %v", pageNum, err)
			return jobCount, fmt.Errorf("ページ%dで詳細リンクの抽出に失敗しました: %w", pageNum, err)
		}

		u.logger.Info("ページ%dで%d件の詳細ページのリンクを抽出しました", pageNum, len(links))

		var pageJobCount int32
		// 求人詳細リンクの処理
		eg, childCtx := errgroup.WithContext(ctx)
		for _, link := range links {
			targetLink := link

			eg.Go(func() error {
				select {
				case <-childCtx.Done():
					u.logger.Warn("コンテキストがキャンセルされたため、ジョブ作成を中断します。")
					return childCtx.Err()
				default:
					// 現在のURLを基準にしてリンクを解決
					var resolvedURL string
					var err error

					switch u.cfg.JobDetailResolveBaseURL {
					case "":
						resolvedURL, err = u.resolveURL(currentURL.String(), targetLink)
					default:
						resolvedURL, err = u.resolveURL(u.cfg.JobDetailResolveBaseURL, targetLink)
					}

					if err != nil {
						u.logger.Warn("ページ%dでURL %sの解決に失敗しました: エラー: %v", pageNum, targetLink, err)
						return nil // エラーを返さずに続行
					}

					u.logger.Info("求人詳細リンクが見つかりました: %s", resolvedURL)

					if err := u.createCrawlJobByURL(ctx, resolvedURL); err != nil {
						u.logger.Warn("ページ%dでURL %sのクロールジョブの作成に失敗しました: エラー: %v", pageNum, resolvedURL, err)
						return nil // エラーを返さずに続行
					}

					atomic.AddInt32(&pageJobCount, 1)
					return nil
				}
			})
		}

		if err := eg.Wait(); err != nil {
			u.logger.Error("並列処理中にエラーが発生しました: %v", err)
			return int(jobCount), fmt.Errorf("ページ%dでの詳細リンク処理中にエラーが発生しました: %w", pageNum, err)
		}

		jobCount += int(pageJobCount)
		u.logger.Info("ページ%dで%d件のジョブを作成しました", pageNum, pageJobCount)

		// 次のページボタンが存在するか確認
		exists, err := u.client.Exists(u.cfg.Selector.NextPageLocator)
		if err != nil {
			u.logger.Error("ページ%dで次のページボタンの存在確認に失敗しました: エラー: %v", pageNum, err)
			return int(jobCount), fmt.Errorf("ページ%dで次のページボタンの存在確認に失敗しました: %w", pageNum, err)
		}

		if !exists {
			u.logger.Info("ページ%dに次のページボタンが見つかりませんでした。ページネーションを停止します。", pageNum)
			return int(jobCount), nil
		}

		// 次のページボタンをクリック
		if err := u.client.Click(u.cfg.Selector.NextPageLocator); err != nil {
			u.logger.Error("ページ%dで次のページボタンのクリックに失敗しました: エラー: %v", pageNum, err)
			return int(jobCount), fmt.Errorf("ページ%dで次のページボタンのクリックに失敗しました: %w", pageNum, err)
		}

		pageNum++
	}
}

// createJobsByTotalCountは、総件数からページ数を計算し、ページネーションURLを構築してクロールジョブを作成します。
func (u *generateCrawlJobUseCase) createJobsByTotalCount(ctx context.Context) (int, error) {
	texts, err := u.client.ExtractText(u.cfg.Selector.TotalCountSelector)
	if err != nil {
		return 0, fmt.Errorf("合計件数テキストの抽出に失敗しました: %w", err)
	}

	if len(texts) == 0 {
		return 0, fmt.Errorf("合計件数テキストが見つかりませんでした")
	}

	if len(texts) > 1 {
		u.logger.Warn("合計件数セレクターに複数の要素がマッチしました。最初の要素を使用します。")
	}

	totalCount, err := u.extractTotalCount(texts[0])
	if err != nil {
		return 0, fmt.Errorf("合計件数の抽出に失敗しました: %w", err)
	}

	u.logger.Info("総件数を抽出しました: %d (テキスト: %s)", totalCount, texts[0])

	pageSize := u.cfg.Pagination.PerPage
	if pageSize == 0 {
		return 0, fmt.Errorf("ページサイズが0です。設定を確認してください。")
	}
	pageCount := (totalCount + pageSize - 1) / pageSize // 切り上げ計算

	topListURL, err := u.client.CurrentURL()
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
func (u *generateCrawlJobUseCase) extractTotalCount(text string) (int, error) {
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
func (u *generateCrawlJobUseCase) createCrawlJobByURL(ctx context.Context, link string) error {
	uParsed, err := url.Parse(link)
	if err != nil {
		return fmt.Errorf("URL %s のパースに失敗しました: %w", link, err)
	}

	job := model.CrawlJob{
		ID:     uuid.New(),
		Status: model.CrawlJobStatusPending,
		URL:    *uParsed,
	}

	isExist, err := u.repo.Exists(ctx, job)
	if err != nil {
		return fmt.Errorf("クロールジョブの存在確認に失敗しました: %w", err)
	}

	if isExist {
		u.logger.Info("既に存在するURLのためスキップします: %s", link)
		return nil
	}

	if err := u.repo.Save(ctx, job); err != nil {
		return fmt.Errorf("クロールジョブの保存に失敗しました: %w", err)
	}

	return nil
}

// buildPaginatedURLは、ベースURLとページ番号に基づいてページネーションされたURLを構築します。
// 設定されたページネーションタイプ（クエリパラメータ、パス、セグメント）に応じてURLを生成します。
func (u *generateCrawlJobUseCase) buildPaginatedURL(baseURL string, page int) (string, error) {
	uParsed, err := url.Parse(baseURL)
	if err != nil {
		u.logger.Error("URLのパースに失敗しました: %s, エラー: %v", baseURL, err)
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
func (u *generateCrawlJobUseCase) normalizeToPageOneURL(rawURL string) string {
	uParsed, err := url.Parse(rawURL)

	if err != nil {
		u.logger.Warn("URLのパースに失敗しました: %s, エラー: %v", rawURL, err)
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
type executeCrawlJobUseCase struct {
	cfg    *config.CrawlerConfig
	client infra.BrowserClient
	repo   repository.CrawlJobRepository
	logger logger.AppLogger
}

// NewCrawlJobExecutorUseCaseは、CrawlJobExecutorUseCaseの新しいインスタンスを作成します。
func NewExecuteCrawlJobUseCase(args CrawlerArgs) *executeCrawlJobUseCase {
	return &executeCrawlJobUseCase{
		cfg:    args.Cfg,
		client: args.Client,
		repo:   args.Repo,
		logger: args.Logger,
	}
}

var (
	ErrNoPendingJobs = errors.New("pending job not found")
)

// Runは、CrawlJobExecutorUseCaseのメイン実行ロジックです。
// PENDING状態のCrawlJobを定期的に取得し、処理します。
func (u *executeCrawlJobUseCase) ExecuteCrawlJob(ctx context.Context) error {
	u.logger.Info("クローラーを開始します")

	var totalProcessedCount int32 // 処理されたジョブの総数

	for {
		jobs, err := u.repo.FindListByStatus(ctx, batchSize, model.CrawlJobStatusPending)
		if err != nil {
			u.logger.Error("保留中のクロールジョブの検索に失敗しました: %v", err)
			break
		}

		if len(jobs) == 0 {
			u.logger.Info("保留中のクロールジョブが見つかりませんでした。処理を終了します。")
			break
		}

		for _, job := range jobs {
			u.processCrawl(ctx, job)
			totalProcessedCount++
		}
		// 短い間隔を置いて次のバッチをフェッチ
		time.Sleep(3 * time.Second) // 必要に応じて調整
	}

	u.logger.Info("クローラーが完了しました。合計処理ジョブ数: %d", totalProcessedCount)
	return nil
}

func (u *executeCrawlJobUseCase) processCrawl(ctx context.Context, job model.CrawlJob) error {
	u.logger.Info("クロールジョブを処理中 ID: %s, URL: %s", job.ID.String(), job.URL.String())

	if u.cfg.Selector.TabClickSelector != "" {
		u.logger.Info("タブをクリックします。セレクター: %s", u.cfg.Selector.TabClickSelector)
		// タブをクリック
		if err := u.client.Click(u.cfg.Selector.TabClickSelector); err != nil {
			u.logger.Error("タブのクリックに失敗しました ID: %s, URL: %s, エラー: %w", job.ID.String(), job.URL.String(), err)
		}
	}
	// HTMLを取得
	html, err := u.client.GetHTML()
	if err != nil {
		u.logger.Error("HTMLの取得に失敗しました ID: %s, URL: %s, エラー: %w", job.ID.String(), job.URL.String(), err)
		return fmt.Errorf("HTMLの取得に失敗しました: %w", err)
	}

	// HTMLを保存
	if err := u.client.SaveHTML(job.ID.String()+".html", html); err != nil {
		u.logger.Error("HTMLの保存に失敗しました ID: %s, URL: %s, エラー: %w", job.ID.String(), job.URL.String(), err)
		return fmt.Errorf("HTMLの保存に失敗しました: %w", err)
	}

	// 現在は、削除が成功してもステータス更新が失敗する可能性があるため、トランザクション管理を検討してください。
	if err := u.repo.Delete(ctx, job); err != nil {
		u.logger.Error("処理済みクロールジョブの削除に失敗しました ID: %s, URL: %s, エラー: %w", job.ID.String(), job.URL.String(), err)
		return fmt.Errorf("クロールジョブの削除に失敗しました: %w", err)
	}

	newJob := model.CrawlJob{
		ID:     job.ID,
		URL:    job.URL,
		Status: model.CrawlJobStatusSuccess,
	}

	// ジョブのステータスをSUCCESSに更新
	if err := u.repo.Save(ctx, newJob); err != nil {
		u.logger.Error("ジョブのステータスをSUCCESSに更新できませんでした ID: %s, URL: %s, エラー: %w", job.ID.String(), job.URL.String(), err)
		return fmt.Errorf("ジョブのステータス更新に失敗しました: %w", err)
	}

	return nil
}
