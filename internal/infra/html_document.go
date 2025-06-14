package infra

import (
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type HTMLDocument interface {
	ExtractText(html string, selector string) ([]string, error)
	ExtractAttribute(html string, selector, attr string) ([]string, error)
	ExtractTextByRegex(html, selector, pattern string) ([]string, error)
}

type htmlDocument struct {
	doc *goquery.Document
}

func NewHTMLDocument(html string) (HTMLDocument, error) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}
	return &htmlDocument{doc: document}, nil
}

// ExtractText はHTMLから特定のセレクタにマッチする要素のテキストを抽出します。
//
// 使用例:
//
//   - 段落テキストの抽出: ExtractText(html, "p")
//     入力: <p>これは段落です</p>
//     出力: ["これは段落です"]
//
//   - リスト項目の抽出: ExtractText(html, "li")
//     入力: <ul><li>項目1</li><li>項目2</li></ul>
//     出力: ["項目1", "項目2"]
//
//   - クラス指定での抽出: ExtractText(html, ".title")
//     入力: <h1 class="title">メインタイトル</h1>
//     出力: ["メインタイトル"]
//
// パラメータ:
//   - html: 解析対象のHTML文字列
//   - selector: 要素を選択するためのCSSセレクタ
//
// 戻り値:
//   - []string: 抽出されたテキストの配列
//   - error: エラーが発生した場合のエラー情報
func (h *htmlDocument) ExtractText(html string, selector string) ([]string, error) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var texts []string
	document.Find(selector).Each(func(_ int, s *goquery.Selection) {
		texts = append(texts, s.Text())
	})

	return texts, nil
}

// ExtractAttribute はHTMLから特定のセレクタにマッチする要素の属性値を抽出します。
//
// 使用例:
//
//   - リンクのhref属性抽出: ExtractAttribute(html, "a", "href")
//     入力: <a href="https://example.com">リンク</a>
//     出力: ["https://example.com"]
//
//   - 画像のsrc属性抽出: ExtractAttribute(html, "img", "src")
//     入力: <img src="image.jpg" alt="画像">
//     出力: ["image.jpg"]
//
//   - カスタムデータ属性の抽出: ExtractAttribute(html, "div", "data-id")
//     入力: <div data-id="12345">コンテンツ</div>
//     出力: ["12345"]
//
// パラメータ:
//   - html: 解析対象のHTML文字列
//   - selector: 要素を選択するためのCSSセレクタ
//   - attr: 抽出する属性名
//
// 戻り値:
//   - []string: 抽出された属性値の配列
//   - error: エラーが発生した場合のエラー情報
func (h *htmlDocument) ExtractAttribute(html string, selector, attr string) ([]string, error) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var attributes []string
	document.Find(selector).Each(func(_ int, s *goquery.Selection) {
		if value, exists := s.Attr(attr); exists {
			attributes = append(attributes, value)
		}
	})

	return attributes, nil
}

// ExtractTextByRegex はHTMLから特定のセレクタにマッチする要素を抽出し、
// その要素のテキストに対して正規表現パターンを適用してマッチした文字列を返します。
//
// 使用例:
//
//   - 価格の抽出: ExtractTextByRegex(html, ".price", `¥[\d,]+`)
//     入力: <div class="price">¥1,980</div>
//     出力: ["¥1,980"]
//
//   - 電話番号の抽出: ExtractTextByRegex(html, ".contact", `\d{2,4}-\d{2,4}-\d{4}`)
//     入力: <span class="contact">TEL: 03-1234-5678</span>
//     出力: ["03-1234-5678"]
//
//   - カッコで囲まれた文字列: ExtractTextByRegex(html, "div", `\((.*?)\)`)
//     入力: <div>これは(重要)な情報です</div>
//     出力: ["(重要)"]
//
//   - 日付の抽出: ExtractTextByRegex(html, "time", `\d{4}/\d{2}/\d{2}`)
//     入力: <time>2024/03/15</time>
//     出力: ["2024/03/15"]
//
//   - メールアドレスの抽出: ExtractTextByRegex(html, "a", `[^@]+@[^@]+\.[^@]+`)
//     入力: <a>contact@example.com</a>
//     出力: ["contact@example.com"]
//
// パラメータ:
//   - html: 解析対象のHTML文字列
//   - selector: 要素を選択するためのCSSセレクタ
//   - pattern: テキストから抽出するための正規表現パターン
//
// 戻り値:
//   - []string: マッチした文字列の配列
//   - error: エラーが発生した場合のエラー情報
func (h *htmlDocument) ExtractTextByRegex(html, selector, pattern string) ([]string, error) {
	document, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	var matches []string
	document.Find(selector).Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		found := re.FindAllString(text, -1)
		if found != nil {
			matches = append(matches, found...)
		}
	})

	return matches, nil
}
