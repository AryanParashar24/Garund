package server

import (
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestCalculateErrorBudgetRemaining(t *testing.T) {
	// 99.9% target, current 100% -> 100% budget remaining
	rem1 := CalculateErrorBudgetRemaining(99.9, 100.0)
	if rem1 != 100 {
		t.Errorf("expected 100%% remaining for perfect uptime, got %f", rem1)
	}

	// 99.9% target, current 99.95% -> 50% budget remaining (0.05% error out of 0.1% allowed)
	rem2 := CalculateErrorBudgetRemaining(99.9, 99.95)
	if math.Abs(rem2-50.0) > 0.01 {
		t.Errorf("expected 50%% remaining, got %f", rem2)
	}

	// 99.9% target, current 99.9% -> 0% budget remaining (exact budget used)
	rem3 := CalculateErrorBudgetRemaining(99.9, 99.9)
	if math.Abs(rem3-0.0) > 0.01 {
		t.Errorf("expected 0%% remaining when target exactly met, got %f", rem3)
	}

	// 99.9% target, current 99.85% -> 0% remaining (budget exhausted)
	rem4 := CalculateErrorBudgetRemaining(99.9, 99.85)
	if rem4 != 0 {
		t.Errorf("expected 0%% remaining when exhausted, got %f", rem4)
	}
}

func TestCalculateBurnRate(t *testing.T) {
	// SLO 99.9% -> allowed 0.1%. Observed error 0.4% -> burn rate 4.0x
	br := CalculateBurnRate(99.9, 99.6)
	if math.Abs(br-4.0) > 0.01 {
		t.Errorf("expected burn rate 4.0, got %f", br)
	}

	// Observed error 0.1% -> burn rate 1.0x
	br1 := CalculateBurnRate(99.9, 99.9)
	if math.Abs(br1-1.0) > 0.01 {
		t.Errorf("expected burn rate 1.0, got %f", br1)
	}
}

func TestEvaluateSLO_UnavailableTelemetry(t *testing.T) {
	sloItem := SLOItem{
		ID:        "slo-1",
		Name:      "Checkout Availability",
		Service:   "checkout",
		Namespace: "default",
		Target:    99.9,
		Window:    "30d",
	}

	// Missing telemetry (current = nil) must produce status "unavailable"
	evaluated := EvaluateSLO(sloItem, nil, "Checkout SLI", "availability")
	if evaluated.Status != "unavailable" {
		t.Errorf("expected status 'unavailable', got '%s'", evaluated.Status)
	}
	if evaluated.Current != nil {
		t.Errorf("expected current value to be nil for missing telemetry, got %v", evaluated.Current)
	}
}

func TestGeneratePromQL(t *testing.T) {
	// Availability PromQL
	inputAvail := PromQLInput{
		Type:         "availability",
		Metric:       "http_requests_total",
		Service:      "checkout",
		Namespace:    "default",
		Window:       "5m",
		GoodStatuses: []string{"2..", "3.."},
	}
	outAvail := GeneratePromQL(inputAvail)
	expectedQuery := `sum(rate(http_requests_total{service="checkout",namespace="default",status=~"2..|3.."}[5m])) / sum(rate(http_requests_total{service="checkout",namespace="default"}[5m])) * 100`
	if outAvail.Query != expectedQuery {
		t.Errorf("expected query:\n%s\ngot:\n%s", expectedQuery, outAvail.Query)
	}

	// Latency Quantile PromQL
	inputLat := PromQLInput{
		Type:       "latency",
		Metric:     "http_request_duration_seconds",
		Service:    "checkout",
		Percentile: "p95",
		Window:     "5m",
	}
	outLat := GeneratePromQL(inputLat)
	expectedLat := `histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket{service="checkout"}[5m])) by (le)) * 1000`
	if outLat.Query != expectedLat {
		t.Errorf("expected query:\n%s\ngot:\n%s", expectedLat, outLat.Query)
	}
}

func TestReliabilityStoreCRUD(t *testing.T) {
	store := NewReliabilityStore("") // memory-only

	sli := SLIItem{
		Name:             "Test SLI",
		ClusterID:        "cluster-test",
		Service:          "auth",
		Namespace:        "default",
		Type:             "availability",
		EvaluationWindow: "5m",
		Enabled:          true,
	}

	savedSLI := store.SaveSLI(sli)
	if savedSLI.ID == "" {
		t.Fatalf("expected saved SLI to have generated ID")
	}

	list := store.ListSLIs("cluster-test")
	if len(list) != 1 {
		t.Fatalf("expected 1 SLI for cluster-test, got %d", len(list))
	}

	deleted := store.DeleteSLI(savedSLI.ID)
	if !deleted {
		t.Errorf("expected SLI deletion to return true")
	}

	listAfter := store.ListSLIs("cluster-test")
	if len(listAfter) != 0 {
		t.Errorf("expected 0 SLIs after deletion, got %d", len(listAfter))
	}
}

func TestAlertStoreWebhookIngestion(t *testing.T) {
	store := &AlertStore{
		alerts: make(map[string]GarundAlert),
	}

	payload := AlertmanagerWebhookPayload{
		Status: "firing",
		Alerts: []AlertmanagerAlert{
			{
				Fingerprint: "fp-test-1",
				Status:      "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
					"severity":  "critical",
					"service":   "payment",
					"namespace": "production",
				},
				Annotations: map[string]string{
					"summary": "Payment failure spike",
				},
				StartsAt: time.Now(),
			},
		},
	}

	count := store.IngestWebhook("cluster-test", payload)
	if count != 1 {
		t.Fatalf("expected 1 alert ingested, got %d", count)
	}

	alerts := store.ListAlerts("cluster-test", "firing")
	if len(alerts) != 1 {
		t.Fatalf("expected 1 active firing alert, got %d", len(alerts))
	}
	if alerts[0].Severity != "P1" {
		t.Errorf("expected critical severity to map to P1, got %s", alerts[0].Severity)
	}
}

