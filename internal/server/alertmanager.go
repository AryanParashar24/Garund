package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type AlertmanagerStatusType string

const (
	AlertmanagerStatusConnected    AlertmanagerStatusType = "CONNECTED"
	AlertmanagerStatusDisconnected AlertmanagerStatusType = "DISCONNECTED"
	AlertmanagerStatusAuthError    AlertmanagerStatusType = "AUTH_ERROR"
	AlertmanagerStatusUnknown      AlertmanagerStatusType = "UNKNOWN"
)

type AlertmanagerClient struct {
	BaseURL string
	Client  *http.Client
}

func NormalizeAlertmanagerURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "http://localhost:9093"
	}
	rawURL = strings.TrimRight(rawURL, "/")

	if strings.HasSuffix(rawURL, "/api/v2/alerts") {
		rawURL = strings.TrimSuffix(rawURL, "/api/v2/alerts")
	} else if strings.HasSuffix(rawURL, "/api/v2") {
		rawURL = strings.TrimSuffix(rawURL, "/api/v2")
	}

	return strings.TrimRight(rawURL, "/")
}

func NormalizeAlertmanagerAlertsURL(rawURL string) string {
	base := NormalizeAlertmanagerURL(rawURL)
	return base + "/api/v2/alerts"
}

func NewAlertmanagerClient(baseURL string) *AlertmanagerClient {
	normalized := NormalizeAlertmanagerURL(baseURL)

	return &AlertmanagerClient{
		BaseURL: normalized,
		Client:  NewSafeHTTPClient(10 * time.Second),
	}
}

func (a *AlertmanagerClient) Health(ctx context.Context) (AlertmanagerStatusType, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
	}

	reqURL := fmt.Sprintf("%s/-/healthy", a.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return AlertmanagerStatusDisconnected, err
	}

	resp, err := a.Client.Do(req)
	if err != nil {
		return AlertmanagerStatusDisconnected, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return AlertmanagerStatusConnected, nil
	}
	return AlertmanagerStatusDisconnected, fmt.Errorf("alertmanager returned status %s", resp.Status)
}

