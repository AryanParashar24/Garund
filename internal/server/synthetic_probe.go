package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type SyntheticProbeConfig struct {
	Name      string
	URL       string
	Interval  time.Duration
	Timeout   time.Duration
	Namespace string
	Service   string
}

type SyntheticProbe struct {
	config SyntheticProbeConfig
	client *http.Client

	requestsTotal metric.Int64Counter
	failuresTotal metric.Int64Counter
	latency       metric.Float64Histogram
	statusesTotal metric.Int64Counter
}

var (
	syntheticProbeOnce sync.Once
	syntheticProbe     *SyntheticProbe
)

func NewSyntheticProbe(
	config SyntheticProbeConfig,
) (*SyntheticProbe, error) {

	meter := otel.Meter("garund.synthetic")

	requestsTotal, err := meter.Int64Counter(
		"garund.synthetic.requests.total",
		metric.WithDescription(
			"Total number of synthetic reliability probe requests",
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create synthetic request counter: %w",
			err,
		)
	}

	failuresTotal, err := meter.Int64Counter(
		"garund.synthetic.failures.total",
		metric.WithDescription(
			"Total number of failed synthetic reliability probes",
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create synthetic failure counter: %w",
			err,
		)
	}

	latency, err := meter.Float64Histogram(
		"garund.synthetic.request.duration",
		metric.WithDescription(
			"Duration of synthetic reliability probe requests",
		),
		metric.WithUnit("ms"),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create synthetic latency histogram: %w",
			err,
		)
	}

	statusesTotal, err := meter.Int64Counter(
		"garund.synthetic.responses.total",
		metric.WithDescription(
			"HTTP responses returned by synthetic reliability probes",
		),
	)

	if err != nil {
		return nil, fmt.Errorf(
			"create synthetic response counter: %w",
			err,
		)
	}

	return &SyntheticProbe{
		config: config,

		client: &http.Client{
			Timeout: config.Timeout,
		},

		requestsTotal: requestsTotal,
		failuresTotal: failuresTotal,
		latency:       latency,
		statusesTotal: statusesTotal,
	}, nil
}

func (p *SyntheticProbe) Run(ctx context.Context) {

	/*
		Do not immediately probe when Garund starts.

		Give the backend a few seconds to finish
		initializing its HTTP server.
	*/

	startDelay := 5 * time.Second

	timer := time.NewTimer(startDelay)

	select {
	case <-ctx.Done():
		timer.Stop()
		return

	case <-timer.C:
	}

	/*
		Perform the first probe.
	*/

	p.probe(ctx)

	/*
		Continue probing periodically.
	*/

	ticker := time.NewTicker(p.config.Interval)
	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():
			return

		case <-ticker.C:
			p.probe(ctx)
		}
	}
}

func (p *SyntheticProbe) probe(
	parentCtx context.Context,
) {

	ctx, cancel := context.WithTimeout(
		parentCtx,
		p.config.Timeout,
	)

	defer cancel()

	start := time.Now()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		p.config.URL,
		nil,
	)

	/*
		Common attributes attached to every measurement.
	*/

	attrs := []attribute.KeyValue{
		attribute.String(
			"garund.probe.name",
			p.config.Name,
		),

		attribute.String(
			"garund.probe.target",
			p.config.URL,
		),

		attribute.String(
			"service.name",
			p.config.Service,
		),

		attribute.String(
			"k8s.namespace.name",
			p.config.Namespace,
		),
	}

	p.requestsTotal.Add(
		ctx,
		1,
		metric.WithAttributes(attrs...),
	)

	if err != nil {

		p.failuresTotal.Add(
			ctx,
			1,
			metric.WithAttributes(
				attrs...,
			),
		)

		p.latency.Record(
			ctx,
			float64(time.Since(start).Microseconds())/1000.0,
			metric.WithAttributes(
				attrs...,
			),
		)

		return
	}

	resp, err := p.client.Do(req)

	elapsed := time.Since(start)

	latencyMs :=
		float64(elapsed.Microseconds()) /
			1000.0

	p.latency.Record(
		ctx,
		latencyMs,
		metric.WithAttributes(
			attrs...,
		),
	)

	if err != nil {

		p.failuresTotal.Add(
			ctx,
			1,
			metric.WithAttributes(
				attrs...,
			),
		)

		return
	}

	defer resp.Body.Close()

	/*
		Record HTTP status.
	*/

	statusAttrs := append(
		attrs,
		attribute.Int(
			"http.response.status_code",
			resp.StatusCode,
		),
	)

	p.statusesTotal.Add(
		ctx,
		1,
		metric.WithAttributes(
			statusAttrs...,
		),
	)

	/*
		2xx and 3xx are considered successful
		for the synthetic availability SLI.
	*/

	success :=
		resp.StatusCode >= 200 &&
			resp.StatusCode < 400

	if !success {

		p.failuresTotal.Add(
			ctx,
			1,
			metric.WithAttributes(
				attrs...,
			),
		)
	}
}

/*
	StartSyntheticReliabilityProbe starts Garund's
	background synthetic reliability monitor.

	The worker is started once even if this function
	is accidentally called more than once.
*/

func StartSyntheticReliabilityProbe(
	ctx context.Context,
) error {

	syntheticProbeOnce.Do(func() {

		url := os.Getenv(
			"GARUND_PROBE_URL",
		)

		if url == "" {
			url = "http://localhost:8080/health"
		}

		interval := 15 * time.Second

		if value := os.Getenv(
			"GARUND_PROBE_INTERVAL",
		); value != "" {

			if seconds, err :=
				strconv.Atoi(value); err == nil &&
				seconds > 0 {

				interval =
					time.Duration(seconds) *
						time.Second
			}
		}

		timeout := 3 * time.Second

		if value := os.Getenv(
			"GARUND_PROBE_TIMEOUT",
		); value != "" {

			if seconds, err :=
				strconv.Atoi(value); err == nil &&
				seconds > 0 {

				timeout =
					time.Duration(seconds) *
						time.Second
			}
		}

		config := SyntheticProbeConfig{
			Name: "garund-api",

			URL: url,

			Interval: interval,

			Timeout: timeout,

			Service: "garund",

			Namespace: "garund",
		}

		probe, err :=
			NewSyntheticProbe(config)

		if err != nil {
			fmt.Printf(
				"failed to initialize synthetic reliability probe: %v\n",
				err,
			)

			return
		}

		syntheticProbe = probe

		go func() {

			fmt.Printf(
				"synthetic reliability probe started: %s every %s\n",
				config.URL,
				config.Interval,
			)

			probe.Run(ctx)

		}()
	})

	return nil
}
