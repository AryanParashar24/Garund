package server

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
)

/*
ReliabilityConfig defines the SLO targets Garund evaluates.
*/
type ReliabilityConfig struct {
	AvailabilityTarget float64
	LatencyTargetMs    float64
	LatencyCompliance  float64
	ErrorRateTarget    float64
	Window             time.Duration
}

/*
ReliabilityResult is the complete SLI/SLO/SLA response.
*/
type ReliabilityResult struct {
	Service   string `json:"service"`
	Namespace string `json:"namespace"`

	Window string `json:"window"`

	SLIs []SLI `json:"slis"`

	SLO SLO `json:"slo"`

	SLA SLA `json:"sla"`
}

/*
SLI represents one reliability indicator.

Value is a pointer intentionally.

nil  = telemetry unavailable
0    = telemetry exists and measured zero
*/
type SLI struct {
	Name string `json:"name"`
	Type string `json:"type"`

	Value  *float64 `json:"value"`
	Target float64  `json:"target"`

	Unit string `json:"unit"`

	GoodEvents  int64 `json:"goodEvents"`
	TotalEvents int64 `json:"totalEvents"`

	Window string `json:"window"`

	Status string `json:"status"`
}

/*
SLO represents a reliability objective.
*/
type SLO struct {
	Name string `json:"name"`

	Service   string `json:"service"`
	Namespace string `json:"namespace"`

	Target float64 `json:"target"`

	Window string `json:"window"`

	SLIType string `json:"sliType"`

	Current *float64 `json:"current"`

	ErrorBudget          float64 `json:"errorBudget"`
	ErrorBudgetRemaining float64 `json:"errorBudgetRemaining"`

	Status string `json:"status"`
}

/*
SLA represents the contractual reliability commitment.
*/
type SLA struct {
	Name string `json:"name"`

	Service   string `json:"service"`
	Namespace string `json:"namespace"`

	AvailabilityTarget *float64 `json:"availabilityTarget,omitempty"`
	LatencyTargetMs    *float64 `json:"latencyTargetMs,omitempty"`

	Window string `json:"window"`

	Status string `json:"status"`
}

/*
ReliabilityStatus converts an SLI measurement
into a dashboard status.
*/
func ReliabilityStatus(
	value float64,
	target float64,
) string {

	if value >= target {
		return "healthy"
	}

	difference := target - value

	if difference <= 0.5 {
		return "warning"
	}

	return "critical"
}

/*
ErrorBudgetRemaining returns the percentage
of the allowed error budget that remains.
*/
func ErrorBudgetRemaining(
	slo float64,
	current float64,
) float64 {

	if slo >= 100 {
		return 100
	}

	allowedError := 100 - slo

	actualError := 100 - current

	if actualError <= 0 {
		return 100
	}

	remaining :=
		((allowedError - actualError) /
			allowedError) * 100

	if remaining < 0 {
		return 0
	}

	if remaining > 100 {
		return 100
	}

	return remaining
}

/*
ErrorBudgetMinutes converts the SLO window
into allowed downtime.
*/
func ErrorBudgetMinutes(
	target float64,
	window time.Duration,
) float64 {

	errorFraction :=
		(100 - target) / 100

	return window.Minutes() *
		errorFraction
}

/*
roundReliability keeps API values clean.
*/
func roundReliability(
	value float64,
) float64 {

	return math.Round(
		value*100,
	) / 100
}

