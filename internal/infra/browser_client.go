package infra

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/playwright-community/playwright-go"
)

// BrowserClientは、クローリングで利用するブラウザ操作のインターフェースです。
type BrowserClient interface {
	Click(selector string) error
	GetHTML() (string, error)
	SaveHTML(filename string, content string) error
	CurrentURL() (*url.URL, error)
	Navigate(url string) error
	ExtractText(selector string) ([]string, error)
	ExtractAttribute(selector, attr string) ([]string, error)
	Exists(selector string) (bool, error)
	Close() error
}

type browserClient struct {
	pw      *playwright.Playwright
	cfg     *config.CrawlerConfig
	browser playwright.Browser
	page    playwright.Page
	context playwright.BrowserContext
}

// NewBrowserClientは、Playwrightを用いたbrowserClientを生成します。
//
// args:
//
//	cfg: クローラー設定
//
// return:
//
//	*browserClient: 生成されたクライアント
//	error: 失敗時のエラー
func NewBrowserClient(cfg *config.CrawlerConfig) (*browserClient, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("playwrightの起動に失敗しました: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(cfg.EnableHeadless),
	})
	if err != nil {
		return nil, fmt.Errorf("ブラウザの起動に失敗しました: %w", err)
	}

	context, err := browser.NewContext(playwright.BrowserNewContextOptions{
		ExtraHttpHeaders: cfg.Headers,
		UserAgent:        &cfg.UserAgent,
	})
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("ブラウザコンテキストの作成に失敗しました: %w", err)
	}

	if err := setupResourceBlocking(context); err != nil {
		return nil, fmt.Errorf("リソースブロックの設定に失敗しました: %w", err)
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, fmt.Errorf("ページの作成に失敗しました: %w", err)
	}

	return &browserClient{
		pw:      pw,
		browser: browser,
		context: context,
		page:    page,
		cfg:     cfg,
	}, nil
}

func setupResourceBlocking(context playwright.BrowserContext) error {
	return context.Route("**/*.{png,jpg,jpeg,gif,svg,woff,woff2,ttf,eot,otf}", func(route playwright.Route) {
		route.Abort()
	})
}

// Navigateは、指定したURLにブラウザを遷移させます。
//
// args:
//
//	url: 遷移先のURL
//
// return:
//
//	error: 失敗時のエラー
func (b *browserClient) Navigate(url string) error {
	if _, err := b.page.Goto(url, playwright.PageGotoOptions{
		Timeout:   playwright.Float(float64(b.cfg.CrawlTimeoutSeconds * 1000)),
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		return fmt.Errorf("ナビゲーションに失敗しました: %v", err)
	}
	return nil
}

// Clickは、指定したセレクタの要素をクリックします。
//
// args:
//
//	selector: クリック対象のCSSセレクタ
//
// return:
//
//	error: 失敗時のエラー
func (b *browserClient) Click(selector string) error {
	locator := b.page.Locator(selector).First()
	if err := locator.WaitFor(); err != nil {
		return fmt.Errorf("セレクター '%s' の可視状態待機に失敗しました: %w", selector, err)
	}
	if err := locator.Click(); err != nil {
		return fmt.Errorf("%sのクリックに失敗しました: %w", selector, err)
	}
	return nil
}

// GetHTMLは、現在のページのHTMLを取得します。
//
// args: なし
// return:
//
//	string: HTML文字列
//	error: 失敗時のエラー
func (b *browserClient) GetHTML() (string, error) {
	if err := b.page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateDomcontentloaded,
	}); err != nil {
		return "", fmt.Errorf("ページ読み込み待機に失敗しました: %w", err)
	}
	html, err := b.page.Content()
	if err != nil {
		return "", fmt.Errorf("ページコンテンツの取得に失敗しました: %w", err)
	}
	return html, nil
}

