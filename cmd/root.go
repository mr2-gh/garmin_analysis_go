package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "garmin-cli",
	Short: "Garmin Connect サイクリングデータ分析CLIツール",
	Long:  `Garmin Connectからサイクリングアクティビティデータを取得し、CTL/ATL/TSBなどのPMCトレーニング負荷を分析・表示するCLIツールです。`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "エラーが発生しました: %v\n", err)
		os.Exit(1)
	}
}

// getCSVPath は、実行ファイルの実体と同じディレクトリにある指定されたファイルの絶対パスを返します。
func getCSVPath(filename string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("実行ファイルのパス取得に失敗しました: %w", err)
	}
	realPath, err := filepath.EvalSymlinks(exePath)
	if err != nil {
		realPath = exePath
	}
	return filepath.Join(filepath.Dir(realPath), filename), nil
}
