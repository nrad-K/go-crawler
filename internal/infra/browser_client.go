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
	Exists(selector string) bool
	Close() error
}

type browserClient struct {
	pw      *playwright.Playwright
	cfg     *config.CrawlerConfig
	browser playwright.Browser
	page    playwright.Page
}

func NewBrowserClient(cfg *config.CrawlerConfig) (BrowserClient, error) {
	if err := playwright.Install(); err != nil {
		return nil, fmt.Errorf("failed to install Playwright: %w", err)
	}
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to start Playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	page, err := browser.NewPage()
	if err != nil {
		return nil, fmt.Errorf("failed to create page: %w", err)
	}

	return &browserClient{
		pw:      pw,
		browser: browser,
		page:    page,
		cfg:     cfg,
	}, nil
}

func (b *browserClient) Navigate(ctx context.Context, url string) error {
	if _, err := b.page.Goto(url); err != nil {
		return fmt.Errorf("failed to navigate: %v", err)
	}
	return nil
}

func (b *browserClient) Click(ctx context.Context, selector string) error {
	if err := b.page.Locator(selector).Click(); err != nil {
		return fmt.Errorf("failed to click %s: %w", selector, err)
	}
	return nil
}

func (b *browserClient) GetHTML(ctx context.Context) (string, error) {
	html, err := b.page.Content()
	if err != nil {
		return "", fmt.Errorf("failed to get page content: %w", err)
	}
	return html, nil
}

func (b *browserClient) SaveHTML(ctx context.Context, filename string, content string) error {
	filePath := filepath.Join(b.cfg.OutputDirectory, filename)
	if err := os.MkdirAll("saved_pages", os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), os.ModePerm); err != nil {
		return fmt.Errorf("failed to write HTML to file: %w", err)
	}

	return nil
}

func (b *browserClient) GetCurrentURL(ctx context.Context) (*url.URL, error) {
	rawURL := b.page.URL()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse current URL: %w", err)
	}
	return parsed, nil
}

func (b *browserClient) Close() error {
	if err := b.browser.Close(); err != nil {
		return err
	}
	if err := b.pw.Stop(); err != nil {
		return err
	}
	return nil
}

func (b *browserClient) ExtractText(selector string) ([]string, error) {
	entries, err := b.page.Locator(selector).All()
	if err != nil {
		return nil, fmt.Errorf("cloud not get entries: %w", err)
	}

	var texts []string
	for _, entry := range entries {
		text, err := entry.TextContent()
		if err != nil {
			return nil, fmt.Errorf("could not get text context: %w", err)
		}
		texts = append(texts, text)
	}

	return texts, nil
}

func (b *browserClient) ExtractAttribute(selector string, attr string) ([]string, error) {
	entries, err := b.page.Locator(selector).All()
	if err != nil {
		return nil, fmt.Errorf("cloud not get entries: %w", err)
	}

	var values []string
	for _, entry := range entries {
		value, err := entry.GetAttribute(attr)
		if err != nil {
			return nil, fmt.Errorf("could not get text context: %w", err)
		}
		values = append(values, value)
	}

	return values, nil
}

func (b *browserClient) Exists(selector string) bool {
	count, err := b.page.Locator(selector).Count()
	if err != nil {
		return false
	}
	return count > 0
}
