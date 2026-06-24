package data

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"

	"garmin-cli/internal/garmin/activities"
)

var CSVHeaders = []string{
	"ｱｸﾃｨﾋﾞﾃｨID",
	"ｱｸﾃｨﾋﾞﾃｨ名",
	"開始時間",
	"距離(km)",
	"時間(分)",
	"獲得標高(m)",
	"平均速度(km/h)",
	"最高速度(km/h)",
	"平均ケイデンス(rpm)",
	"最大ケイデンス(rpm)",
	"平均パワー(W)",
	"最大パワー(W)",
	"最大平均パワー(20分)(W)",
	"NormalizedPower(W)",
	"intensityFactor",
	"ゾーン1時間(分)",
	"ゾーン2時間(分)",
	"ゾーン3時間(分)",
	"ゾーン4時間(分)",
	"ゾーン5時間(分)",
	"ゾーン6時間(分)",
	"ゾーン7時間(分)",
	"平均心拍数",
	"最大心拍数",
	"消費カロリー",
	"VO2Max",
	"有酸素TE",
	"無酸素TE",
	"ﾄﾚｰﾆﾝｸﾞｽﾄﾚｽｽｺｱ",
	"FTP(W)",
}

// LoadCSV は、BOM付きUTF-8のCSVファイルからアクティビティデータを読み込みます。
func LoadCSV(filename string) ([]activities.Summary, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// BOM（Byte Order Mark）が存在する場合はスキップ
	bom := make([]byte, 3)
	if _, err := f.Read(bom); err != nil {
		return nil, err
	}
	if bom[0] != 0xEF || bom[1] != 0xBB || bom[2] != 0xBF {
		// BOMがない場合はファイルの読み込み位置を先頭に戻す
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, err
		}
	}

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) <= 1 {
		return nil, nil // データが空、またはヘッダー行のみの場合
	}

	headerMap := make(map[string]int)
	for i, h := range records[0] {
		headerMap[h] = i
	}

	var list []activities.Summary
	for _, rec := range records[1:] {
		val := func(header string) string {
			if idx, ok := headerMap[header]; ok && idx < len(rec) {
				return rec[idx]
			}
			return ""
		}
		valF := func(header string) float64 {
			s := val(header)
			if s == "" {
				return 0
			}
			f, _ := strconv.ParseFloat(s, 64)
			return f
		}
		valI64 := func(header string) int64 {
			s := val(header)
			if s == "" {
				return 0
			}
			i, _ := strconv.ParseInt(s, 10, 64)
			return i
		}

		s := activities.Summary{
			ID:                   valI64("ｱｸﾃｨﾋﾞﾃｨID"),
			Name:                 val("ｱｸﾃｨﾋﾞﾃｨ名"),
			StartTimeLocal:       val("開始時間"),
			DistanceMeters:       valF("距離(km)") * 1000.0,
			DurationSeconds:      valF("時間(分)") * 60.0,
			ElevationGain:        valF("獲得標高(m)"),
			AverageSpeed:         valF("平均速度(km/h)") / 3.6,
			MaxSpeed:             valF("最高速度(km/h)") / 3.6,
			AverageBikingCadence: valF("平均ケイデンス(rpm)"),
			MaxBikingCadence:     valF("最大ケイデンス(rpm)"),
			AvgPower:             valF("平均パワー(W)"),
			MaxPower:             valF("最大パワー(W)"),
			Max20MinPower:        valF("最大平均パワー(20分)(W)"),
			NormPower:            valF("NormalizedPower(W)"),
			IntensityFactor:      valF("intensityFactor"),
			PowerTimeInZone1:     valF("ゾーン1時間(分)") * 60.0,
			PowerTimeInZone2:     valF("ゾーン2時間(分)") * 60.0,
			PowerTimeInZone3:     valF("ゾーン3時間(分)") * 60.0,
			PowerTimeInZone4:     valF("ゾーン4時間(分)") * 60.0,
			PowerTimeInZone5:     valF("ゾーン5時間(分)") * 60.0,
			PowerTimeInZone6:     valF("ゾーン6時間(分)") * 60.0,
			PowerTimeInZone7:     valF("ゾーン7時間(分)") * 60.0,
			AverageHR:            valF("平均心拍数"),
			MaxHR:                valF("最大心拍数"),
			Calories:             valF("消費カロリー"),
			VO2Max:               valF("VO2Max"),
			AerobicTE:            valF("有酸素TE"),
			AnaerobicTE:          valF("無酸素TE"),
			TSS:                  valF("ﾄﾚｰﾆﾝｸﾞｽﾄﾚｽｽｺｱ"),
		}
		// 開始時間（StartTimeLocal）から日付部分（YYYY-MM-DD）の抽出を試みる
		if len(s.StartTimeLocal) >= 10 {
			s.Date = s.StartTimeLocal[:10]
		}

		list = append(list, s)
	}

	return list, nil
}

