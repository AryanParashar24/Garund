package server

import (
	"fmt"
	"strings"
	"time"
)

type PromQLInput struct {
	Type         string   `json:"type"` // availability, error_rate, latency, throughput, saturation, custom
	Metric       string   `json:"metric"`
	GoodStatuses []string `json:"goodStatuses,omitempty"`
	BadStatuses  []string `json:"badStatuses,omitempty"`
	Percentile   string   `json:"percentile,omitempty"` // p50, p90, p95, p99
	Window       string   `json:"window"`
	Service      string   `json:"service"`
	Namespace    string   `json:"namespace"`
	CustomQuery  string   `json:"customQuery,omitempty"`
}

type PromQLOutput struct {
	Query      string `json:"query"`
	GoodQuery  string `json:"goodQuery,omitempty"`
	TotalQuery string `json:"totalQuery,omitempty"`
}

type QueryValidationResult struct {
	Valid           bool         `json:"valid"`
	CurrentValue    *float64     `json:"currentValue"`
	SeriesCount     int          `json:"seriesCount"`
	EvaluationMs    int64        `json:"evaluationMs"`
	ErrorMessage    string       `json:"errorMessage,omitempty"`
	HasData         bool         `json:"hasData"`
	GeneratedPromQL PromQLOutput `json:"generatedPromQL"`
}

func GeneratePromQL(input PromQLInput) PromQLOutput {
	window := input.Window
	if window == "" {
		window = "5m"
	}

	labels := make([]string, 0)
	if input.Service != "" {
		labels = append(labels, fmt.Sprintf(`service="%s"`, input.Service))
	}
	if input.Namespace != "" {
		labels = append(labels, fmt.Sprintf(`namespace="%s"`, input.Namespace))
	}

	lblString := ""
	if len(labels) > 0 {
		lblString = "{" + strings.Join(labels, ",") + "}"
	}

	switch input.Type {
	case "availability":
		metric := input.Metric
		if metric == "" {
			metric = "http_requests_total"
		}
		statusRegex := "2..|3.."
		if len(input.GoodStatuses) > 0 {
			statusRegex = strings.Join(input.GoodStatuses, "|")
		}

		goodLabels := make([]string, len(labels))
		copy(goodLabels, labels)
		goodLabels = append(goodLabels, fmt.Sprintf(`status=~"%s"`, statusRegex))
		goodLblString := "{" + strings.Join(goodLabels, ",") + "}"

		goodQuery := fmt.Sprintf(`sum(rate(%s%s[%s]))`, metric, goodLblString, window)
		totalQuery := fmt.Sprintf(`sum(rate(%s%s[%s]))`, metric, lblString, window)
		combined := fmt.Sprintf(`%s / %s * 100`, goodQuery, totalQuery)

		return PromQLOutput{
			Query:      combined,
			GoodQuery:  goodQuery,
			TotalQuery: totalQuery,
		}

	case "error_rate":
		metric := input.Metric
		if metric == "" {
			metric = "http_requests_total"
		}
		statusRegex := "5.."
		if len(input.BadStatuses) > 0 {
			statusRegex = strings.Join(input.BadStatuses, "|")
		}

		badLabels := make([]string, len(labels))
		copy(badLabels, labels)
		badLabels = append(badLabels, fmt.Sprintf(`status=~"%s"`, statusRegex))
		badLblString := "{" + strings.Join(badLabels, ",") + "}"

		goodQuery := fmt.Sprintf(`sum(rate(%s%s[%s]))`, metric, badLblString, window)
		totalQuery := fmt.Sprintf(`sum(rate(%s%s[%s]))`, metric, lblString, window)
		combined := fmt.Sprintf(`%s / %s * 100`, goodQuery, totalQuery)

		return PromQLOutput{
			Query:      combined,
			GoodQuery:  goodQuery,
			TotalQuery: totalQuery,
		}

	case "latency":
		metric := input.Metric
		if metric == "" {
			metric = "http_request_duration_seconds"
		}
		if !strings.HasSuffix(metric, "_bucket") {
			metric = metric + "_bucket"
		}

		quantile := "0.95"
		switch input.Percentile {
		case "p50":
			quantile = "0.50"
		case "p90":
			quantile = "0.90"
		case "p95":
			quantile = "0.95"
		case "p99":
			quantile = "0.99"
		}

		query := fmt.Sprintf(`histogram_quantile(%s, sum(rate(%s%s[%s])) by (le)) * 1000`, quantile, metric, lblString, window)
		return PromQLOutput{
			Query: query,
		}

	case "throughput":
		metric := input.Metric
		if metric == "" {
			metric = "http_requests_total"
		}
		query := fmt.Sprintf(`sum(rate(%s%s[%s]))`, metric, lblString, window)
		return PromQLOutput{
			Query: query,
		}

	case "saturation":
		metric := input.Metric
		if metric == "" {
			metric = "container_cpu_usage_seconds_total"
		}
		query := fmt.Sprintf(`sum(rate(%s%s[%s]))`, metric, lblString, window)
		return PromQLOutput{
			Query: query,
		}

	case "custom":
		return PromQLOutput{
			Query: input.CustomQuery,
		}

	default:
		if input.CustomQuery != "" {
			return PromQLOutput{Query: input.CustomQuery}
		}
		return PromQLOutput{
			Query: fmt.Sprintf(`sum(rate(%s%s[%s]))`, input.Metric, lblString, window),
		}
	}
}

func ValidateAndTestQuery(client *PrometheusClient, input PromQLInput) QueryValidationResult {
	start := time.Now()
	output := GeneratePromQL(input)

	if output.Query == "" {
		return QueryValidationResult{
			Valid:           false,
			ErrorMessage:    "PromQL query is empty",
			GeneratedPromQL: output,
		}
	}

	if client == nil {
		// Mock offline validation if prometheus client is nil
		return QueryValidationResult{
			Valid:           true,
			EvaluationMs:    time.Since(start).Milliseconds(),
			ErrorMessage:    "Prometheus client disconnected",
			GeneratedPromQL: output,
		}
	}

	val, hasData, err := client.QueryOptional(output.Query)
	evalTime := time.Since(start).Milliseconds()

	if err != nil {
		return QueryValidationResult{
			Valid:           false,
			EvaluationMs:    evalTime,
			ErrorMessage:    err.Error(),
			GeneratedPromQL: output,
		}
	}

	seriesCount := 0
	if input.Metric != "" {
		seriesCount, _ = client.Series([]string{input.Metric})
	}

	var curVal *float64
	if hasData {
		curVal = &val
	}

	return QueryValidationResult{
		Valid:           true,
		CurrentValue:    curVal,
		HasData:         hasData,
		SeriesCount:     seriesCount,
		EvaluationMs:    evalTime,
		GeneratedPromQL: output,
	}
}
