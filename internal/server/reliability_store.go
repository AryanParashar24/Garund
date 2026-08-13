package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type SLIItem struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description,omitempty"`
	ClusterID        string    `json:"clusterId"`
	Service          string    `json:"service"`
	Namespace        string    `json:"namespace"`
	Type             string    `json:"type"` // availability, error_rate, latency, throughput, saturation, custom
	Target           float64   `json:"target,omitempty"`
	Query            string    `json:"query,omitempty"`
	GoodQuery        string    `json:"goodQuery,omitempty"`
	TotalQuery       string    `json:"totalQuery,omitempty"`
	Unit             string    `json:"unit"`
	EvaluationWindow string    `json:"evaluationWindow"`
	Enabled          bool      `json:"enabled"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SLOItem struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	ClusterID   string    `json:"clusterId"`
	Service     string    `json:"service"`
	Namespace   string    `json:"namespace"`
	SLIID       string    `json:"sliId"`
	Target      float64   `json:"target"`
	Window      string    `json:"window"`
	Owner       string    `json:"owner,omitempty"`
	Team        string    `json:"team,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SLAItem struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description,omitempty"`
	ClusterID          string    `json:"clusterId"`
	Service            string    `json:"service"`
	Namespace          string    `json:"namespace"`
	AvailabilityTarget *float64  `json:"availabilityTarget,omitempty"`
	LatencyTargetMs    *float64  `json:"latencyTargetMs,omitempty"`
	Window             string    `json:"window"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type AlertPolicyItem struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	ClusterID     string    `json:"clusterId"`
	Service       string    `json:"service"`
	Namespace     string    `json:"namespace"`
	SLOID         string    `json:"sloId,omitempty"`
	SLIID         string    `json:"sliId,omitempty"`
	ConditionType string    `json:"conditionType"` // burn_rate, sli_threshold, slo_breach, etc.
	Threshold     float64   `json:"threshold"`
	Duration      string    `json:"duration"` // 5m, 15m, etc.
	Severity      string    `json:"severity"` // P1, P2, P3, P4
	DestinationID string    `json:"destinationId,omitempty"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type NotificationDestination struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Type      string            `json:"type"` // pagerduty, slack, email, webhook
	ClusterID string            `json:"clusterId"`
	Config    map[string]string `json:"config,omitempty"`
	Enabled   bool              `json:"enabled"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type ReliabilityStoreData struct {
	SLIs             map[string]SLIItem                 `json:"slis"`
	SLOs             map[string]SLOItem                 `json:"slos"`
	SLAs             map[string]SLAItem                 `json:"slas"`
	AlertPolicies    map[string]AlertPolicyItem         `json:"alertPolicies"`
	Destinations     map[string]NotificationDestination `json:"destinations"`
	PrometheusURLs   map[string]string                  `json:"prometheusUrls"`   // clusterID -> URL
	AlertmanagerURLs map[string]string                  `json:"alertmanagerUrls"` // clusterID -> URL
}

type ReliabilityStore struct {
	mu       sync.RWMutex
	filePath string
	data     ReliabilityStoreData
}

var (
	globalStore     *ReliabilityStore
	globalStoreOnce sync.Once
)

func GetReliabilityStore() *ReliabilityStore {
	globalStoreOnce.Do(func() {
		home, _ := os.UserHomeDir()
		storeDir := filepath.Join(home, ".garund")
		_ = os.MkdirAll(storeDir, 0755)
		filePath := filepath.Join(storeDir, "reliability_store.json")
		globalStore = NewReliabilityStore(filePath)
	})
	return globalStore
}

func NewReliabilityStore(filePath string) *ReliabilityStore {
	store := &ReliabilityStore{
		filePath: filePath,
		data: ReliabilityStoreData{
			SLIs:             make(map[string]SLIItem),
			SLOs:             make(map[string]SLOItem),
			SLAs:             make(map[string]SLAItem),
			AlertPolicies:    make(map[string]AlertPolicyItem),
			Destinations:     make(map[string]NotificationDestination),
			PrometheusURLs:   make(map[string]string),
			AlertmanagerURLs: make(map[string]string),
		},
	}

	_ = store.load()
	store.seedDefaultsIfEmpty()
	return store
}