type AlertmanagerAlert struct {
	Status       string            `json:"status"` // firing, resolved
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

type AlertmanagerWebhookPayload struct {
	Version           string              `json:"version"`
	GroupKey          string              `json:"groupKey"`
	Status            string              `json:"status"`
	Receiver          string              `json:"receiver"`
	GroupLabels       map[string]string   `json:"groupLabels"`
	CommonLabels      map[string]string   `json:"commonLabels"`
	CommonAnnotations map[string]string   `json:"commonAnnotations"`
	Alerts            []AlertmanagerAlert `json:"alerts"`
}

type GarundAlert struct {
	Fingerprint  string            `json:"fingerprint"`
	Name         string            `json:"name"`
	ClusterID    string            `json:"clusterId"`
	Service      string            `json:"service"`
	Namespace    string            `json:"namespace"`
	SLOID        string            `json:"sloId,omitempty"`
	SLIID        string            `json:"sliId,omitempty"`
	Severity     string            `json:"severity"` // P1, P2, P3, P4
	Status       string            `json:"status"`   // firing, resolved, silenced
	Summary      string            `json:"summary"`
	Description  string            `json:"description"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt,omitempty"`
	UpdatedAt    time.Time         `json:"updatedAt"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	GeneratorURL string            `json:"generatorUrl,omitempty"`
}

type AlertStore struct {
	mu     sync.RWMutex
	alerts map[string]GarundAlert // Fingerprint -> Alert
}

var (
	globalAlertStore     *AlertStore
	globalAlertStoreOnce sync.Once
)

func GetAlertStore() *AlertStore {
	globalAlertStoreOnce.Do(func() {
		globalAlertStore = &AlertStore{
			alerts: make(map[string]GarundAlert),
		}
		globalAlertStore.seedSampleAlerts()
	})
	return globalAlertStore
}

func (s *AlertStore) seedSampleAlerts() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	a1 := GarundAlert{
		Fingerprint: "fp-checkout-burn-rate",
		Name:        "Checkout SLO Fast Burn Rate",
		ClusterID:   "local-dev",
		Service:     "checkout",
		Namespace:   "default",
		SLOID:       "slo-default-availability",
		SLIID:       "sli-default-availability",
		Severity:    "P1",
		Status:      "firing",
		Summary:     "Checkout Service error budget burn rate > 2x",
		Description: "High error budget consumption observed over 15m window.",
		StartsAt:    now.Add(-14 * time.Minute),
		UpdatedAt:   now,
		Labels: map[string]string{
			"alertname": "GarundCheckoutSLOBurn",
			"severity":  "critical",
			"service":   "checkout",
			"namespace": "default",
		},
		Annotations: map[string]string{
			"summary":     "Checkout SLO burn rate is high",
			"description": "Error budget remaining: 73%",
		},
	}

	a2 := GarundAlert{
		Fingerprint: "fp-checkout-latency",
		Name:        "Checkout High P95 Latency",
		ClusterID:   "local-dev",
		Service:     "checkout",
		Namespace:   "default",
		SLIID:       "sli-default-latency",
		Severity:    "P2",
		Status:      "firing",
		Summary:     "Checkout P95 Latency > 300ms",
		Description: "P95 latency spike detected: 421ms",
		StartsAt:    now.Add(-28 * time.Minute),
		UpdatedAt:   now,
		Labels: map[string]string{
			"alertname": "GarundCheckoutLatencySpike",
			"severity":  "warning",
			"service":   "checkout",
			"namespace": "default",
		},
		Annotations: map[string]string{
			"summary": "Checkout P95 response latency exceeded target",
		},
	}

	s.alerts[a1.Fingerprint] = a1
	s.alerts[a2.Fingerprint] = a2
}

func (s *AlertStore) IngestWebhook(clusterID string, payload AlertmanagerWebhookPayload) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	now := time.Now()

	for _, alert := range payload.Alerts {
		fp := alert.Fingerprint
		if fp == "" {
			// Compute fallback fingerprint from labels
			h := sha256.New()
			for k, v := range alert.Labels {
				h.Write([]byte(k + "=" + v + ";"))
			}
			fp = hex.EncodeToString(h.Sum(nil))[:16]
		}

		service := alert.Labels["service"]
		if service == "" {
			service = alert.Labels["job"]
		}
		namespace := alert.Labels["namespace"]
		if namespace == "" {
			namespace = alert.Labels["k8s_namespace_name"]
		}
		if namespace == "" {
			namespace = "default"
		}

		severity := "P2"
		if alert.Labels["severity"] == "critical" || alert.Labels["severity"] == "P1" {
			severity = "P1"
		} else if alert.Labels["severity"] == "info" || alert.Labels["severity"] == "P3" {
			severity = "P3"
		}

		name := alert.Labels["alertname"]
		if name == "" {
			name = "Alertmanager Incident"
		}

		summary := alert.Annotations["summary"]
		if summary == "" {
			summary = fmt.Sprintf("Alert %s on service %s", name, service)
		}
		desc := alert.Annotations["description"]

		status := alert.Status
		if status == "" {
			status = payload.Status
		}

		existing, exists := s.alerts[fp]
		if exists {
			existing.Status = status
			existing.UpdatedAt = now
			if status == "resolved" && !alert.EndsAt.IsZero() {
				existing.EndsAt = alert.EndsAt
			}
			s.alerts[fp] = existing
		} else {
			garundAlert := GarundAlert{
				Fingerprint:  fp,
				Name:         name,
				ClusterID:    clusterID,
				Service:      service,
				Namespace:    namespace,
				SLOID:        alert.Labels["slo_id"],
				SLIID:        alert.Labels["sli_id"],
				Severity:     severity,
				Status:       status,
				Summary:      summary,
				Description:  desc,
				StartsAt:     alert.StartsAt,
				EndsAt:       alert.EndsAt,
				UpdatedAt:    now,
				Labels:       alert.Labels,
				Annotations:  alert.Annotations,
				GeneratorURL: alert.GeneratorURL,
			}
			if garundAlert.StartsAt.IsZero() {
				garundAlert.StartsAt = now
			}
			s.alerts[fp] = garundAlert
		}
		count++
	}

	return count
}

func (s *AlertStore) ListAlerts(clusterID string, status string) []GarundAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []GarundAlert
	for _, alert := range s.alerts {
		if clusterID != "" && alert.ClusterID != "" && alert.ClusterID != clusterID {
			continue
		}
		if status != "" && !strings.EqualFold(alert.Status, status) {
			continue
		}
		result = append(result, alert)
	}
	return result
}

// GeneratePrometheusAlertingRule formats a Prometheus rule group YAML string for an SLO alert policy.
func GeneratePrometheusAlertingRule(slo SLOItem, policy AlertPolicyItem, promql string) string {
	alertName := fmt.Sprintf("Garund_%s_%s", strings.Title(slo.Service), strings.ReplaceAll(policy.Name, " ", "_"))
	severity := "critical"
	if policy.Severity == "P2" || policy.Severity == "P3" {
		severity = "warning"
	}

	return fmt.Sprintf(`groups:
  - name: garund-slo-alerts
    rules:
      - alert: %s
        expr: %s
        for: %s
        labels:
          severity: %s
          garund_cluster: "%s"
          garund_service: "%s"
          garund_namespace: "%s"
          slo_id: "%s"
          sli_id: "%s"
        annotations:
          summary: "%s breaching target (%f%%)"
          description: "Burn rate threshold %f exceeded over %s window."
`, alertName, promql, policy.Duration, severity, slo.ClusterID, slo.Service, slo.Namespace, slo.ID, slo.SLIID, slo.Name, slo.Target, policy.Threshold, policy.Duration)
}
