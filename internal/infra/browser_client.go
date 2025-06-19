package infra

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/nrad-K/go-crawler/internal/config"
	"github.com/playwright-community/playwright-go"
)

type BrowserClient interface {
	Click(ctx context.Context, selector string) error
	GetHTML(ctx context.Context) (string, error)
	SaveHTML(ctx context.Context, filename string, content string) error
	GetCurrentURL(ctx context.Context) (*url.URL, error)
	Navigate(ctx context.Context, url string) error
	ExtractText(selector string) ([]string, error)
	ExtractAttribute(selector, attr string) ([]string, error)
	Exists(selector string) (bool, error)
	NewPage(ctx context.Context, url string) (BrowserClient, error)
	Close() error
}

type browserClient struct {
	pw      *playwright.Playwright
	cfg     *config.CrawlerConfig
	browser playwright.Browser
	page    playwright.Page
	context playwright.BrowserContext
}

func NewBrowserClient(cfg *config.CrawlerConfig) (BrowserClient, error) {
	if err := playwright.Install(); err != nil {
		return nil, fmt.Errorf("playwrightのインストールに失敗しました: %w", err)
	}
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
		browser.Close()
		pw.Stop()
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
	return context.Route("**/*.{png,jpg,jpeg,gif,svg,woff,woff2,ttf,eot,otf,css}", func(route playwright.Route) {
		route.Abort()
	})
}

func (b *browserClient) NewPage(ctx context.Context, url string) (BrowserClient, error) {
	context, err := b.browser.NewContext(playwright.BrowserNewContextOptions{
		ExtraHttpHeaders: b.cfg.Headers,
		UserAgent:        &b.cfg.UserAgent,
	})
	if err != nil {
		context.Close()
		return nil, fmt.Errorf("ブラウザコンテキストの作成に失敗しました: %w", err)
	}
	page, err := context.NewPage()
	if err != nil {
		context.Close()
		return nil, fmt.Errorf("新しいページの作成に失敗しました: %w", err)
	}

	if _, err := page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); err != nil {
		page.Close()
		context.Close()
		return nil, fmt.Errorf("ナビゲーションに失敗しました: %v", err)
	}

	return &browserClient{
		pw:      b.pw,
		browser: b.browser,
		cfg:     b.cfg,
		context: context,
		page:    page,
	}, nil
}
func (b *browserClient) Navigate(ctx context.Context, url string) error {
	if _, err := b.page.Goto(url); err != nil {
		return fmt.Errorf("ナビゲーションに失敗しました: %v", err)
	}
	return nil
}

func (b *browserClient) Click(ctx context.Context, selector string) error {
	locator := b.page.Locator(selector).First()
	if err := locator.WaitFor(); err != nil {
		return fmt.Errorf("セレクター '%s' の可視状態待機に失敗しました: %w", selector, err)
	}
	if err := locator.Click(); err != nil {
		return fmt.Errorf("%sのクリックに失敗しました: %w", selector, err)
	}
	return nil
}

func (b *browserClient) GetHTML(ctx context.Context) (string, error) {
	html, err := b.page.Content()
	if err != nil {
		return "", fmt.Errorf("ページコンテンツの取得に失敗しました: %w", err)
	}
	return html, nil
}

func (b *browserClient) SaveHTML(ctx context.Context, filename string, content string) error {
	filePath := filepath.Join(b.cfg.OutputDir, filename)
	if err := os.MkdirAll(b.cfg.OutputDir, os.ModePerm); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗しました: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), os.ModePerm); err != nil {
		return fmt.Errorf("HTMLファイルの書き込みに失敗しました: %w", err)
	}

	return nil
}

func (b *browserClient) GetCurrentURL(ctx context.Context) (*url.URL, error) {
	rawURL := b.page.URL()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("現在のURLのパースに失敗しました: %w", err)
	}
	return parsed, nil
}

func (b *browserClient) Close() error {
	if err := b.browser.Close(); err != nil {
		return fmt.Errorf("ブラウザを閉じれませんでした: %w", err)
	}
	if err := b.pw.Stop(); err != nil {
		return fmt.Errorf("playwrightの停止に失敗しました: %w", err)
	}
	return nil

}
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

func (b *browserClient) Exists(selector string) (bool, error) {
	count, err := b.page.Locator(selector).Count()
	if err != nil {
		return false, fmt.Errorf("セレクター %s の要素数カウントに失敗しました: %w", selector, err)
	}
	return count > 0, nil
}
