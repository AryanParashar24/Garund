package server

import (
	"fmt"
	"math"
	"time"
)

type MultiWindowBurnRate struct {
	Window1h  *float64 `json:"window1h"`
	Window6h  *float64 `json:"window6h"`
	Window24h *float64 `json:"window24h"`
}

type EvaluatedSLI struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Type             string   `json:"type"`
	Value            *float64 `json:"value"`
	Target           float64  `json:"target"`
	Unit             string   `json:"unit"`
	GoodEvents       int64    `json:"goodEvents"`
	TotalEvents      int64    `json:"totalEvents"`
	EvaluationWindow string   `json:"evaluationWindow"`
	Status           string   `json:"status"` // healthy, warning, critical, unavailable
	Query            string   `json:"query"`
	GoodQuery        string   `json:"goodQuery,omitempty"`
	TotalQuery       string   `json:"totalQuery,omitempty"`
	EvaluatedAt      time.Time `json:"evaluatedAt"`
}

type EvaluatedSLO struct {
	ID                   string              `json:"id"`
	Name                 string              `json:"name"`
	Service              string              `json:"service"`
	Namespace            string              `json:"namespace"`
	SLIID                string              `json:"sliId"`
	SLIName              string              `json:"sliName"`
	SLIType              string              `json:"sliType"`
	Target               float64             `json:"target"`
	Window               string              `json:"window"`
	Current              *float64            `json:"current"`
	AllowedError         float64             `json:"allowedError"`
	ErrorBudgetRemaining float64             `json:"errorBudgetRemaining"`
	TotalBudgetMinutes   float64             `json:"totalBudgetMinutes"`
	RemainingMinutes     float64             `json:"remainingMinutes"`
	ConsumedMinutes      float64             `json:"consumedMinutes"`
	BurnRate             MultiWindowBurnRate `json:"burnRate"`
	Status               string              `json:"status"` // healthy, at_risk, exhausted, unavailable
	Owner                string              `json:"owner,omitempty"`
	Team                 string              `json:"team,omitempty"`
	EvaluatedAt          time.Time           `json:"evaluatedAt"`
}

type EvaluatedSLA struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Service            string   `json:"service"`
	Namespace          string   `json:"namespace"`
	AvailabilityTarget *float64 `json:"availabilityTarget,omitempty"`
	LatencyTargetMs    *float64 `json:"latencyTargetMs,omitempty"`
	Window             string   `json:"window"`
	SafetyMargin       *float64 `json:"safetyMargin,omitempty"` // percentage points margin over SLA
	Status             string   `json:"status"`               // compliant, at_risk, breached, unavailable
	EvaluatedAt        time.Time `json:"evaluatedAt"`
}

type FullReliabilityOverview struct {
	ClusterID   string         `json:"clusterId"`
	Service     string         `json:"service"`
	Namespace   string         `json:"namespace"`
	EvaluatedAt time.Time      `json:"evaluatedAt"`
	SLIs        []EvaluatedSLI `json:"slis"`
	SLOs        []EvaluatedSLO `json:"slos"`
	SLAs        []EvaluatedSLA `json:"slas"`
	Summary     struct {
		OverallHealthScore int `json:"overallHealthScore"`
		TotalSLOs          int `json:"totalSlos"`
		HealthySLOs        int `json:"healthySlos"`
		AtRiskSLOs         int `json:"atRiskSlos"`
		ExhaustedSLOs      int `json:"exhaustedSlos"`
		ActiveAlerts       int `json:"activeAlerts"`
	} `json:"summary"`
}

func CalculateSLIStatus(value *float64, target float64, sliType string) string {
	if value == nil {
		return "unavailable"
	}

	val := *value

	switch sliType {
	case "availability":
		if val >= target {
			return "healthy"
		}
		if target-val <= 0.5 {
			return "warning"
		}
		return "critical"

	case "error_rate":
		if val <= target {
			return "healthy"
		}
		if val-target <= 0.5 {
			return "warning"
		}
		return "critical"

	case "latency":
		if val <= target {
			return "healthy"
		}
		if val-target <= 50 {
			return "warning"
		}
		return "critical"

	default:
		if val >= target {
			return "healthy"
		}
		return "critical"
	}
}

func CalculateErrorBudgetRemaining(target float64, current float64) float64 {
	if target >= 100 || current >= 100 {
		return 100
	}

	allowedError := 100 - target
	if allowedError <= 0 {
		return 100
	}

	actualError := 100 - current
	if actualError <= 0 {
		return 100
	}

	consumedPct := (actualError / allowedError) * 100
	remaining := 100 - consumedPct

	if remaining < 0 {
		return 0
	}
	if remaining > 100 {
		return 100
	}
	return roundTwoDecimals(remaining)
}

func CalculateWindowMinutes(windowStr string) float64 {
	switch windowStr {
	case "5m":
		return 5
	case "15m":
		return 15
	case "1h":
		return 60
	case "6h":
		return 360
	case "24h":
		return 1440
	case "7d":
		return 10080
	case "30d":
		return 43200
	case "90d":
		return 129600
	default:
		return 43200
	}
}

func CalculateBurnRate(target float64, current float64) float64 {
	allowedError := 100 - target
	if allowedError <= 0 {
		return 0
	}

	actualError := 100 - current
	if actualError <= 0 {
		return 0
	}

	burnRate := actualError / allowedError
	return roundTwoDecimals(burnRate)
}

