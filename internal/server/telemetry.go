package server

import (
	"context"
	"sync"

	"github.com/garund/garund/pkg/analyzer"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

func RecordAnalysis(
	ctx context.Context,
	kind string,
	namespace string,
	name string,
	healthy bool,
	findings []analyzer.Finding,
) {
	tracer := otel.Tracer("garund.analyzer")

	_, span := tracer.Start(
		ctx,
		"garund.analyze."+kind,
		trace.WithAttributes(
			attribute.String("k8s.resource.kind", kind),
			attribute.String("k8s.namespace.name", namespace),
			attribute.String("k8s.resource.name", name),
			attribute.Bool("garund.healthy", healthy),
			attribute.Int(
				"garund.findings.count",
				len(findings),
			),
		),
	)

	defer span.End()

	for _, finding := range findings {
		span.AddEvent(
			"garund.finding",
			trace.WithAttributes(
				attribute.String(
					"garund.finding.severity",
					finding.Severity,
				),
				attribute.String(
					"garund.finding.reason",
					finding.Reason,
				),
				attribute.String(
					"garund.finding.message",
					finding.Message,
				),
				attribute.String(
					"garund.finding.evidence",
					finding.Evidence,
				),
			),
		)
	}
}

// ReliabilityTracer provides tracing helper functions for reliability operations
func RecordReliabilitySpan(ctx context.Context, spanName string, clusterID, sliID, sloID, service, namespace string) (context.Context, trace.Span) {
	tracer := otel.Tracer("garund.reliability")
	attrs := []attribute.KeyValue{
		attribute.String("garund.cluster.id", clusterID),
	}
	if sliID != "" {
		attrs = append(attrs, attribute.String("garund.sli.id", sliID))
	}
	if sloID != "" {
		attrs = append(attrs, attribute.String("garund.slo.id", sloID))
	}
	if service != "" {
		attrs = append(attrs, attribute.String("garund.service", service))
	}
	if namespace != "" {
		attrs = append(attrs, attribute.String("garund.namespace", namespace))
	}

	return tracer.Start(ctx, spanName, trace.WithAttributes(attrs...))
}

// Internal Garund Metrics Counters
type GarundMetrics struct {
	ReliabilityQueriesTotal     metric.Int64Counter
	ReliabilityQueryErrorsTotal metric.Int64Counter
	SLIEvaluationsTotal         metric.Int64Counter
	SLIEvaluationErrorsTotal    metric.Int64Counter
	SLOEvaluationsTotal         metric.Int64Counter
	AlertsReceivedTotal         metric.Int64Counter
	PrometheusQueryDuration     metric.Float64Histogram
}

var (
	globalMetrics     *GarundMetrics
	globalMetricsOnce sync.Once
)

func GetGarundMetrics() *GarundMetrics {
	globalMetricsOnce.Do(func() {
		meter := otel.Meter("garund.internal")

		queriesTotal, _ := meter.Int64Counter("garund_reliability_queries_total")
		queryErrorsTotal, _ := meter.Int64Counter("garund_reliability_query_errors_total")
		sliEvalTotal, _ := meter.Int64Counter("garund_sli_evaluations_total")
		sliEvalErrorsTotal, _ := meter.Int64Counter("garund_sli_evaluation_errors_total")
		sloEvalTotal, _ := meter.Int64Counter("garund_slo_evaluations_total")
		alertsReceivedTotal, _ := meter.Int64Counter("garund_alerts_received_total")
		queryDuration, _ := meter.Float64Histogram("garund_prometheus_query_duration_seconds", metric.WithUnit("s"))

		globalMetrics = &GarundMetrics{
			ReliabilityQueriesTotal:     queriesTotal,
			ReliabilityQueryErrorsTotal: queryErrorsTotal,
			SLIEvaluationsTotal:         sliEvalTotal,
			SLIEvaluationErrorsTotal:    sliEvalErrorsTotal,
			SLOEvaluationsTotal:         sloEvalTotal,
			AlertsReceivedTotal:         alertsReceivedTotal,
			PrometheusQueryDuration:     queryDuration,
		}
	})
	return globalMetrics
}
