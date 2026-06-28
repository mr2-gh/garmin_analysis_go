package cmd

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"sort"
	"time"

	"garmin-cli/internal/garmin/activities"
	"garmin-cli/pkg/data"
	"garmin-cli/pkg/pmc"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

//go:embed index.html
var htmlContent []byte

var (
	webFlag  bool
	portFlag int
)

type KPIsJSON struct {
	ThisWeekTSS      float64 `json:"thisWeekTSS"`
	ThisWeekDuration float64 `json:"thisWeekDuration"`
	ThisWeekDistance float64 `json:"thisWeekDistance"`
	LatestFTP        float64 `json:"latestFTP"`
}

type ActivityJSON struct {
	ID                   int64   `json:"id"`
	Name                 string  `json:"name"`
	Date                 string  `json:"date"`
	StartTimeLocal       string  `json:"startTimeLocal"`
	Type                 string  `json:"type"`
	DistanceKm           float64 `json:"distanceKm"`
	DurationMin          float64 `json:"durationMin"`
	ElevationM           float64 `json:"elevationM"`
	AvgSpeed             float64 `json:"avgSpeed"`
	MaxSpeed             float64 `json:"maxSpeed"`
	AvgCadence           float64 `json:"avgCadence"`
	MaxCadence           float64 `json:"maxCadence"`
	AvgPower             float64 `json:"avgPower"`
	MaxPower             float64 `json:"maxPower"`
	Max20MinPower        float64 `json:"max20MinPower"`
	NormalizedPower      float64 `json:"normalizedPower"`
	IntensityFactor      float64 `json:"intensityFactor"`
	AvgHR                float64 `json:"avgHR"`
	MaxHR                float64 `json:"maxHR"`
	Calories             float64 `json:"calories"`
	VO2Max               float64 `json:"vo2Max"`
	AerobicTE            float64 `json:"aerobicTE"`
	AnaerobicTE          float64 `json:"anaerobicTE"`
	TSS                  float64 `json:"tss"`
	FTP                  float64 `json:"ftp"`
	Zone1                float64 `json:"zone1"`
	Zone2                float64 `json:"zone2"`
	Zone3                float64 `json:"zone3"`
	Zone4                float64 `json:"zone4"`
	Zone5                float64 `json:"zone5"`
	Zone6                float64 `json:"zone6"`
	Zone7                float64 `json:"zone7"`
}

type PMCJSON struct {
	Date string  `json:"date"`
	TSS  float64 `json:"tss"`
	CTL  float64 `json:"ctl"`
	ATL  float64 `json:"atl"`
	TSB  float64 `json:"tsb"`
}

type WeeklyJSON struct {
	WeekStart string    `json:"weekStart"`
	TSS       float64   `json:"tss"`
	Duration  float64   `json:"duration"`
	Distance  float64   `json:"distance"`
	Elevation float64   `json:"elevation"`
	CTL       float64   `json:"ctl"`
	ATL       float64   `json:"atl"`
	TSB       float64   `json:"tsb"`
	FTP       float64   `json:"ftp"`
	Zones     []float64 `json:"zones"`
}

