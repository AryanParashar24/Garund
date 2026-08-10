package main

import (
	"context"

	"garund-backend/analyzer"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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
