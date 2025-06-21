package infra

import (
	"fmt"
	"os"
	"path/filepath"
)

// HTMLFileLoaderは、ローカルファイルシステムからHTMLファイルの読み込みに関連する操作を提供します。
type HTMLFileLoader struct{}

// NewHTMLFileLoaderは、HTMLFileLoaderの新しいインスタンスを生成します。
func NewHTMLFileLoader() *HTMLFileLoader {
	return &HTMLFileLoader{}
}

// LoadHTMLFileは、指定されたパスからHTMLファイルを読み込み、その内容を文字列として返します。
//
// args:
//
//	path : 読み込むHTMLファイルのパス
//
// return:
//
//	string : ファイルの内容
//	error  : ファイルの読み込み中にエラーが発生した場合
func (f *HTMLFileLoader) LoadHTMLFile(path string) (string, error) {
	html, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read HTML file: %w", err)
	}
	return string(html), nil
}

// ListHTMLFilePathsは、指定されたディレクトリ配下のすべての.htmlファイルのパスを再帰的に検索して返します。
//
// args:
//
//	dir : 検索を開始するディレクトリのパス
//
// return:
//
//	[]string : 見つかったHTMLファイルのパスのスライス
//	error    : ディレクトリの走査中にエラーが発生した場合
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
