package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// PrometheusStatusType defines the health status of a Prometheus connection.
type PrometheusStatusType string

const (
	PrometheusStatusConnected    PrometheusStatusType = "CONNECTED"
	PrometheusStatusDegraded     PrometheusStatusType = "DEGRADED"
	PrometheusStatusDisconnected PrometheusStatusType = "DISCONNECTED"
	PrometheusStatusAuthError    PrometheusStatusType = "AUTH_ERROR"
	PrometheusStatusUnknown      PrometheusStatusType = "UNKNOWN"
)

type PrometheusConfig struct {
	ClusterID     string               `json:"clusterId"`
	URL           string               `json:"url"`
	Status        PrometheusStatusType `json:"status"`
	Version       string               `json:"version,omitempty"`
	LastQueryTime *time.Time           `json:"lastQueryTime,omitempty"`
	LastError     string               `json:"lastError,omitempty"`
}

type PrometheusClient struct {
	BaseURL string
	Client  *http.Client
	mu      sync.RWMutex
	status  PrometheusStatusType
	lastErr error
	lastSuccess *time.Time
}

type PrometheusResponse struct {
	Status    string                 `json:"status"`
	ErrorType string                 `json:"errorType,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Data      PrometheusResponseData `json:"data"`
}

type PrometheusResponseData struct {
	ResultType string             `json:"resultType"`
	Result     []PrometheusQueryResult `json:"result"`
}

type PrometheusQueryResult struct {
	Metric map[string]string `json:"metric"`
	Value  []interface{}     `json:"value,omitempty"`
	Values [][]interface{}   `json:"values,omitempty"`
}

type PrometheusRangePoint struct {
	Timestamp float64 `json:"timestamp"`
	Value     float64 `json:"value"`
}

type PrometheusSeriesData struct {
	Timestamp float64
	Value     float64
}

func NewPrometheusClient(baseURL string) *PrometheusClient {
	if baseURL == "" {
		baseURL = "http://localhost:9090"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	return &PrometheusClient{
		BaseURL: baseURL,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		status: PrometheusStatusUnknown,
	}
}

func (p *PrometheusClient) Health(ctx context.Context) (PrometheusStatusType, string, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	reqURL := fmt.Sprintf("%s/-/healthy", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		p.setStatus(PrometheusStatusDisconnected, err)
		return PrometheusStatusDisconnected, "", err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		p.setStatus(PrometheusStatusDisconnected, err)
		return PrometheusStatusDisconnected, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		p.setStatus(PrometheusStatusAuthError, fmt.Errorf("auth error: %s", resp.Status))
		return PrometheusStatusAuthError, "", fmt.Errorf("auth error: %s", resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		p.setStatus(PrometheusStatusDegraded, fmt.Errorf("unhealthy status: %s", resp.Status))
		return PrometheusStatusDegraded, "", fmt.Errorf("unhealthy status: %s", resp.Status)
	}

	// Fetch build version
	version := "3.x"
	flagsURL := fmt.Sprintf("%s/api/v1/status/buildinfo", p.BaseURL)
	if flagsReq, err := http.NewRequestWithContext(ctx, "GET", flagsURL, nil); err == nil {
		if flagsResp, err := p.Client.Do(flagsReq); err == nil && flagsResp.StatusCode == http.StatusOK {
			var buildInfo struct {
				Data struct {
					Version string `json:"version"`
				} `json:"data"`
			}
			if json.NewDecoder(flagsResp.Body).Decode(&buildInfo) == nil && buildInfo.Data.Version != "" {
				version = buildInfo.Data.Version
			}
			flagsResp.Body.Close()
		}
	}

	p.setStatus(PrometheusStatusConnected, nil)
	return PrometheusStatusConnected, version, nil
}

func (p *PrometheusClient) setStatus(status PrometheusStatusType, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.status = status
	p.lastErr = err
	if err == nil {
		now := time.Now()
		p.lastSuccess = &now
	}
}

func (p *PrometheusClient) Query(query string) (float64, error) {
	val, ok, err := p.QueryOptional(query)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}
	return val, nil
}

func (p *PrometheusClient) QueryOptional(query string) (float64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	endpoint, err := url.Parse(p.BaseURL + "/api/v1/query")
	if err != nil {
		return 0, false, err
	}

	params := endpoint.Query()
	params.Set("query", query)
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if err != nil {
		p.setStatus(PrometheusStatusDisconnected, err)
		return 0, false, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		p.setStatus(PrometheusStatusDisconnected, err)
		return 0, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("prometheus returned %s", resp.Status)
		p.setStatus(PrometheusStatusDegraded, err)
		return 0, false, err
	}

	var data PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, false, err
	}

	if data.Status != "success" {
		err := fmt.Errorf("prometheus query error: %s", data.Error)
		return 0, false, err
	}

	p.setStatus(PrometheusStatusConnected, nil)

	if len(data.Data.Result) == 0 || len(data.Data.Result[0].Value) < 2 {
		return 0, false, nil
	}

	valStr, ok := data.Data.Result[0].Value[1].(string)
	if !ok {
		return 0, false, fmt.Errorf("unexpected prometheus value format")
	}

	var result float64
	if _, err := fmt.Sscanf(valStr, "%f", &result); err != nil {
		return 0, false, err
	}

	return result, true, nil
}

func (p *PrometheusClient) QueryRange(query string, start, end time.Time, step time.Duration) ([]PrometheusRangePoint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	endpoint, err := url.Parse(p.BaseURL + "/api/v1/query_range")
	if err != nil {
		return nil, err
	}

	if step <= 0 {
		step = 60 * time.Second
	}

	params := endpoint.Query()
	params.Set("query", query)
	params.Set("start", fmt.Sprintf("%d", start.Unix()))
	params.Set("end", fmt.Sprintf("%d", end.Unix()))
	params.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus range query returned %s", resp.Status)
	}

	var data PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if data.Status != "success" {
		return nil, fmt.Errorf("prometheus range query failed: %s", data.Error)
	}

	var points []PrometheusRangePoint
	if len(data.Data.Result) > 0 {
		for _, rawPoint := range data.Data.Result[0].Values {
			if len(rawPoint) >= 2 {
				ts, tsOk := rawPoint[0].(float64)
				valStr, valOk := rawPoint[1].(string)
				if tsOk && valOk {
					var val float64
					fmt.Sscanf(valStr, "%f", &val)
					points = append(points, PrometheusRangePoint{
						Timestamp: ts,
						Value:     val,
					})
				}
			}
		}
	}

	return points, nil
}

func (p *PrometheusClient) LabelNames() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/api/v1/labels", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (p *PrometheusClient) LabelValues(label string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reqURL := fmt.Sprintf("%s/api/v1/label/%s/values", p.BaseURL, url.PathEscape(label))
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (p *PrometheusClient) Metrics() ([]string, error) {
	return p.LabelValues("__name__")
}

func (p *PrometheusClient) Series(matchers []string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	endpoint, err := url.Parse(p.BaseURL + "/api/v1/series")
	if err != nil {
		return 0, err
	}

	params := endpoint.Query()
	for _, m := range matchers {
		params.Add("match[]", m)
	}
	endpoint.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status string              `json:"status"`
		Data   []map[string]string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return 0, err
	}

	return len(apiResp.Data), nil
}

func (p *PrometheusClient) Readiness(ctx context.Context) (bool, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	reqURL := fmt.Sprintf("%s/-/ready", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return false, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

func (p *PrometheusClient) Metadata(ctx context.Context, metric string) (map[string]interface{}, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	endpoint, err := url.Parse(p.BaseURL + "/api/v1/metadata")
	if err != nil {
		return nil, err
	}

	if metric != "" {
		params := endpoint.Query()
		params.Set("metric", metric)
		endpoint.RawQuery = params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status string                 `json:"status"`
		Data   map[string]interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (p *PrometheusClient) Rules(ctx context.Context) (interface{}, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	reqURL := fmt.Sprintf("%s/api/v1/rules", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status string      `json:"status"`
		Data   interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (p *PrometheusClient) Alerts(ctx context.Context) (interface{}, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	reqURL := fmt.Sprintf("%s/api/v1/alerts", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status string      `json:"status"`
		Data   interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

func (p *PrometheusClient) Targets(ctx context.Context) (interface{}, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	reqURL := fmt.Sprintf("%s/api/v1/targets", p.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Status string      `json:"status"`
		Data   interface{} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Data, nil
}

