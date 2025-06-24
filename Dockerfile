# syntax=docker/dockerfile:1
FROM golang:1.24

WORKDIR /app

# 依存関係をコピーしてダウンロード
COPY go.mod go.sum ./
RUN go mod download

# ソースコードをコピーしてビルド
COPY . .
RUN make build

# Playwrightドライバーをインストール
RUN go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5200.0 install --with-deps

EXPOSE 8080

CMD ["make", "all"]