func (s *ReliabilityStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.filePath == "" {
		return nil
	}

	bytes, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var data ReliabilityStoreData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}

	if data.SLIs == nil {
		data.SLIs = make(map[string]SLIItem)
	}
	if data.SLOs == nil {
		data.SLOs = make(map[string]SLOItem)
	}
	if data.SLAs == nil {
		data.SLAs = make(map[string]SLAItem)
	}
	if data.AlertPolicies == nil {
		data.AlertPolicies = make(map[string]AlertPolicyItem)
	}
	if data.Destinations == nil {
		data.Destinations = make(map[string]NotificationDestination)
	}
	if data.PrometheusURLs == nil {
		data.PrometheusURLs = make(map[string]string)
	}
	if data.AlertmanagerURLs == nil {
		data.AlertmanagerURLs = make(map[string]string)
	}

	s.data = data
	return nil
}

func (s *ReliabilityStore) saveLocked() error {
	if s.filePath == "" {
		return nil
	}

	bytes, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.filePath)
	_ = os.MkdirAll(dir, 0755)

	return os.WriteFile(s.filePath, bytes, 0644)
}

func (s *ReliabilityStore) seedDefaultsIfEmpty() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.data.SLIs) > 0 {
		return
	}

	now := time.Now()
	// Seed initial default SLIs/SLOs/SLAs for default cluster
	sli1 := SLIItem{
		ID:               "sli-default-availability",
		Name:             "Checkout Service Availability",
		Description:      "Percentage of successful checkout requests (2xx/3xx)",
		ClusterID:        "local-dev",
		Service:          "checkout",
		Namespace:        "default",
		Type:             "availability",
		GoodQuery:        `sum(rate(http_requests_total{service="checkout",status=~"2..|3.."}[5m]))`,
		TotalQuery:       `sum(rate(http_requests_total{service="checkout"}[5m]))`,
		Unit:             "%",
		EvaluationWindow: "5m",
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	sli2 := SLIItem{
		ID:               "sli-default-latency",
		Name:             "Checkout P95 Latency",
		Description:      "95th percentile HTTP response duration",
		ClusterID:        "local-dev",
		Service:          "checkout",
		Namespace:        "default",
		Type:             "latency",
		Query:            `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{service="checkout"}[5m])) by (le)) * 1000`,
		Unit:             "ms",
		EvaluationWindow: "5m",
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	sli3 := SLIItem{
		ID:               "sli-default-error-rate",
		Name:             "Checkout Error Rate",
		Description:      "Percentage of 5xx HTTP request errors",
		ClusterID:        "local-dev",
		Service:          "checkout",
		Namespace:        "default",
		Type:             "error_rate",
		GoodQuery:        `sum(rate(http_requests_total{service="checkout",status=~"5.."}[5m]))`,
		TotalQuery:       `sum(rate(http_requests_total{service="checkout"}[5m]))`,
		Unit:             "%",
		EvaluationWindow: "5m",
		Enabled:          true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	s.data.SLIs[sli1.ID] = sli1
	s.data.SLIs[sli2.ID] = sli2
	s.data.SLIs[sli3.ID] = sli3

	slo1 := SLOItem{
		ID:          "slo-default-availability",
		Name:        "Checkout 99.9% Availability",
		Description: "Target 99.9% uptime over a 30-day window",
		ClusterID:   "local-dev",
		Service:     "checkout",
		Namespace:   "default",
		SLIID:       sli1.ID,
		Target:      99.9,
		Window:      "30d",
		Owner:       "Checkout Platform Team",
		Team:        "Payments SRE",
		Enabled:     true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	s.data.SLOs[slo1.ID] = slo1

	availTarget := 99.9
	latencyTarget := 300.0
	sla1 := SLAItem{
		ID:                 "sla-default-checkout",
		Name:               "Customer Service SLA",
		Description:        "External customer availability SLA",
		ClusterID:          "local-dev",
		Service:            "checkout",
		Namespace:          "default",
		AvailabilityTarget: &availTarget,
		LatencyTargetMs:    &latencyTarget,
		Window:             "30d",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	s.data.SLAs[sla1.ID] = sla1

	pol1 := AlertPolicyItem{
		ID:            "policy-default-burn-rate",
		Name:          "Checkout SLO Fast Burn Rate Alert",
		ClusterID:     "local-dev",
		Service:       "checkout",
		Namespace:     "default",
		SLOID:         slo1.ID,
		SLIID:         sli1.ID,
		ConditionType: "burn_rate",
		Threshold:     2.0,
		Duration:      "15m",
		Severity:      "P1",
		Enabled:       true,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	s.data.AlertPolicies[pol1.ID] = pol1

	_ = s.saveLocked()
}

// SLI operations
func (s *ReliabilityStore) ListSLIs(clusterID string) []SLIItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SLIItem, 0)
	for _, item := range s.data.SLIs {
		if clusterID == "" || item.ClusterID == clusterID {
			result = append(result, item)
		}
	}
	return result
}

func (s *ReliabilityStore) SaveSLI(item SLIItem) SLIItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = fmt.Sprintf("sli-%d", time.Now().UnixNano())
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	s.data.SLIs[item.ID] = item
	_ = s.saveLocked()
	return item
}

func (s *ReliabilityStore) GetSLI(id string) (SLIItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data.SLIs[id]
	return item, ok
}

func (s *ReliabilityStore) DeleteSLI(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.SLIs[id]; ok {
		delete(s.data.SLIs, id)
		_ = s.saveLocked()
		return true
	}
	return false
}

// SLO operations
func (s *ReliabilityStore) ListSLOs(clusterID string) []SLOItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SLOItem, 0)
	for _, item := range s.data.SLOs {
		if clusterID == "" || item.ClusterID == clusterID {
			result = append(result, item)
		}
	}
	return result
}

