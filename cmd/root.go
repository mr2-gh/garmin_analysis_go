package cmd

import (
	"fmt"
	"os"

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