type APIDataResponse struct {
	KPIs       KPIsJSON       `json:"kpis"`
	Activities []ActivityJSON `json:"activities"`
	PMC        []PMCJSON      `json:"pmc"`
	Weekly     []WeeklyJSON   `json:"weekly"`
}

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "トレーニングステータスとPMCモデルの統計情報を表示します（--webでブラウザ表示可能）",
	RunE: func(cmd *cobra.Command, args []string) error {
		csvFilename := "garmin_cycling_activities.csv"
		acts, err := data.LoadCSV(csvFilename)
		if err != nil {
			return fmt.Errorf("CSVファイルのロードに失敗しました: %w (先に `fetch` コマンドを実行してください)", err)
		}

		if len(acts) == 0 {
			fmt.Println("アクティビティデータがありません。")
			return nil
		}

		// アクティビティを開始日時の昇順（古い順）にソート
		sort.Slice(acts, func(i, j int) bool {
			return acts[i].StartTimeLocal < acts[j].StartTimeLocal
		})

		// 解析対象の日付範囲を特定
		var minDate, maxDate time.Time
		for _, act := range acts {
			t, err := time.ParseInLocation("2006-01-02T15:04:05", act.StartTimeLocal, time.Local)
			if err != nil {
				t, err = time.ParseInLocation("2006-01-02", act.Date, time.Local)
			}
			if err == nil {
				if minDate.IsZero() || t.Before(minDate) {
					minDate = t
				}
				if t.After(maxDate) {
					maxDate = t
				}
			}
		}

		// 時間情報を切り捨てて日付に正規化
		minDate = time.Date(minDate.Year(), minDate.Month(), minDate.Day(), 0, 0, 0, 0, time.Local)
		today := time.Now()
		today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.Local)
		if today.After(maxDate) {
			maxDate = today
		} else {
			maxDate = time.Date(maxDate.Year(), maxDate.Month(), maxDate.Day(), 0, 0, 0, 0, time.Local)
		}

		// 日ごとのTSS合計を算出するためのマップ
		dailyTSSMap := make(map[string]float64)
		for _, act := range acts {
			dailyTSSMap[act.Date] += act.TSS
		}

		// 日次のデータスライスを生成
		var dailyMetrics []pmc.DailyMetrics
		for d := minDate; !d.After(maxDate); d = d.AddDate(0, 0, 1) {
			dateStr := d.Format("2006-01-02")
			dailyMetrics = append(dailyMetrics, pmc.DailyMetrics{
				Date: d,
				TSS:  dailyTSSMap[dateStr],
			})
		}

		// PMC（CTL / ATL / TSB）モデルの計算
		pmcCalculated := pmc.CalculatePMC(dailyMetrics)

		// KPI（重要指標）の計算
		// 直近のFTP（設定値または最終算出値）を取得
		latestFTP := 0.0
		for _, act := range acts {
			ftpVal := 0.0
			if act.NormPower > 0 && act.IntensityFactor > 0 {
				ftpVal = act.NormPower / act.IntensityFactor
			}
			if ftpVal > 0 {
				latestFTP = ftpVal
			}
		}

		// 今週の開始日（月曜日）を特定
		now := time.Now()
		offset := int(now.Weekday() - time.Monday)
		if offset < 0 {
			offset += 7
		}
		currentWeekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, -offset)

		// 今週（月曜〜本日）の総TSS、時間、距離を計算
		thisWeekTSS := 0.0
		thisWeekDuration := 0.0
		thisWeekDistance := 0.0
		for _, act := range acts {
			t, err := time.ParseInLocation("2006-01-02", act.Date, time.Local)
			if err == nil {
				if !t.Before(currentWeekStart) && !t.After(today) {
					thisWeekTSS += act.TSS
					thisWeekDuration += act.DurationSeconds / 60.0
					thisWeekDistance += act.DistanceMeters / 1000.0
				}
			}
		}

		// --web フラグが有効な場合はWebサーバーを起動
		if webFlag {
			return startWebServer(portFlag, acts, pmcCalculated, latestFTP, thisWeekTSS, thisWeekDuration, thisWeekDistance)
		}

		// CLIコンソール上でのサマリー表示
		latestPMC := pmcCalculated[len(pmcCalculated)-1]

		fmt.Println("==================================================")
		fmt.Println("🚴‍♂️ ロードバイクトレーニング ステータスサマリー")
		fmt.Println("==================================================")
		fmt.Printf("最新の推定FTP  : %.0f W\n", latestFTP)
		fmt.Printf("現在の体力(CTL): %.1f\n", latestPMC.CTL)
		fmt.Printf("現在の疲労(ATL): %.1f\n", latestPMC.ATL)
		fmt.Printf("現在の調子(TSB): %.1f\n", latestPMC.TSB)
		fmt.Println("--------------------------------------------------")
		fmt.Println("今週の進捗（月曜日始まり）:")
		fmt.Printf("  総TSS        : %.1f\n", thisWeekTSS)
		fmt.Printf("  総走行時間   : %.1f 時間 (%.0f 分)\n", thisWeekDuration/60.0, thisWeekDuration)
		fmt.Printf("  総走行距離   : %.1f km\n", thisWeekDistance)
		fmt.Println("==================================================")

		// 週次サマリーテーブルと簡易グラフ用にデータを週単位にグループ化
		weeklyMap := make(map[string]*WeeklyJSON)
		var weekKeys []string

		for _, dm := range pmcCalculated {
			// Find Monday for this date
			dOffset := int(dm.Date.Weekday() - time.Monday)
			if dOffset < 0 {
				dOffset += 7
			}
			monday := dm.Date.AddDate(0, 0, -dOffset)
			weekKey := monday.Format("2006-01-02") + "～"

			wm, exists := weeklyMap[weekKey]
			if !exists {
				wm = &WeeklyJSON{WeekStart: weekKey}
				weeklyMap[weekKey] = wm
				weekKeys = append(weekKeys, weekKey)
			}
			wm.TSS += dm.TSS
			wm.CTL = dm.CTL
			wm.ATL = dm.ATL
			wm.TSB = dm.TSB
		}

		// 各アクティビティから週次の合計距離・時間を加算
		for _, act := range acts {
			t, err := time.ParseInLocation("2006-01-02", act.Date, time.Local)
			if err == nil {
				dOffset := int(t.Weekday() - time.Monday)
				if dOffset < 0 {
					dOffset += 7
				}
				monday := t.AddDate(0, 0, -dOffset)
				weekKey := monday.Format("2006-01-02") + "～"
				if wm, exists := weeklyMap[weekKey]; exists {
					wm.Duration += act.DurationSeconds / 60.0
					wm.Distance += act.DistanceMeters / 1000.0
				}
			}
		}

		sort.Strings(weekKeys)

		// 表示用には直近の10週間のみを保持
		displayWeeks := weekKeys
		if len(displayWeeks) > 10 {
			displayWeeks = displayWeeks[len(displayWeeks)-10:]
		}

		fmt.Println("\n📊 直近数週間の推移 (CTL / ATL / TSB は週末時点)")
		table := tablewriter.NewWriter(cmd.OutOrStdout())
		table.SetHeader([]string{"週開始日", "総TSS", "時間(時間)", "距離(km)", "CTL(体力)", "ATL(疲労)", "TSB(調子)"})
		table.SetBorder(false)
		table.SetAutoWrapText(false)

		for _, k := range displayWeeks {
			wm := weeklyMap[k]
			table.Append([]string{
				wm.WeekStart,
				fmt.Sprintf("%.1f", wm.TSS),
				fmt.Sprintf("%.1f", wm.Duration/60.0),
				fmt.Sprintf("%.1f", wm.Distance),
				fmt.Sprintf("%.1f", wm.CTL),
				fmt.Sprintf("%.1f", wm.ATL),
				fmt.Sprintf("%.1f", wm.TSB),
			})
		}
		table.Render()

		fmt.Println("\n📈 週次総TSSの推移グラフ (1文字 ■ = 20 TSS)")
		for _, k := range displayWeeks {
			wm := weeklyMap[k]
			barWidth := int(wm.TSS / 20.0)
			if barWidth > 40 {
				barWidth = 40
			}
			bar := ""
			for i := 0; i < barWidth; i++ {
				bar += "■"
			}
			for i := barWidth; i < 40; i++ {
				bar += "░"
			}
			fmt.Printf("%s | %s | TSS:%5.1f (CTL:%5.1f)\n", wm.WeekStart, bar, wm.TSS, wm.CTL)
		}

		return nil
	},
}

