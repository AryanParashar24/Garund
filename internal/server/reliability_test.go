package server

import (
	"bytes"
	"encoding/json"
	"fmt"
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

func TestAlertPolicyCRUDAndClusterIsolation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterReliabilityRoutes(router)

	// 1. Empty policy list -> 200 []
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/clusters/cluster-a/alerts/policies", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for empty policy list, got %d", w.Code)
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"policies":[]`)) {
		t.Errorf("expected empty policies array [], got %s", w.Body.String())
	}

	// 2. Nonexistent policy -> 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/cluster-a/alerts/policies/nonexistent-id", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nonexistent policy GET, got %d", w.Code)
	}

	// 3. Create policy for cluster-a
	newPolicyJSON := `{
		"name": "High Error Rate Policy",
		"condition": "burn_rate_exceeded",
		"threshold": 2.5,
		"severity": "critical",
		"enabled": true
	}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/clusters/cluster-a/alerts/policies", bytes.NewBufferString(newPolicyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created when creating policy, got %d: %s", w.Code, w.Body.String())
	}

	var created AlertPolicyItem
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal created policy: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected created policy to have an ID")
	}
	if created.ClusterID != "cluster-a" {
		t.Errorf("expected clusterId cluster-a, got %s", created.ClusterID)
	}

	// 4. GET created policy on cluster-a
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/cluster-a/alerts/policies/"+created.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for GET created policy, got %d", w.Code)
	}

	// 5. GET created policy on cluster-b (wrong clusterId isolation) -> 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/cluster-b/alerts/policies/"+created.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 when querying policy with wrong clusterId, got %d", w.Code)
	}

	// 6. List policies on cluster-a vs cluster-b
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/cluster-a/alerts/policies", nil)
	router.ServeHTTP(w, req)
	if !bytes.Contains(w.Body.Bytes(), []byte(created.ID)) {
		t.Errorf("expected cluster-a policy list to contain %s", created.ID)
	}

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/cluster-b/alerts/policies", nil)
	router.ServeHTTP(w, req)
	if bytes.Contains(w.Body.Bytes(), []byte(created.ID)) {
		t.Errorf("expected cluster-b policy list NOT to contain %s", created.ID)
	}

	// 7. Update policy on cluster-a (PUT)
	updateJSON := `{
		"name": "High Error Rate Policy Updated",
		"condition": "burn_rate_exceeded",
		"threshold": 5.0,
		"severity": "critical",
		"enabled": true
	}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/clusters/cluster-a/alerts/policies/"+created.ID, bytes.NewBufferString(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for PUT policy update, got %d: %s", w.Code, w.Body.String())
	}

	var updated AlertPolicyItem
	_ = json.Unmarshal(w.Body.Bytes(), &updated)
	if updated.Threshold != 5.0 || updated.Name != "High Error Rate Policy Updated" {
		t.Errorf("policy update mismatch, got threshold %f, name %s", updated.Threshold, updated.Name)
	}

	// 8. Update policy on cluster-b (wrong clusterId isolation) -> 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/clusters/cluster-b/alerts/policies/"+created.ID, bytes.NewBufferString(updateJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for PUT policy update with wrong clusterId, got %d", w.Code)
	}

	// 9. Delete policy on cluster-b (wrong clusterId isolation) -> 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/clusters/cluster-b/alerts/policies/"+created.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for DELETE policy with wrong clusterId, got %d", w.Code)
	}

	// 10. Delete policy on cluster-a -> 200
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/clusters/cluster-a/alerts/policies/"+created.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK for DELETE policy, got %d", w.Code)
	}

	// 11. Verify policy is deleted -> 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/cluster-a/alerts/policies/"+created.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for deleted policy GET, got %d", w.Code)
	}
}

func TestAlertPoliciesCanonicalRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterReliabilityRoutes(router)

	// 1. Empty policy list -> HTTP 200 []
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/clusters/test-cluster-x/alert-policies", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /api/clusters/test-cluster-x/alert-policies, got %d", w.Code)
	}
	if w.Body.String() != "[]" {
		t.Fatalf("expected empty array '[]', got '%s'", w.Body.String())
	}

	// 2. Create policy
	newPolicyJSON := `{
		"name": "Canonical Policy Test",
		"conditionType": "burn_rate",
		"threshold": 2.0,
		"severity": "P1",
		"enabled": true
	}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/clusters/test-cluster-x/alert-policies", bytes.NewBufferString(newPolicyJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var created AlertPolicyItem
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to unmarshal created policy: %v", err)
	}

	// 3. GET request -> returns array with policy
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/test-cluster-x/alert-policies", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", w.Code)
	}
	var list []AlertPolicyItem
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("failed to unmarshal array: %v", err)
	}
	if len(list) != 1 || list[0].ID != created.ID {
		t.Fatalf("expected 1 policy in array, got %d", len(list))
	}

	// 4. Cluster isolation: test-cluster-y must return []
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/test-cluster-y/alert-policies", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "[]" {
		t.Fatalf("expected [] for isolated cluster y, got %d: %s", w.Code, w.Body.String())
	}

	// 5. Delete policy
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("DELETE", "/api/clusters/test-cluster-x/alert-policies/"+created.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on delete, got %d", w.Code)
	}

	// 6. GET request -> returns []
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/test-cluster-x/alert-policies", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "[]" {
		t.Fatalf("expected [] after deletion, got %d: %s", w.Code, w.Body.String())
	}
}

