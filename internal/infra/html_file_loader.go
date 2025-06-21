package infra

import (
	"fmt"
	"os"
	"path/filepath"
)

type HTMLFileLoader struct{}

func NewHTMLFileLoader() *HTMLFileLoader {
	return &HTMLFileLoader{}
}

func (f *HTMLFileLoader) LoadHTMLFile(path string) (string, error) {
	html, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML file: %w", err)
	}
	return string(html), nil
}

func (f *HTMLFileLoader) ListHTMLFilePaths(dir string) ([]string, error) {
	// 指定ディレクトリ配下の全ての.htmlファイルを再帰的に取得する
	paths := make([]string, 0, 10000)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".html" {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return paths, fmt.Errorf("ディレクトリの走査に失敗しました: %w", err)
	}

	return paths, nil
}