func startWebServer(port int, acts []activities.Summary, pmcCalculated []pmc.DailyMetrics, latestFTP float64, thisWeekTSS, thisWeekDuration, thisWeekDistance float64) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(htmlContent)
	})

	http.HandleFunc("/api/data", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// APIレスポンス用構造体の構築
		kpis := KPIsJSON{
			ThisWeekTSS:      thisWeekTSS,
			ThisWeekDuration: thisWeekDuration,
			ThisWeekDistance: thisWeekDistance,
			LatestFTP:        latestFTP,
		}

		// アクティビティ一覧は最新順（降順）にする
		var activitiesJSON []ActivityJSON
		for i := len(acts) - 1; i >= 0; i-- {
			act := acts[i]
			ftpVal := 0.0
			if act.NormPower > 0 && act.IntensityFactor > 0 {
				ftpVal = act.NormPower / act.IntensityFactor
			}
			activitiesJSON = append(activitiesJSON, ActivityJSON{
				ID:                   act.ID,
				Name:                 act.Name,
				Date:                 act.Date,
				StartTimeLocal:       act.StartTimeLocal,
				Type:                 act.Type,
				DistanceKm:           act.DistanceMeters / 1000.0,
				DurationMin:          act.DurationSeconds / 60.0,
				ElevationM:           act.ElevationGain,
				AvgSpeed:             act.AverageSpeed,
				MaxSpeed:             act.MaxSpeed,
				AvgCadence:           act.AverageBikingCadence,
				MaxCadence:           act.MaxBikingCadence,
				AvgPower:             act.AvgPower,
				MaxPower:             act.MaxPower,
				Max20MinPower:        act.Max20MinPower,
				NormalizedPower:      act.NormPower,
				IntensityFactor:      act.IntensityFactor,
				AvgHR:                act.AverageHR,
				MaxHR:                act.MaxHR,
				Calories:             act.Calories,
				VO2Max:               act.VO2Max,
				AerobicTE:            act.AerobicTE,
				AnaerobicTE:          act.AnaerobicTE,
				TSS:                  act.TSS,
				FTP:                  ftpVal,
				Zone1:                act.PowerTimeInZone1 / 60.0,
				Zone2:                act.PowerTimeInZone2 / 60.0,
				Zone3:                act.PowerTimeInZone3 / 60.0,
				Zone4:                act.PowerTimeInZone4 / 60.0,
				Zone5:                act.PowerTimeInZone5 / 60.0,
				Zone6:                act.PowerTimeInZone6 / 60.0,
				Zone7:                act.PowerTimeInZone7 / 60.0,
			})
		}

		var pmcJSON []PMCJSON
		for _, dm := range pmcCalculated {
			pmcJSON = append(pmcJSON, PMCJSON{
				Date: dm.Date.Format("2006-01-02"),
				TSS:  dm.TSS,
				CTL:  dm.CTL,
				ATL:  dm.ATL,
				TSB:  dm.TSB,
			})
		}

		// 週単位にグループ化
		weeklyMap := make(map[string]*WeeklyJSON)
		var weekKeys []string

		for _, dm := range pmcCalculated {
			dOffset := int(dm.Date.Weekday() - time.Monday)
			if dOffset < 0 {
				dOffset += 7
			}
			monday := dm.Date.AddDate(0, 0, -dOffset)
			weekKey := monday.Format("2006-01-02") + "～"

			wm, exists := weeklyMap[weekKey]
			if !exists {
				wm = &WeeklyJSON{
					WeekStart: weekKey,
					Zones:     make([]float64, 7),
				}
				weeklyMap[weekKey] = wm
				weekKeys = append(weekKeys, weekKey)
			}
			wm.TSS += dm.TSS
			wm.CTL = dm.CTL
			wm.ATL = dm.ATL
			wm.TSB = dm.TSB
		}

		// 各アクティビティから週次の合計値を加算
		for _, act := range acts {
			t, err := time.ParseInLocation("2006-01-02", act.Date, time.Local)
			if err == nil {
				dOffset := int(t.Weekday() - time.Monday)
				if dOffset < 0 {
					dOffset += 7
				}
				monday := t.AddDate(0, 0, -dOffset)
				weekKey := monday.Format("2006-01-02") + "～"
				if wm, exists := weeklyMap[weekKey]; exists {
					wm.Duration += act.DurationSeconds / 60.0
					wm.Distance += act.DistanceMeters / 1000.0
					wm.Elevation += act.ElevationGain
					wm.Zones[0] += act.PowerTimeInZone1 / 60.0
					wm.Zones[1] += act.PowerTimeInZone2 / 60.0
					wm.Zones[2] += act.PowerTimeInZone3 / 60.0
					wm.Zones[3] += act.PowerTimeInZone4 / 60.0
					wm.Zones[4] += act.PowerTimeInZone5 / 60.0
					wm.Zones[5] += act.PowerTimeInZone6 / 60.0
					wm.Zones[6] += act.PowerTimeInZone7 / 60.0

					ftpVal := 0.0
					if act.NormPower > 0 && act.IntensityFactor > 0 {
						ftpVal = act.NormPower / act.IntensityFactor
					}
					if ftpVal > wm.FTP {
						wm.FTP = ftpVal
					}
				}
			}
		}

		// FTPが記録されていない週に対して、前週のFTPを引き継ぐ（前方補完）
		sort.Strings(weekKeys)
		runningFTP := 0.0
		for _, k := range weekKeys {
			wm := weeklyMap[k]
			if wm.FTP > 0 {
				runningFTP = wm.FTP
			} else {
				wm.FTP = runningFTP
			}
		}

		var weeklyList []WeeklyJSON
		for _, k := range weekKeys {
			weeklyList = append(weeklyList, *weeklyMap[k])
		}

		resp := APIDataResponse{
			KPIs:       kpis,
			Activities: activitiesJSON,
			PMC:        pmcJSON,
			Weekly:     weeklyList,
		}

		_ = json.NewEncoder(w).Encode(resp)
	})

	url := fmt.Sprintf("http://localhost:%d", port)
	fmt.Printf("ダッシュボードを起動しました: %s\n", url)
	openBrowser(url)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		fmt.Printf("ブラウザを自動で開くことができませんでした。手動で開いてください: %s\n", url)
	}
}

func init() {
	statsCmd.Flags().BoolVarP(&webFlag, "web", "w", false, "ダッシュボードをブラウザで表示します")
	statsCmd.Flags().IntVarP(&portFlag, "port", "p", 8501, "Webサーバーのポート番号")
	rootCmd.AddCommand(statsCmd)
}