func TestEvaluateSLA_SafetyMargin(t *testing.T) {
	availTarget := 99.0
	slaItem := SLAItem{
		ID:                 "sla-1",
		Name:               "Customer SLA",
		Service:            "checkout",
		Namespace:          "default",
		AvailabilityTarget: &availTarget,
		Window:             "30d",
	}

	// SLO target is 99.9%, SLA target is 99.0% -> safety margin is +0.9%
	currAvail := 99.95
	evaluated := EvaluateSLA(slaItem, &currAvail, nil, 99.9)

	if evaluated.Status != "compliant" {
		t.Errorf("expected status 'compliant', got '%s'", evaluated.Status)
	}
	if evaluated.SafetyMargin == nil || *evaluated.SafetyMargin != 0.9 {
		t.Errorf("expected safety margin 0.9, got %v", evaluated.SafetyMargin)
	}
}

func TestDestinationCRUD(t *testing.T) {
	store := NewReliabilityStore("") // memory-only

	dest := NotificationDestination{
		Name:      "PagerDuty OnCall",
		Type:      "pagerduty",
		ClusterID: "cluster-prod",
		Enabled:   true,
		Config: map[string]string{
			"routing_key": "pd-secret-token",
		},
	}

	saved := store.SaveDestination(dest)
	if saved.ID == "" {
		t.Fatalf("expected destination ID to be generated")
	}

	list := store.ListDestinations("cluster-prod")
	if len(list) != 1 {
		t.Fatalf("expected 1 destination for cluster-prod, got %d", len(list))
	}

	deleted := store.DeleteDestination(saved.ID)
	if !deleted {
		t.Errorf("expected DeleteDestination to return true")
	}
}

func TestReliabilityHTTPOverviewEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterReliabilityRoutes(router)

	endpoints := []string{
		"/api/clusters/local-dev/reliability/overview",
		"/api/clusters/local-dev/slis",
		"/api/clusters/local-dev/slos",
		"/api/clusters/local-dev/slas",
		"/api/clusters/local-dev/alerts/active",
		"/api/clusters/local-dev/prometheus/status",
		"/api/reliability/overview",
	}

	for _, ep := range endpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", ep, nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK && w.Code != http.StatusTemporaryRedirect {
			t.Errorf("expected endpoint %s to return 200 or 307 redirect, got HTTP %d: %s", ep, w.Code, w.Body.String())
		}
	}
}