// WriteCSV は、アクティビティデータをBOM付きUTF-8のCSVファイルに書き出します。
func WriteCSV(filename string, list []activities.Summary) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// UTF-8 の BOM を書き込み（Excelなどでの文字化けを防ぐため）
	if _, err := f.Write([]byte{0xEF, 0xBB, 0xBF}); err != nil {
		return err
	}

	writer := csv.NewWriter(f)
	defer writer.Flush()

	if err := writer.Write(CSVHeaders); err != nil {
		return err
	}

	for _, s := range list {
		// 可能であればNPとIFから当時の推定FTPを計算
		ftpVal := 0.0
		if s.NormPower > 0 && s.IntensityFactor > 0 {
			ftpVal = s.NormPower / s.IntensityFactor
		}

		row := []string{
			strconv.FormatInt(s.ID, 10),
			s.Name,
			s.StartTimeLocal,
			fmt.Sprintf("%.5f", s.DistanceMeters/1000.0),
			fmt.Sprintf("%.5f", s.DurationSeconds/60.0),
			fmt.Sprintf("%.1f", s.ElevationGain),
			fmt.Sprintf("%.5f", s.AverageSpeed*3.6),
			fmt.Sprintf("%.5f", s.MaxSpeed*3.6),
			fmt.Sprintf("%.1f", s.AverageBikingCadence),
			fmt.Sprintf("%.1f", s.MaxBikingCadence),
			fmt.Sprintf("%.1f", s.AvgPower),
			fmt.Sprintf("%.1f", s.MaxPower),
			fmt.Sprintf("%.1f", s.Max20MinPower),
			fmt.Sprintf("%.1f", s.NormPower),
			fmt.Sprintf("%.5f", s.IntensityFactor),
			fmt.Sprintf("%.5f", s.PowerTimeInZone1/60.0),
			fmt.Sprintf("%.5f", s.PowerTimeInZone2/60.0),
			fmt.Sprintf("%.5f", s.PowerTimeInZone3/60.0),
			fmt.Sprintf("%.5f", s.PowerTimeInZone4/60.0),
			fmt.Sprintf("%.5f", s.PowerTimeInZone5/60.0),
			fmt.Sprintf("%.5f", s.PowerTimeInZone6/60.0),
			fmt.Sprintf("%.5f", s.PowerTimeInZone7/60.0),
			fmt.Sprintf("%.1f", s.AverageHR),
			fmt.Sprintf("%.1f", s.MaxHR),
			fmt.Sprintf("%.1f", s.Calories),
			fmt.Sprintf("%.1f", s.VO2Max),
			fmt.Sprintf("%.1f", s.AerobicTE),
			fmt.Sprintf("%.1f", s.AnaerobicTE),
			fmt.Sprintf("%.1f", s.TSS),
			fmt.Sprintf("%.0f", ftpVal),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}