func (s *ReliabilityStore) GetSLO(id string) (SLOItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data.SLOs[id]
	return item, ok
}

func (s *ReliabilityStore) SaveSLO(item SLOItem) SLOItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = fmt.Sprintf("slo-%d", time.Now().UnixNano())
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	s.data.SLOs[item.ID] = item
	_ = s.saveLocked()
	return item
}

func (s *ReliabilityStore) DeleteSLO(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.SLOs[id]; ok {
		delete(s.data.SLOs, id)
		_ = s.saveLocked()
		return true
	}
	return false
}

// SLA operations
func (s *ReliabilityStore) ListSLAs(clusterID string) []SLAItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]SLAItem, 0)
	for _, item := range s.data.SLAs {
		if clusterID == "" || item.ClusterID == clusterID {
			result = append(result, item)
		}
	}
	return result
}

func (s *ReliabilityStore) GetSLA(id string) (SLAItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data.SLAs[id]
	return item, ok
}

func (s *ReliabilityStore) SaveSLA(item SLAItem) SLAItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = fmt.Sprintf("sla-%d", time.Now().UnixNano())
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	s.data.SLAs[item.ID] = item
	_ = s.saveLocked()
	return item
}

func (s *ReliabilityStore) DeleteSLA(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.SLAs[id]; ok {
		delete(s.data.SLAs, id)
		_ = s.saveLocked()
		return true
	}
	return false
}

// AlertPolicy operations
func (s *ReliabilityStore) ListAlertPolicies(clusterID string) []AlertPolicyItem {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]AlertPolicyItem, 0)
	for _, item := range s.data.AlertPolicies {
		if clusterID == "" || item.ClusterID == clusterID {
			result = append(result, item)
		}
	}
	return result
}

func (s *ReliabilityStore) GetAlertPolicy(id string) (AlertPolicyItem, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data.AlertPolicies[id]
	return item, ok
}

func (s *ReliabilityStore) SaveAlertPolicy(item AlertPolicyItem) AlertPolicyItem {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = fmt.Sprintf("policy-%d", time.Now().UnixNano())
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	s.data.AlertPolicies[item.ID] = item
	_ = s.saveLocked()
	return item
}

func (s *ReliabilityStore) DeleteAlertPolicy(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.AlertPolicies[id]; ok {
		delete(s.data.AlertPolicies, id)
		_ = s.saveLocked()
		return true
	}
	return false
}

