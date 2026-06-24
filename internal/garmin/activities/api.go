package activities

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"garmin-cli/internal/garmin/client"
)

type TypeInfo struct {
	TypeKey string `json:"typeKey"`
}

type ListItem struct {
	ActivityID           int64    `json:"activityId"`
	ActivityName         string   `json:"activityName"`
	StartTimeLocal       string   `json:"startTimeLocal"`
	ActivityType         TypeInfo `json:"activityType"`
	Distance             float64  `json:"distance"` // meters
	Duration             float64  `json:"duration"` // seconds
	ElevationGain        float64  `json:"elevationGain"`
	AverageSpeed         float64  `json:"averageSpeed"` // m/s
	MaxSpeed             float64  `json:"maxSpeed"`     // m/s
	AverageBikingCadence float64  `json:"averageBikingCadenceInRevPerMinute"`
	MaxBikingCadence     float64  `json:"maxBikingCadenceInRevPerMinute"`
	AvgPower             float64  `json:"avgPower"`
	MaxPower             float64  `json:"maxPower"`
	Max20MinPower        float64  `json:"max20MinPower"`
	NormPower            float64  `json:"normPower"`
	IntensityFactor      float64  `json:"intensityFactor"`
	PowerTimeInZone1     float64  `json:"powerTimeInZone_1"`
	PowerTimeInZone2     float64  `json:"powerTimeInZone_2"`
	PowerTimeInZone3     float64  `json:"powerTimeInZone_3"`
	PowerTimeInZone4     float64  `json:"powerTimeInZone_4"`
	PowerTimeInZone5     float64  `json:"powerTimeInZone_5"`
	PowerTimeInZone6     float64  `json:"powerTimeInZone_6"`
	PowerTimeInZone7     float64  `json:"powerTimeInZone_7"`
	AverageHR            float64  `json:"averageHR"`
	MaxHR                float64  `json:"maxHR"`
	Calories             float64  `json:"calories"`
	VO2Max               float64  `json:"vO2MaxValue"`
	AerobicTE            float64  `json:"aerobicTrainingEffect"`
	AnaerobicTE          float64  `json:"anaerobicTrainingEffect"`
	TSS                  float64  `json:"trainingStressScore"`
}

type Summary struct {
	ID                   int64
	Name                 string
	Date                 string
	StartTimeLocal       string
	Type                 string
	DistanceMeters       float64
	DurationSeconds      float64
	ElevationGain        float64
	AverageSpeed         float64
	MaxSpeed             float64
	AverageBikingCadence float64
	MaxBikingCadence     float64
	AvgPower             float64
	MaxPower             float64
	Max20MinPower        float64
	NormPower            float64
	IntensityFactor      float64
	PowerTimeInZone1     float64
	PowerTimeInZone2     float64
	PowerTimeInZone3     float64
	PowerTimeInZone4     float64
	PowerTimeInZone5     float64
	PowerTimeInZone6     float64
	PowerTimeInZone7     float64
	AverageHR            float64
	MaxHR                float64
	Calories             float64
	VO2Max               float64
	AerobicTE            float64
	AnaerobicTE          float64
	TSS                  float64
}

func List(ctx context.Context, c *client.Client, limit int, after, before, activityType string) ([]Summary, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be > 0")
	}

	if after != "" {
		d, err := parseDate(after)
		if err != nil {
			return nil, err
		}
		after = d
	}
	if before != "" {
		d, err := parseDate(before)
		if err != nil {
			return nil, err
		}
		before = d
	}
	if after != "" && before != "" && before < after {
		return nil, fmt.Errorf("--before (%s) is before --after (%s)", before, after)
	}

	var out []Summary
	start := 0
	pageSize := 50
	if limit < pageSize {
		pageSize = limit
	}

	for len(out) < limit {
		var page []ListItem
		q := url.Values{
			"limit": {strconv.Itoa(pageSize)},
			"start": {strconv.Itoa(start)},
		}
		if err := c.GetJSON(ctx, "/activitylist-service/activities/search/activities", q, &page); err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		for _, item := range page {
			s := item.ToSummary()
			if !passesFilters(s, after, before, activityType) {
				continue
			}
			out = append(out, s)
			if len(out) >= limit {
				break
			}
		}

		if after != "" {
			oldest := page[len(page)-1].startDate()
			if oldest != "" && oldest < after {
				break
			}
		}

		start += len(page)
		if len(page) < pageSize {
			break
		}
	}

	return out, nil
}

func GetUserWeight(ctx context.Context, c *client.Client) (float64, error) {
	var settings map[string]any
	if err := c.GetJSON(ctx, "/userprofile-service/userprofile/settings", nil, &settings); err != nil {
		return 0, err
	}

	dto, ok := settings["userProfileSettings"].(map[string]any)
	if !ok || dto == nil {
		dto, ok = settings["userProfileDTO"].(map[string]any)
	}

	if ok && dto != nil {
		if rawWeight, exists := dto["weight"]; exists {
			if w, err := strconv.ParseFloat(fmt.Sprintf("%v", rawWeight), 64); err == nil {
				if w > 1000 {
					return w / 1000.0, nil
				}
				return w, nil
			}
		}
	}
	return 0, fmt.Errorf("weight field not found or invalid in user profile settings")
}

func (a ListItem) startDate() string {
	if len(a.StartTimeLocal) >= 10 {
		return a.StartTimeLocal[:10]
	}
	return ""
}

func (a ListItem) ToSummary() Summary {
	return Summary{
		ID:                   a.ActivityID,
		Name:                 a.ActivityName,
		Date:                 a.startDate(),
		StartTimeLocal:       a.StartTimeLocal,
		Type:                 a.ActivityType.TypeKey,
		DistanceMeters:       a.Distance,
		DurationSeconds:      a.Duration,
		ElevationGain:        a.ElevationGain,
		AverageSpeed:         a.AverageSpeed,
		MaxSpeed:             a.MaxSpeed,
		AverageBikingCadence: a.AverageBikingCadence,
		MaxBikingCadence:     a.MaxBikingCadence,
		AvgPower:             a.AvgPower,
		MaxPower:             a.MaxPower,
		Max20MinPower:        a.Max20MinPower,
		NormPower:            a.NormPower,
		IntensityFactor:      a.IntensityFactor,
		PowerTimeInZone1:     a.PowerTimeInZone1,
		PowerTimeInZone2:     a.PowerTimeInZone2,
		PowerTimeInZone3:     a.PowerTimeInZone3,
		PowerTimeInZone4:     a.PowerTimeInZone4,
		PowerTimeInZone5:     a.PowerTimeInZone5,
		PowerTimeInZone6:     a.PowerTimeInZone6,
		PowerTimeInZone7:     a.PowerTimeInZone7,
		AverageHR:            a.AverageHR,
		MaxHR:                a.MaxHR,
		Calories:             a.Calories,
		VO2Max:               a.VO2Max,
		AerobicTE:            a.AerobicTE,
		AnaerobicTE:          a.AnaerobicTE,
		TSS:                  a.TSS,
	}
}

func passesFilters(a Summary, after, before, activityType string) bool {
	if after != "" && a.Date != "" && a.Date < after {
		return false
	}
	if before != "" && a.Date != "" && a.Date > before {
		return false
	}
	if strings.TrimSpace(activityType) != "" {
		t := strings.ToLower(strings.TrimSpace(activityType))
		typ := strings.ToLower(a.Type)
		if typ != t && !strings.Contains(typ, t) {
			return false
		}
	}
	return true
}

func parseDate(s string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return "", fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return t.Format("2006-01-02"), nil
}
