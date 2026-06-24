package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"garmin-cli/internal/garmin/activities"
	"garmin-cli/internal/garmin/auth"
	"garmin-cli/internal/garmin/client"
	"garmin-cli/internal/garmin/config"
	"garmin-cli/pkg/data"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Garmin Connectからサイクリングアクティビティをダウンロードします",
	RunE: func(cmd *cobra.Command, args []string) error {
		// カレントディレクトリまたは親ディレクトリから .env を読み込む
		_ = godotenv.Load(".env", "../.env")

		username := os.Getenv("GARMIN_USERNAME")
		password := os.Getenv("GARMIN_PASSWORD")
		maxActsStr := os.Getenv("MAX_ACTIVITY_NUM")

		if username == "" || password == "" {
			return fmt.Errorf("GARMIN_USERNAME または GARMIN_PASSWORD が環境変数（.env）に設定されていません")
		}

		maxActs := 50
		if maxActsStr != "" {
			if v, err := strconv.Atoi(maxActsStr); err == nil {
				maxActs = v
			}
		}

		ctx := context.Background()

		configDir, err := config.ResolveConfigDir("")
		if err != nil {
			return fmt.Errorf("設定ディレクトリの解決に失敗しました: %w", err)
		}

		profile := "default"
		fmt.Println("[1/4] Garmin Connect にログインしています...")

		promptMFA := func() (string, error) {
			fmt.Print("Garminからメール等で届いたMFA（ワンタイムコード）を入力してください: ")
			var code string
			_, err := fmt.Scanln(&code)
			return code, err
		}

		var cl *client.Client
		sess, err := auth.LoadSession(configDir, profile)
		if err == nil {
			// キャッシュされたセッションが存在する場合はロード
			cl = client.NewWithSession(configDir, profile, sess, client.Options{})
		} else {
			// キャッシュがない、または無効な場合は新規ログインを試行
			fmt.Println("      -> 新しいセッションを開始します...")
			sess, err = auth.Login(ctx, configDir, username, password, promptMFA)
			if err != nil {
				return fmt.Errorf("ログインに失敗しました: %w", err)
			}
			err = auth.SaveSession(configDir, profile, sess)
			if err != nil {
				fmt.Printf("      -> 警告: セッションの保存に失敗しました: %v\n", err)
			}
			cl = client.NewWithSession(configDir, profile, sess, client.Options{})
		}

		// ユーザーの体重データを取得（設定されていれば）
		weight, err := activities.GetUserWeight(ctx, cl)
		if err == nil {
			fmt.Printf("      -> ユーザー体重を取得しました: %.1f kg\n", weight)
		} else {
			fmt.Printf("      -> ユーザー体重の取得をスキップしました: %v\n", err)
		}

		// アクティビティ一覧の取得
		fmt.Println("[2/4] アクティビティ一覧を取得しています...")
		// サーバーに負荷をかけないようにまとめて取得し、後でロードバイクとインドアサイクリングをフィルタリングします。
		// フィルタリング後に十分な件数が残るよう、MAX_ACTIVITY_NUM の5倍（最低100件）を取得します。
		limitToFetch := maxActs * 5
		if limitToFetch < 100 {
			limitToFetch = 100
		}

		rawActs, err := activities.List(ctx, cl, limitToFetch, "", "", "")
		if err != nil {
			return fmt.Errorf("アクティビティ一覧の取得に失敗しました: %w", err)
		}

		fmt.Printf("      -> 合計 %d 件のアクティビティを読み込みました。\n", len(rawActs))

		fmt.Println("[3/4] 対象データ（ロードバイク・インドアサイクリング）の抽出を開始します...")
		var filteredActs []activities.Summary
		for _, act := range rawActs {
			t := strings.ToLower(act.Type)
			if t == "road_biking" || t == "indoor_cycling" {
				filteredActs = append(filteredActs, act)
				if len(filteredActs) >= maxActs {
					break
				}
			}
		}

		fmt.Printf("      -> %d 件 of サイクリングアクティビティを抽出しました。\n", len(filteredActs))

		// CSVファイルへの書き出し
		outputFilename := "garmin_cycling_activities.csv"
		err = data.WriteCSV(outputFilename, filteredActs)
		if err != nil {
			return fmt.Errorf("CSVへの書き出しに失敗しました: %w", err)
		}

		fmt.Printf("[4/4] すべての処理が完了しました。データは %s に正常に保存されました。\n", outputFilename)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(fetchCmd)
}