// Destination operations
func (s *ReliabilityStore) ListDestinations(clusterID string) []NotificationDestination {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]NotificationDestination, 0)
	for _, item := range s.data.Destinations {
		if clusterID == "" || item.ClusterID == clusterID {
			result = append(result, SanitizeDestination(item))
		}
	}
	return result
}

func (s *ReliabilityStore) GetDestination(id string) (NotificationDestination, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data.Destinations[id]
	return item, ok
}

func (s *ReliabilityStore) SaveDestination(item NotificationDestination) NotificationDestination {
	s.mu.Lock()
	defer s.mu.Unlock()

	if item.ID == "" {
		item.ID = fmt.Sprintf("dest-%d", time.Now().UnixNano())
		item.CreatedAt = time.Now()
	}
	item.UpdatedAt = time.Now()
	s.data.Destinations[item.ID] = item
	_ = s.saveLocked()
	return item
}

func (s *ReliabilityStore) DeleteDestination(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data.Destinations[id]; ok {
		delete(s.data.Destinations, id)
		_ = s.saveLocked()
		return true
	}
	return false
}

func SanitizeDestination(dest NotificationDestination) NotificationDestination {
	sanitized := dest
	if dest.Config != nil {
		sanitized.Config = make(map[string]string)
		for k, v := range dest.Config {
			lk := fmt.Sprintf("%s", k)
			lkLower := strings.ToLower(lk)
			if strings.Contains(lkLower, "secret") || strings.Contains(lkLower, "token") || strings.Contains(lkLower, "key") || strings.Contains(lkLower, "password") || strings.Contains(lkLower, "credential") || strings.Contains(lkLower, "auth") {
				if len(v) > 8 {
					sanitized.Config[k] = v[:4] + "****" + v[len(v)-4:]
				} else if len(v) > 0 {
					sanitized.Config[k] = "****"
				} else {
					sanitized.Config[k] = ""
				}
			} else {
				sanitized.Config[k] = v
			}
		}
	}
	return sanitized
}

// Config per cluster
func (s *ReliabilityStore) GetPrometheusURL(clusterID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if url, ok := s.data.PrometheusURLs[clusterID]; ok && url != "" {
		return url
	}
	return os.Getenv("PROMETHEUS_URL")
}

func (s *ReliabilityStore) SetPrometheusURL(clusterID, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.PrometheusURLs[clusterID] = url
	_ = s.saveLocked()
}

func (s *ReliabilityStore) GetAlertmanagerURL(clusterID string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if url, ok := s.data.AlertmanagerURLs[clusterID]; ok && url != "" {
		return url
	}
	return os.Getenv("ALERTMANAGER_URL")
}

func (s *ReliabilityStore) SetAlertmanagerURL(clusterID, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data.AlertmanagerURLs[clusterID] = url
	_ = s.saveLocked()
}

func ValidateDestinationConfig(dest NotificationDestination) error {
	if strings.TrimSpace(dest.Name) == "" {
		return fmt.Errorf("destination name is required")
	}

	switch strings.ToLower(dest.Type) {
	case "webhook":
		rawURL := dest.Config["url"]
		if rawURL == "" {
			rawURL = dest.Config["webhook_url"]
		}
		if rawURL == "" {
			return fmt.Errorf("webhook destination requires a 'url' or 'webhook_url' in configuration")
		}
		if err := ValidateSafeURL(rawURL); err != nil {
			return fmt.Errorf("invalid webhook URL: %w", err)
		}

	case "pagerduty":
		rk := dest.Config["routing_key"]
		if rk == "" {
			rk = dest.Config["integration_key"]
		}
		if rk == "" {
			rk = dest.Config["service_key"]
		}
		if rk == "" {
			return fmt.Errorf("pagerduty destination requires 'routing_key', 'integration_key', or 'service_key' in configuration")
		}

	case "alertmanager":
		baseURL := dest.Config["url"]
		if baseURL == "" {
			baseURL = dest.Config["base_url"]
		}
		if baseURL == "" {
			return fmt.Errorf("alertmanager destination requires a 'url' or 'base_url' in configuration")
		}
		if err := ValidateSafeURL(baseURL); err != nil {
			return fmt.Errorf("invalid alertmanager base URL: %w", err)
		}
	}

	return nil
}
