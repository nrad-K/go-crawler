# バイナリ名
BINARY=go-crawler

# ビルド
build:
	go build -o $(BINARY) .

# 依存関係の取得
deps:
	go mod tidy

# フォーマット
fmt:
	go fmt ./...

# テスト
test:
	go test ./...

# クリーニング
clean:
	rm -f $(BINARY)

all:
	./go-crawler crawler -g && \
	./go-crawler crawler -e && \
	./go-crawler scrape


.PHONY: build deps fmt test clean crawler-generate crawler-execute scrape