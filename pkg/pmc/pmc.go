package pmc

import (
	"math"
	"time"
)

type DailyMetrics struct {
	Date time.Time
	TSS  float64
	CTL  float64
	ATL  float64
	TSB  float64
}

// CalculatePMC は、日ごとのTSSからCTL、ATL、およびTSBを計算します。
// 入力の dailyTSS は日付の昇順でソートされている必要があります。
func CalculatePMC(dailyTSS []DailyMetrics) []DailyMetrics {
	if len(dailyTSS) == 0 {
		return nil
	}

	alphaCTL := 1.0 - math.Exp(-1.0/42.0)
	alphaATL := 1.0 - math.Exp(-1.0/7.0)

	result := make([]DailyMetrics, len(dailyTSS))
	copy(result, dailyTSS)

	// 計算開始日（初期状態）の設定
	// Pythonの ewm(adjust=False) では y_0 = x_0 となるため、
	// CTL_0 = TSS_0, ATL_0 = TSS_0、TSB_0 = 0.0 となります。
	result[0].CTL = result[0].TSS
	result[0].ATL = result[0].TSS
	result[0].TSB = result[0].CTL - result[0].ATL // 0.0

	for i := 1; i < len(result); i++ {
		result[i].CTL = (1.0-alphaCTL)*result[i-1].CTL + alphaCTL*result[i].TSS
		result[i].ATL = (1.0-alphaATL)*result[i-1].ATL + alphaATL*result[i].TSS
		result[i].TSB = result[i-1].CTL - result[i-1].ATL
	}

	return result
}