func TestSLI_SLO_SLA_CRUD_Completeness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterReliabilityRoutes(router)

	clusterA := "test-cluster-crud-a"
	clusterB := "test-cluster-crud-b"

	// 1. SLI CRUD
	sliJSON := `{
		"name": "Test SLI",
		"type": "availability",
		"service": "cart",
		"namespace": "prod",
		"evaluationWindow": "5m"
	}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/clusters/"+clusterA+"/slis", bytes.NewBufferString(sliJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for SLI POST, got %d: %s", w.Code, w.Body.String())
	}
	var createdSLI SLIItem
	_ = json.Unmarshal(w.Body.Bytes(), &createdSLI)

	// GET SLI by ID
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/"+clusterA+"/slis/"+createdSLI.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for SLI GET, got %d", w.Code)
	}

	// Cluster isolation: GET SLI on clusterB -> 404
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/clusters/"+clusterB+"/slis/"+createdSLI.ID, nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for cross-cluster SLI GET, got %d", w.Code)
	}

	// PUT SLI update
	updateSLIJSON := `{
		"name": "Test SLI Updated",
		"type": "availability",
		"service": "cart",
		"namespace": "prod",
		"evaluationWindow": "15m"
	}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/clusters/"+clusterA+"/slis/"+createdSLI.ID, bytes.NewBufferString(updateSLIJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for SLI PUT, got %d", w.Code)
	}

	// 2. SLO CRUD
	sloJSON := fmt.Sprintf(`{
		"name": "Test SLO",
		"service": "cart",
		"namespace": "prod",
		"sliId": "%s",
		"target": 99.9,
		"window": "30d"
	}`, createdSLI.ID)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/clusters/"+clusterA+"/slos", bytes.NewBufferString(sloJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for SLO POST, got %d: %s", w.Code, w.Body.String())
	}
	var createdSLO SLOItem
	_ = json.Unmarshal(w.Body.Bytes(), &createdSLO)

	// PUT SLO update
	updateSLOJSON := fmt.Sprintf(`{
		"name": "Test SLO Updated",
		"service": "cart",
		"namespace": "prod",
		"sliId": "%s",
		"target": 99.5,
		"window": "30d"
	}`, createdSLI.ID)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/clusters/"+clusterA+"/slos/"+createdSLO.ID, bytes.NewBufferString(updateSLOJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for SLO PUT, got %d", w.Code)
	}

	// 3. SLA CRUD
	target := 99.5
	slaItem := SLAItem{
		Name:               "Test SLA",
		Service:            "cart",
		Namespace:          "prod",
		AvailabilityTarget: &target,
		Window:             "30d",
	}
	slaBytes, _ := json.Marshal(slaItem)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/clusters/"+clusterA+"/slas", bytes.NewBuffer(slaBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for SLA POST, got %d: %s", w.Code, w.Body.String())
	}
	var createdSLA SLAItem
	_ = json.Unmarshal(w.Body.Bytes(), &createdSLA)

	// PUT SLA update
	newTarget := 99.0
	createdSLA.AvailabilityTarget = &newTarget
	createdSLA.Name = "Test SLA Updated"
	slaBytes, _ = json.Marshal(createdSLA)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/clusters/"+clusterA+"/slas/"+createdSLA.ID, bytes.NewBuffer(slaBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for SLA PUT, got %d", w.Code)
	}
}

