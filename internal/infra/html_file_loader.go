package infra

import (
	"fmt"
	"os"
	"path/filepath"
)

type HTMLFileLoader interface {
	LoadHTMLFile(path string) (string, error)
	ListHTMLFilePaths(dir string) ([]string, error)
}

type htmlFileLoader struct {
}

func NewHTMLFileLoader() *htmlFileLoader {
	return &htmlFileLoader{}
}

func (f *htmlFileLoader) LoadHTMLFile(path string) (string, error) {
	html, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML file: %w", err)
	}
	return string(html), nil
}

func (f *htmlFileLoader) ListHTMLFilePaths(dir string) ([]string, error) {
	paths, err := filepath.Glob(filepath.Join("html", "*.html"))
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}
	return paths, nil
}
