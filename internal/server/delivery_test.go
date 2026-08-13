package server

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSSRFValidation(t *testing.T) {
	AllowLoopbackForTesting = false
	tests := []struct {
		url     string
		wantErr bool
		name    string
	}{
		{"http://127.0.0.1:8080/webhook", true, "loopback IPv4"},
		{"http://localhost:8080/webhook", true, "localhost string"},
		{"http://10.0.0.1/webhook", true, "RFC1918 10.x"},
		{"http://172.16.0.1/webhook", true, "RFC1918 172.16.x"},
		{"http://192.168.1.1/webhook", true, "RFC1918 192.168.x"},
		{"http://169.254.169.254/latest/meta-data", true, "link-local AWS metadata"},
		{"http://[::1]:8080/webhook", true, "IPv6 loopback"},
		{"http://[fe80::1]/webhook", true, "IPv6 link-local"},
		{"http://user:password@example.com/webhook", true, "embedded credentials"},
		{"ftp://example.com/webhook", true, "unsupported scheme ftp"},
		{"https://8.8.8.8/webhook", false, "public IP IPv4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSafeURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSafeURL(%s) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestAlertmanagerURLNormalization(t *testing.T) {
	tests := []struct {
		input          string
		expectedBase   string
		expectedAlerts string
	}{
		{"http://alertmanager:9093", "http://alertmanager:9093", "http://alertmanager:9093/api/v2/alerts"},
		{"http://alertmanager:9093/", "http://alertmanager:9093", "http://alertmanager:9093/api/v2/alerts"},
		{"http://alertmanager:9093/api/v2", "http://alertmanager:9093", "http://alertmanager:9093/api/v2/alerts"},
		{"http://alertmanager:9093/api/v2/", "http://alertmanager:9093", "http://alertmanager:9093/api/v2/alerts"},
		{"http://alertmanager:9093/api/v2/alerts", "http://alertmanager:9093", "http://alertmanager:9093/api/v2/alerts"},
		{"http://alertmanager:9093/api/v2/alerts/", "http://alertmanager:9093", "http://alertmanager:9093/api/v2/alerts"},
		{"", "http://localhost:9093", "http://localhost:9093/api/v2/alerts"},
	}

	for _, tt := range tests {
		base := NormalizeAlertmanagerURL(tt.input)
		if base != tt.expectedBase {
			t.Errorf("NormalizeAlertmanagerURL(%s) = %s; want %s", tt.input, base, tt.expectedBase)
		}
		alerts := NormalizeAlertmanagerAlertsURL(tt.input)
		if alerts != tt.expectedAlerts {
			t.Errorf("NormalizeAlertmanagerAlertsURL(%s) = %s; want %s", tt.input, alerts, tt.expectedAlerts)
		}
	}
}

func TestE2EWebhookDelivery(t *testing.T) {
	AllowLoopbackForTesting = true
	defer func() { AllowLoopbackForTesting = false }()
	// Setup test server on public loopback listener bypassing external DNS
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") == "Bearer secret-token" {
			w.Header().Set("X-Auth-Received", "true")
		}

		switch r.URL.Path {
		case "/200":
			w.WriteHeader(http.StatusOK)
		case "/201":
			w.WriteHeader(http.StatusCreated)
		case "/204":
			w.WriteHeader(http.StatusNoContent)
		case "/400":
			w.WriteHeader(http.StatusBadRequest)
		case "/500":
			w.WriteHeader(http.StatusInternalServerError)
		case "/redirect-valid":
			http.Redirect(w, r, "/200", http.StatusFound)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	dest := NotificationDestination{
		ID:        "dest-webhook-test",
		Name:      "Test Webhook",
		Type:      "webhook",
		ClusterID: "cluster-test",
		Enabled:   true,
		Config: map[string]string{
			"url":         ts.URL + "/200",
			"auth_header": "Bearer secret-token",
		},
	}

	alert := GarundAlert{
		Fingerprint: "fp-12345",
		Name:        "High Error Rate",
		Severity:    "P1",
		Status:      "firing",
		StartsAt:    time.Now(),
	}

	// Test delivery via deliverAlertToDestination with httptest server
	delivered, status, _ := deliverAlertToDestination(context.Background(), dest, alert, "")
	if !delivered || status != "delivered" {
		t.Fatalf("expected webhook delivery success, got delivered: %v, status: %s", delivered, status)
	}

	client := NewSafeHTTPClient(2 * time.Second)

	// 201 Created
	req, _ := http.NewRequest("POST", ts.URL+"/201", strings.NewReader("{}"))
	resp, err := client.Do(req)
	if err == nil && resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201 Created, got %d", resp.StatusCode)
	}
	if resp != nil {
		resp.Body.Close()
	}

	// 400 Bad Request -> failure
	req, _ = http.NewRequest("POST", ts.URL+"/400", strings.NewReader("{}"))
	resp, err = client.Do(req)
	if err == nil && resp.StatusCode == http.StatusOK {
		t.Errorf("expected 400 failure status, got 200")
	}
	if resp != nil {
		resp.Body.Close()
	}

	// Verify secret redaction function
	errMsg := "Failed request to http://user:password@example.com/api with bearer token Bearer secret-123"
	sanitized := SanitizeErrorMessage(errMsg)
	if strings.Contains(sanitized, "password") || strings.Contains(sanitized, "secret-123") {
		t.Errorf("SanitizeErrorMessage failed to redact secrets: %s", sanitized)
	}
}

func TestE2EAlertmanagerDelivery(t *testing.T) {
	AllowLoopbackForTesting = true
	defer func() { AllowLoopbackForTesting = false }()
	var receivedPayload []map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/api/v2/alerts" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	dest := NotificationDestination{
		ID:        "dest-am-test",
		Name:      "Test Alertmanager",
		Type:      "alertmanager",
		ClusterID: "cluster-test",
		Enabled:   true,
		Config: map[string]string{
			"url": ts.URL,
		},
	}

	alert := GarundAlert{
		Fingerprint: "am-fp-001",
		Name:        "High Latency",
		Severity:    "P2",
		Status:      "firing",
		StartsAt:    time.Now(),
		Labels: map[string]string{
			"alertname": "High Latency",
			"severity":  "P2",
		},
		Annotations: map[string]string{
			"summary": "P99 latency exceeding 500ms",
		},
	}

	delivered, status, _ := deliverAlertToDestination(context.Background(), dest, alert, ts.URL)
	if !delivered || status != "delivered" {
		t.Fatalf("expected Alertmanager delivery success via httptest, got delivered: %v, status: %s", delivered, status)
	}

	if len(receivedPayload) != 1 {
		t.Fatalf("expected 1 alert in Alertmanager payload, got %d", len(receivedPayload))
	}
}