func EvaluateSLO(item SLOItem, currentSLI *float64, sliName string, sliType string) EvaluatedSLO {
	now := time.Now()
	allowedError := roundTwoDecimals(100 - item.Target)
	windowMinutes := CalculateWindowMinutes(item.Window)
	totalBudgetMinutes := roundTwoDecimals(windowMinutes * (allowedError / 100))

	if currentSLI == nil {
		return EvaluatedSLO{
			ID:                   item.ID,
			Name:                 item.Name,
			Service:              item.Service,
			Namespace:            item.Namespace,
			SLIID:                item.SLIID,
			SLIName:              sliName,
			SLIType:              sliType,
			Target:               item.Target,
			Window:               item.Window,
			Current:              nil,
			AllowedError:         allowedError,
			ErrorBudgetRemaining: 0,
			TotalBudgetMinutes:   totalBudgetMinutes,
			RemainingMinutes:     0,
			ConsumedMinutes:      0,
			BurnRate: MultiWindowBurnRate{
				Window1h:  nil,
				Window6h:  nil,
				Window24h: nil,
			},
			Status:      "unavailable",
			Owner:       item.Owner,
			Team:        item.Team,
			EvaluatedAt: now,
		}
	}

	curr := *currentSLI
	remainingPct := CalculateErrorBudgetRemaining(item.Target, curr)
	remainingMin := roundTwoDecimals(totalBudgetMinutes * (remainingPct / 100))
	consumedMin := roundTwoDecimals(totalBudgetMinutes - remainingMin)

	brVal := CalculateBurnRate(item.Target, curr)
	br1h := brVal
	br6h := roundTwoDecimals(brVal * 0.8)
	br24h := roundTwoDecimals(brVal * 0.5)

	status := "healthy"
	if remainingPct <= 0 {
		status = "exhausted"
	} else if remainingPct < 20 {
		status = "at_risk"
	}

	return EvaluatedSLO{
		ID:                   item.ID,
		Name:                 item.Name,
		Service:              item.Service,
		Namespace:            item.Namespace,
		SLIID:                item.SLIID,
		SLIName:              sliName,
		SLIType:              sliType,
		Target:               item.Target,
		Window:               item.Window,
		Current:              &curr,
		AllowedError:         allowedError,
		ErrorBudgetRemaining: remainingPct,
		TotalBudgetMinutes:   totalBudgetMinutes,
		RemainingMinutes:     remainingMin,
		ConsumedMinutes:      consumedMin,
		BurnRate: MultiWindowBurnRate{
			Window1h:  &br1h,
			Window6h:  &br6h,
			Window24h: &br24h,
		},
		Status:      status,
		Owner:       item.Owner,
		Team:        item.Team,
		EvaluatedAt: now,
	}
}

func EvaluateSLA(item SLAItem, currentAvailability *float64, currentLatency *float64, sloTarget float64) EvaluatedSLA {
	now := time.Now()
	if currentAvailability == nil {
		return EvaluatedSLA{
			ID:                 item.ID,
			Name:               item.Name,
			Service:            item.Service,
			Namespace:          item.Namespace,
			AvailabilityTarget: item.AvailabilityTarget,
			LatencyTargetMs:    item.LatencyTargetMs,
			Window:             item.Window,
			SafetyMargin:       nil,
			Status:             "unavailable",
			EvaluatedAt:        now,
		}
	}

	currAvail := *currentAvailability
	targetAvail := 99.9
	if item.AvailabilityTarget != nil {
		targetAvail = *item.AvailabilityTarget
	}

	var safetyMargin *float64
	if sloTarget > targetAvail {
		margin := roundTwoDecimals(sloTarget - targetAvail)
		safetyMargin = &margin
	}

	status := "compliant"
	if currAvail < targetAvail {
		status = "breached"
	} else if currAvail < (targetAvail + 0.1) {
		status = "at_risk"
	}

	return EvaluatedSLA{
		ID:                 item.ID,
		Name:               item.Name,
		Service:            item.Service,
		Namespace:          item.Namespace,
		AvailabilityTarget: item.AvailabilityTarget,
		LatencyTargetMs:    item.LatencyTargetMs,
		Window:             item.Window,
		SafetyMargin:       safetyMargin,
		Status:             status,
		EvaluatedAt:        now,
	}
}

func roundTwoDecimals(v float64) float64 {
	return math.Round(v*100) / 100
}

func FormatHumanDuration(d time.Duration) string {
	if d.Hours() >= 24 {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
	if d.Hours() >= 1 {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.0fm", d.Minutes())
}

type LegacySLIMeasurement struct {
	Value       *float64
	GoodEvents  int64
	TotalEvents int64
	Status      string
}

func calculateAvailabilitySLI(good, total float64, target float64, available bool) LegacySLIMeasurement {
	if !available || total <= 0 {
		return LegacySLIMeasurement{Value: nil, Status: "unavailable"}
	}
	val := (good / total) * 100
	if val > 100 {
		val = 100
	}
	status := CalculateSLIStatus(&val, target, "availability")
	return LegacySLIMeasurement{
		Value:       &val,
		GoodEvents:  int64(good),
		TotalEvents: int64(total),
		Status:      status,
	}
}

func calculateErrorRateSLI(bad, total float64, target float64, available bool) LegacySLIMeasurement {
	if !available || total <= 0 {
		return LegacySLIMeasurement{Value: nil, Status: "unavailable"}
	}
	val := (bad / total) * 100
	status := CalculateSLIStatus(&val, target, "error_rate")
	return LegacySLIMeasurement{
		Value:       &val,
		GoodEvents:  int64(bad),
		TotalEvents: int64(total),
		Status:      status,
	}
}

func calculateLatencySLI(val float64, target float64, available bool) LegacySLIMeasurement {
	if !available || val <= 0 {
		return LegacySLIMeasurement{Value: nil, Status: "unavailable"}
	}
	status := CalculateSLIStatus(&val, target, "latency")
	return LegacySLIMeasurement{
		Value:  &val,
		Status: status,
	}
}
