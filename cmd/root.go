package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmdは、アプリケーションのエントリーポイントとなるルートコマンドです。
var rootCmd = &cobra.Command{
	Use:   "go-crawler",
	Short: "求人情報サイトのクローリングとスクレイピングを行うツールです。",
	Long: `go-crawlerは、求人情報のURLを収集するクローラー機能と、
ダウンロード済みのHTMLファイルから詳細情報を抽出するスクレイパー機能を提供します。`,
}

// Executeは、全てのサブコマンドをルートコマンドに追加し、フラグを適切に設定します。
// この関数はmain.main()から呼び出され、rootCmdに対して一度だけ実行される必要があります。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