func TestTestAlertDeliveryTruthfulness(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterReliabilityRoutes(router)

	clusterID := fmt.Sprintf("cluster-deliv-%d", time.Now().UnixNano())

	// Create policy
	polJSON := `{
		"name": "Delivery Test Policy",
		"conditionType": "burn_rate",
		"threshold": 2.0,
		"severity": "P1",
		"enabled": true
	}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/clusters/"+clusterID+"/alert-policies", bytes.NewBufferString(polJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	var pol AlertPolicyItem
	_ = json.Unmarshal(w.Body.Bytes(), &pol)

	// Case 1: No destination configured -> delivered: false, status: not_configured
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/clusters/"+clusterID+"/alert-policies/"+pol.ID+"/test", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for test alert, got %d", w.Code)
	}
	var res1 struct {
		Delivered bool   `json:"delivered"`
		Status    string `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res1)
	if res1.Delivered || res1.Status != "not_configured" {
		t.Fatalf("expected delivered: false, status: not_configured, got delivered: %v, status: %s", res1.Delivered, res1.Status)
	}

	// Case 2: Create destination of unsupported type -> status: unsupported
	destJSON := `{
		"name": "Unsupported Dest",
		"type": "custom_email",
		"enabled": true
	}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/clusters/"+clusterID+"/destinations", bytes.NewBufferString(destJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("POST", "/api/clusters/"+clusterID+"/alert-policies/"+pol.ID+"/test", nil)
	router.ServeHTTP(w, req)
	var res2 struct {
		Delivered bool   `json:"delivered"`
		Status    string `json:"status"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &res2)
	if res2.Delivered || res2.Status != "unsupported" {
		t.Fatalf("expected delivered: false, status: unsupported, got delivered: %v, status: %s", res2.Delivered, res2.Status)
	}
}

func floatPtr(v float64) *float64 { return &v }

func TestSLIStatusEvaluationSemantics(t *testing.T) {
	// Availability (higher is better)
	valAvailHigh := 99.95
	valAvailLow := 98.0
	if status := CalculateSLIStatus(&valAvailHigh, floatPtr(99.9), "availability"); status != "healthy" {
		t.Errorf("expected healthy for availability 99.95 vs target 99.9, got %s", status)
	}
	if status := CalculateSLIStatus(&valAvailLow, floatPtr(99.9), "availability"); status != "critical" {
		t.Errorf("expected critical for availability 98.0 vs target 99.9, got %s", status)
	}

	// Error Rate (lower is better)
	valErrLow := 0.05
	valErrHigh := 2.5
	if status := CalculateSLIStatus(&valErrLow, floatPtr(0.1), "error_rate"); status != "healthy" {
		t.Errorf("expected healthy for error rate 0.05 vs target 0.1, got %s", status)
	}
	if status := CalculateSLIStatus(&valErrHigh, floatPtr(0.1), "error_rate"); status != "critical" {
		t.Errorf("expected critical for error rate 2.5 vs target 0.1, got %s", status)
	}

	// Latency (lower is better)
	valLatLow := 150.0
	valLatHigh := 500.0
	if status := CalculateSLIStatus(&valLatLow, floatPtr(300.0), "latency"); status != "healthy" {
		t.Errorf("expected healthy for latency 150ms vs target 300ms, got %s", status)
	}
	if status := CalculateSLIStatus(&valLatHigh, floatPtr(300.0), "latency"); status != "critical" {
		t.Errorf("expected critical for latency 500ms vs target 300ms, got %s", status)
	}
}

func TestConfiguredTargetPreservation(t *testing.T) {
	val := 99.94

	// When evaluated against default 99.9% target, 99.94% is HEALTHY
	status999 := CalculateSLIStatus(&val, floatPtr(99.9), "availability")
	if status999 != "healthy" {
		t.Errorf("expected 99.94%% vs 99.9%% target to be healthy, got %s", status999)
	}

	// When evaluated against configured 99.95% target, 99.94% is NOT healthy (it's warning)
	status9995 := CalculateSLIStatus(&val, floatPtr(99.95), "availability")
	if status9995 == "healthy" {
		t.Errorf("expected 99.94%% vs 99.95%% target to NOT be healthy, got %s", status9995)
	}

	// Test resolveSLITarget honors SLI target and SLO target
	store := NewReliabilityStore(t.TempDir() + "/test-store.json")
	sli := store.SaveSLI(SLIItem{
		Name:      "Custom Target SLI",
		ClusterID: "c1",
		Service:   "payment",
		Namespace: "prod",
		Type:      "availability",
		Target:    99.99,
	})

	target := resolveSLITarget(sli, store, "c1")
	if target == nil || *target != 99.99 {
		t.Errorf("expected resolveSLITarget to return SLI explicit target 99.99, got %v", target)
	}

	// Test resolveSLITarget falls back to associated SLO target if SLI target is 0
	sli2 := store.SaveSLI(SLIItem{
		Name:      "SLO Target SLI",
		ClusterID: "c1",
		Service:   "cart",
		Namespace: "prod",
		Type:      "availability",
		Target:    0,
	})
	_ = store.SaveSLO(SLOItem{
		Name:      "Cart SLO",
		ClusterID: "c1",
		Service:   "cart",
		Namespace: "prod",
		SLIID:     sli2.ID,
		Target:    99.95,
	})

	target2 := resolveSLITarget(sli2, store, "c1")
	if target2 == nil || *target2 != 99.95 {
		t.Errorf("expected resolveSLITarget to return associated SLO target 99.95, got %v", target2)
	}
}