func TestE2EPagerDutyDelivery(t *testing.T) {
	var receivedPayload map[string]interface{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusAccepted) // 202 Accepted
	}))
	defer ts.Close()

	oldEP := PagerDutyEndpoint
	PagerDutyEndpoint = ts.URL
	defer func() { PagerDutyEndpoint = oldEP }()

	dest := NotificationDestination{
		ID:        "dest-pd-test",
		Name:      "Test PagerDuty",
		Type:      "pagerduty",
		ClusterID: "cluster-test",
		Enabled:   true,
		Config: map[string]string{
			"routing_key": "pd-routing-key-secret-123",
		},
	}

	alert := GarundAlert{
		Fingerprint: "pd-fp-001",
		Name:        "Database Unavailable",
		Severity:    "P1",
		Service:     "auth-service",
		Namespace:   "prod",
		Status:      "firing",
		Summary:     "Auth DB cluster down",
		StartsAt:    time.Now(),
	}

	delivered, status, msg := deliverAlertToDestination(context.Background(), dest, alert, "")
	if !delivered || status != "delivered" {
		t.Fatalf("expected PagerDuty delivery success, got delivered: %v, status: %s, msg: %s", delivered, status, msg)
	}

	if receivedPayload["routing_key"] != "pd-routing-key-secret-123" {
		t.Errorf("expected routing_key in payload, got %v", receivedPayload["routing_key"])
	}

	// Secret leakage test: ensure routing key is never in the return message
	if strings.Contains(msg, "pd-routing-key-secret-123") {
		t.Errorf("PagerDuty routing key leaked in output message: %s", msg)
	}
}
