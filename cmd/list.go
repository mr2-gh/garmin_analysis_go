package cmd

import (
	"fmt"
	"sort"
	"strings"

	"garmin-cli/pkg/data"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	limitFlag int
	typeFlag  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "保存されているサイクリングアクティビティの一覧を表示します",
	RunE: func(cmd *cobra.Command, args []string) error {
		csvPath, err := getCSVPath("garmin_cycling_activities.csv")
		if err != nil {
			return err
		}
		acts, err := data.LoadCSV(csvPath)
		if err != nil {
			return fmt.Errorf("CSVファイルのロードに失敗しました: %w (先に `fetch` コマンドを実行してください)", err)
		}

		if len(acts) == 0 {
			fmt.Println("アクティビティデータがありません。")
			return nil
		}

		// 開始時間の降順（最新順）でソート
		sort.Slice(acts, func(i, j int) bool {
			return acts[i].StartTimeLocal > acts[j].StartTimeLocal
		})

		// テーブル表示の初期化
		table := tablewriter.NewWriter(cmd.OutOrStdout())
		table.SetHeader([]string{"開始時間", "アクティビティ名", "タイプ", "距離(km)", "時間(分)", "平均パワー", "TSS", "推定FTP"})
		table.SetBorder(false)
		table.SetAutoWrapText(false)

		count := 0
		for _, act := range acts {
			// 表示件数制限の適用
			if limitFlag > 0 && count >= limitFlag {
				break
			}

			// アクティビティタイプによるフィルタリング（指定がある場合）
			if typeFlag != "" {
				actType := strings.ToLower(act.Type)
				filterType := strings.ToLower(typeFlag)
				if !strings.Contains(actType, filterType) {
					continue
				}
			}

			// NPとIFから当時の推定FTPを逆算
			ftpVal := 0.0
			if act.NormPower > 0 && act.IntensityFactor > 0 {
				ftpVal = act.NormPower / act.IntensityFactor
			}

			ftpStr := "-"
			if ftpVal > 0 {
				ftpStr = fmt.Sprintf("%.0f W", ftpVal)
			}

			table.Append([]string{
				act.StartTimeLocal,
				act.Name,
				act.Type,
				fmt.Sprintf("%.2f", act.DistanceMeters/1000.0),
				fmt.Sprintf("%.1f", act.DurationSeconds/60.0),
				fmt.Sprintf("%.0f W", act.AvgPower),
				fmt.Sprintf("%.1f", act.TSS),
				ftpStr,
			})
			count++
		}

		table.Render()
		fmt.Printf("\n表示件数: %d 件 (総件数: %d 件)\n", count, len(acts))
		return nil
	},
}

func init() {
	listCmd.Flags().IntVarP(&limitFlag, "limit", "n", 20, "表示するアクティビティの最大件数")
	listCmd.Flags().StringVarP(&typeFlag, "type", "t", "", "アクティビティタイプでフィルター (road_biking, indoor_cycling 等)")
	rootCmd.AddCommand(listCmd)
}