/*
registerReliabilityRoutes adds the reliability API.

Prometheus is the source of request-level
telemetry.

Kubernetes Events are NOT used to fabricate
availability or error-rate measurements.
*/
func registerReliabilityRoutes(
	router *gin.Engine,
	clientset kubernetes.Interface,
	prometheusClient *PrometheusClient,
) {

	router.GET("/reliability", func(c *gin.Context) {

		namespace := c.Query("namespace")
		serviceName := c.Query("service")

		availabilityTarget := 99.9
		errorRateTarget := 0.1
		latencyTarget := 300.0

		/*
			Prometheus queries.

			These are currently cluster-wide.

			We will make them service-aware after
			confirming the labels exposed by OTel.
		*/

		totalQuery := `
			sum(
				rate(
					http_server_request_duration_seconds_count[5m]
				)
			)
		`

		successQuery := `
			sum(
				rate(
					http_server_request_duration_seconds_count{
						http_response_status_code=~"2..|3.."
					}[5m]
				)
			)
		`

		errorQuery := `
			sum(
				rate(
					http_server_request_duration_seconds_count{
						http_response_status_code=~"5.."
					}[5m]
				)
			)
		`

		latencyQuery := `
			histogram_quantile(
				0.95,
				sum(
					rate(
						http_server_request_duration_seconds_bucket[5m]
					)
				) by (le)
			) * 1000
		`

		/*
			Total requests.
		*/
		totalRequests,
			totalAvailable,
			err := prometheusClient.QueryOptional(
			totalQuery,
		)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": err.Error(),
				},
			)
			return
		}

		/*
			Successful requests.
		*/
		successfulRequests,
			successAvailable,
			err := prometheusClient.QueryOptional(
			successQuery,
		)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": err.Error(),
				},
			)
			return
		}

		/*
			5xx requests.
		*/
		errorRequests,
			errorAvailable,
			err := prometheusClient.QueryOptional(
			errorQuery,
		)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": err.Error(),
				},
			)
			return
		}

		/*
			p95 latency.
		*/
		latency,
			latencyAvailable,
			err := prometheusClient.QueryOptional(
			latencyQuery,
		)

		if err != nil {
			c.JSON(
				http.StatusInternalServerError,
				gin.H{
					"error": err.Error(),
				},
			)
			return
		}

		/*
			Calculate SLIs.
		*/

		availability :=
			calculateAvailabilitySLI(
				successfulRequests,
				totalRequests,
				availabilityTarget,
				totalAvailable &&
					successAvailable,
			)

		errorRate :=
			calculateErrorRateSLI(
				errorRequests,
				totalRequests,
				errorRateTarget,
				totalAvailable &&
					errorAvailable,
			)

		latencySLI :=
			calculateLatencySLI(
				latency,
				latencyTarget,
				latencyAvailable,
			)

		/*
			Construct SLI API objects.
		*/

		availabilityMetric := SLI{
			Name: "Availability",
			Type: "availability",

			Value: availability.Value,

			Target: availabilityTarget,
			Unit:   "%",

			GoodEvents:  availability.GoodEvents,
			TotalEvents: availability.TotalEvents,

			Window: "5m",

			Status: availability.Status,
		}

		errorRateMetric := SLI{
			Name: "Error Rate",
			Type: "error_rate",

			Value: errorRate.Value,

			Target: errorRateTarget,
			Unit:   "%",

			GoodEvents:  errorRate.GoodEvents,
			TotalEvents: errorRate.TotalEvents,

			Window: "5m",

			Status: errorRate.Status,
		}

		latencyMetric := SLI{
			Name: "Latency",
			Type: "latency",

			Value: latencySLI.Value,

			Target: latencyTarget,
			Unit:   "ms",

			GoodEvents:  0,
			TotalEvents: 0,

			Window: "5m",

			Status: latencySLI.Status,
		}

		/*
			SLO evaluation.

			No telemetry means unavailable,
			not 0%.
		*/

		var current *float64

		if availability.Value != nil {
			current = availability.Value
		}

		errorBudget :=
			100 - availabilityTarget

		errorBudgetRemaining := 0.0

		sloStatus := "unavailable"

		if current != nil {

			errorBudgetRemaining =
				ErrorBudgetRemaining(
					availabilityTarget,
					*current,
				)

			sloStatus =
				ReliabilityStatus(
					*current,
					availabilityTarget,
				)
		}

		slo := SLO{
			Name: "Service Availability",

			Service:   serviceName,
			Namespace: namespace,

			Target: availabilityTarget,

			Window: "30d",

			SLIType: "availability",

			Current: current,

			ErrorBudget: errorBudget,

			ErrorBudgetRemaining: errorBudgetRemaining,

			Status: sloStatus,
		}

		/*
			SLA evaluation.
		*/

		slaStatus := "unavailable"

		if current != nil {

			switch {
			case *current < availabilityTarget:
				slaStatus = "breached"

			case *current <
				availabilityTarget+0.2:
				slaStatus = "at_risk"

			default:
				slaStatus = "compliant"
			}
		}

		sla := SLA{
			Name: "Standard Service SLA",

			Service:   serviceName,
			Namespace: namespace,

			AvailabilityTarget: &availabilityTarget,

			LatencyTargetMs: &latencyTarget,

			Window: "30d",

			Status: slaStatus,
		}

		/*
			Return reliability response.
		*/

		c.JSON(
			http.StatusOK,
			ReliabilityResult{
				Service:   serviceName,
				Namespace: namespace,

				Window: "30d",

				SLIs: []SLI{
					availabilityMetric,
					latencyMetric,
					errorRateMetric,
				},

				SLO: slo,
				SLA: sla,
			},
		)
	})

	/*
		Keep clientset referenced for now.

		The reliability engine will use it once
		we add Kubernetes-to-service telemetry
		mapping.
	*/
	_ = clientset
}