func TestMissingTargetNoFabricatedDefaults(t *testing.T) {
	store := NewReliabilityStore(t.TempDir() + "/test-store.json")

	types := []string{"availability", "error_rate", "latency", "throughput", "saturation"}
	for _, stype := range types {
		sli := store.SaveSLI(SLIItem{
			Name:      "Unconfigured SLI " + stype,
			ClusterID: "c-no-target",
			Service:   "demo",
			Namespace: "default",
			Type:      stype,
			Target:    0,
		})

		target := resolveSLITarget(sli, store, "c-no-target")
		if target != nil {
			t.Fatalf("expected nil target for type %s without config, got %f", stype, *target)
		}

		val := 99.9
		status := CalculateSLIStatus(&val, target, stype)
		if status != "unavailable" {
			t.Fatalf("expected status 'unavailable' for unconfigured target of type %s, got %s", stype, status)
		}
	}
}

func TestSLIUpdateImmutabilityAndPatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterReliabilityRoutes(router)

	clusterID := "cluster-immutability-test"
	sliID := "sli-orig-123"

	// Create initial SLI
	initJSON := `{
		"id": "` + sliID + `",
		"name": "Original SLI",
		"service": "checkout",
		"namespace": "prod",
		"type": "availability",
		"target": 99.9,
		"query": "up",
		"enabled": true
	}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/clusters/"+clusterID+"/slis", bytes.NewBufferString(initJSON))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("failed to create initial SLI, got code %d", w.Code)
	}

	var created SLIItem
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	createdTime := created.CreatedAt

	// Test 1: PUT attempting to change ID, ClusterID, and CreatedAt
	putPayload := `{
		"id": "hacked-id",
		"clusterId": "hacked-cluster",
		"name": "Updated Name via PUT",
		"service": "checkout",
		"namespace": "prod",
		"type": "availability",
		"target": 99.95,
		"query": "up"
	}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PUT", "/api/clusters/"+clusterID+"/slis/"+sliID, bytes.NewBufferString(putPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for PUT update, got %d", w.Code)
	}

	var putRes SLIItem
	_ = json.Unmarshal(w.Body.Bytes(), &putRes)
	if putRes.ID != sliID {
		t.Errorf("PUT allowed changing ID! Expected %s, got %s", sliID, putRes.ID)
	}
	if putRes.ClusterID != clusterID {
		t.Errorf("PUT allowed changing ClusterID! Expected %s, got %s", clusterID, putRes.ClusterID)
	}
	if !putRes.CreatedAt.Equal(createdTime) {
		t.Errorf("PUT allowed changing CreatedAt! Expected %v, got %v", createdTime, putRes.CreatedAt)
	}
	if putRes.Name != "Updated Name via PUT" || putRes.Target != 99.95 {
		t.Errorf("PUT failed to update legitimate config fields, got name: %s, target: %f", putRes.Name, putRes.Target)
	}

	// Test 2: PATCH partial update with target only -> preserves Name, Query, Service, Namespace
	patchPayload := `{"target": 99.99}`
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("PATCH", "/api/clusters/"+clusterID+"/slis/"+sliID, bytes.NewBufferString(patchPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for PATCH update, got %d", w.Code)
	}

	var patchRes SLIItem
	_ = json.Unmarshal(w.Body.Bytes(), &patchRes)
	if patchRes.Target != 99.99 {
		t.Errorf("PATCH failed to update target! Expected 99.99, got %f", patchRes.Target)
	}
	if patchRes.Name != "Updated Name via PUT" || patchRes.Service != "checkout" || patchRes.Namespace != "prod" || patchRes.Query != "up" {
		t.Errorf("PATCH erased unsupplied fields! Got name: %s, service: %s, namespace: %s, query: %s", patchRes.Name, patchRes.Service, patchRes.Namespace, patchRes.Query)
	}
	if patchRes.ID != sliID || patchRes.ClusterID != clusterID {
		t.Errorf("PATCH altered immutable identity fields! Got ID: %s, ClusterID: %s", patchRes.ID, patchRes.ClusterID)
	}
}