// SaveHTMLは、HTMLをファイルに保存します。
//
// args:
//
//	filename: 保存ファイル名
//	content: HTML文字列
//
// return:
//
//	error: 失敗時のエラー
func (b *browserClient) SaveHTML(filename string, content string) error {
	filePath := filepath.Join(b.cfg.OutputDir, filename)
	if err := os.MkdirAll(b.cfg.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗しました: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), os.ModePerm); err != nil {
		return fmt.Errorf("HTMLファイルの書き込みに失敗しました: %w", err)
	}

	return nil
}

// CurrentURLは、現在のページのURLを返します。
//
// args: なし
// return:
//
//	*url.URL: 現在のURL
//	error: 失敗時のエラー
func (b *browserClient) CurrentURL() (*url.URL, error) {
	rawURL := b.page.URL()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("現在のURLのパースに失敗しました: %w", err)
	}
	return parsed, nil
}

// Closeは、ブラウザとPlaywrightインスタンスを閉じます。
//
// args: なし
// return:
//
//	error: 失敗時のエラー
func (b *browserClient) Close() error {
	if err := b.context.Close(); err != nil {
		return fmt.Errorf("ブラウザコンテキストのクローズに失敗しました: %w", err)
	}

	if err := b.browser.Close(); err != nil {
		return fmt.Errorf("ブラウザを閉じれませんでした: %w", err)
	}

	if err := b.pw.Stop(); err != nil {
		return fmt.Errorf("playwrightの停止に失敗しました: %w", err)
	}
	return nil
}

// ExtractTextは、指定したセレクタに一致する要素のテキストを抽出します。
//
// args:
//
//	selector: CSSセレクタ
//
// return:
//
//	[]string: テキストのリスト
//	error: 失敗時のエラー
func (b *browserClient) ExtractText(selector string) ([]string, error) {
	locator := b.page.Locator(selector)
	if err := locator.First().WaitFor(); err != nil {
		return nil, fmt.Errorf("テキスト抽出前のセレクター待機に失敗しました: %w", err)
	}
	entries, err := locator.All()
	if err != nil {
		return nil, fmt.Errorf("エントリの取得に失敗しました: %w", err)
	}

	texts := make([]string, 0, len(entries))
	for _, entry := range entries {
		text, err := entry.TextContent()
		if err != nil {
			return nil, fmt.Errorf("テキストコンテンツの取得に失敗しました: %w", err)
		}

		texts = append(texts, text)
	}

	return texts, nil
}

// ExtractAttributeは、指定したセレクタに一致する要素から属性値を抽出します。
//
// args:
//
//	selector: CSSセレクタ
//	attr: 属性名
//
// return:
//
//	[]string: 属性値のリスト
//	error: 失敗時のエラー
func (b *browserClient) ExtractAttribute(selector string, attr string) ([]string, error) {
	locator := b.page.Locator(selector)
	if err := locator.First().WaitFor(); err != nil {
		return nil, fmt.Errorf("属性抽出前のセレクター待機に失敗しました: %w", err)
	}
	entries, err := locator.All()
	if err != nil {
		return nil, fmt.Errorf("エントリの取得に失敗しました: %w", err)
	}

	values := make([]string, 0, len(entries))
	for _, entry := range entries {
		value, err := entry.GetAttribute(attr)
		if err != nil {
			return nil, fmt.Errorf("属性値の取得に失敗しました: %w", err)
		}
		if value != "" {
			values = append(values, value)
		}
	}

	return values, nil
}

// Existsは、指定したセレクタに一致する要素が存在するか判定します。
//
// args:
//
//	selector: CSSセレクタ
//
// return:
//
//	bool: 存在する場合はtrue
//	error: 失敗時のエラー
func (b *browserClient) Exists(selector string) (bool, error) {
	count, err := b.page.Locator(selector).Count()
	if err != nil {
		return false, fmt.Errorf("セレクター %s の要素数カウントに失敗しました: %w", selector, err)
	}
	return count > 0, nil
}
