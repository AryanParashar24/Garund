package doctor

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/garund/garund/internal/buildinfo"
	"github.com/garund/garund/internal/kubernetes"
)

// Check represents a single diagnostic result.
type Check struct {
	Name    string
	Passed  bool
	Detail  string
	Remedy  string
}

// Report holds all diagnostic checks.
type Report struct {
	Checks []Check
}

// AllPassed returns true when every check passed.
func (r Report) AllPassed() bool {
	for _, c := range r.Checks {
		if !c.Passed {
			return false
		}
	}
	return true
}

// Run executes all Garund preflight checks.
func Run(serverURL string) Report {
	if serverURL == "" {
		serverURL = "http://localhost:8080"
	}

	report := Report{}

	report.Checks = append(report.Checks, checkVersion())
	report.Checks = append(report.Checks, checkKubeconfig()...)
	report.Checks = append(report.Checks, checkPrometheus()...)
	report.Checks = append(report.Checks, checkOTel()...)
	report.Checks = append(report.Checks, checkServer(serverURL)...)

	return report
}

func checkVersion() Check {
	return Check{
		Name:   "Garund version",
		Passed: true,
		Detail: fmt.Sprintf("%s (go %s)", buildinfo.Version, runtime.Version()),
	}
}

func checkKubeconfig() []Check {
	path, err := kubernetes.ResolveKubeconfig()
	if err != nil {
		return []Check{{
			Name:   "Kubernetes configuration",
			Passed: false,
			Detail: err.Error(),
			Remedy: "Ensure ~/.kube/config exists or set KUBECONFIG.",
		}}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []Check{{
			Name:   "Kubernetes configuration",
			Passed: false,
			Detail: fmt.Sprintf("Not found: %s", path),
			Remedy: "Install kubectl and configure a cluster context.",
		}}
	}

	info, err := kubernetes.NewClient(path)
	if err != nil {
		return []Check{{
			Name:   "Kubernetes configuration",
			Passed: false,
			Detail: err.Error(),
			Remedy: "Fix kubeconfig or run: kubectl cluster-info",
		}}
	}

	checks := []Check{
		{
			Name:   "Kubernetes configuration",
			Passed: true,
			Detail: path,
		},
		{
			Name:   "Current context",
			Passed: info.CurrentContext != "",
			Detail: info.CurrentContext,
			Remedy: "Run: kubectl config use-context <name>",
		},
	}

	if err := kubernetes.CanListPods(info); err != nil {
		checks = append(checks, Check{
			Name:   "Pods readable",
			Passed: false,
			Detail: err.Error(),
			Remedy: "Grant list/watch/get on pods to your kubeconfig user.",
		})
	} else {
		checks = append(checks, Check{Name: "Pods readable", Passed: true})
	}

	if err := kubernetes.CanListServices(info); err != nil {
		checks = append(checks, Check{
			Name:   "Services readable",
			Passed: false,
			Detail: err.Error(),
			Remedy: "Grant list/watch/get on services.",
		})
	} else {
		checks = append(checks, Check{Name: "Services readable", Passed: true})
	}

	if err := kubernetes.CanListEvents(info); err != nil {
		checks = append(checks, Check{
			Name:   "Events readable",
			Passed: false,
			Detail: err.Error(),
			Remedy: "Grant list/watch/get on events.",
		})
	} else {
		checks = append(checks, Check{Name: "Events readable", Passed: true})
	}

	return checks
}

func checkPrometheus() []Check {
	url := os.Getenv("GARUND_PROMETHEUS_URL")
	if url == "" {
		url = os.Getenv("PROMETHEUS_URL")
	}
	if url == "" {
		url = "http://localhost:9090"
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url + "/api/v1/query?query=up")
	if err != nil {
		return []Check{{
			Name:   "Prometheus reachable",
			Passed: false,
			Detail: fmt.Sprintf("Expected: %s", url),
			Remedy: "Start Prometheus or set GARUND_PROMETHEUS_URL.",
		}}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []Check{{
			Name:   "Prometheus reachable",
			Passed: false,
			Detail: fmt.Sprintf("%s returned %s", url, resp.Status),
			Remedy: "Start Prometheus or set GARUND_PROMETHEUS_URL.",
		}}
	}

	return []Check{{
		Name:   "Prometheus reachable",
		Passed: true,
		Detail: url,
	}}
}

func checkOTel() []Check {
	addr := os.Getenv("GARUND_OTEL_ENDPOINT")
	if addr == "" {
		addr = "localhost:4317"
	}

	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return []Check{{
			Name:   "OpenTelemetry configured",
			Passed: false,
			Detail: fmt.Sprintf("Cannot reach %s", addr),
			Remedy: "Start OTel Collector: docker compose -f docker-compose.observability.yml up -d",
		}}
	}
	conn.Close()

	return []Check{{
		Name:   "OpenTelemetry configured",
		Passed: true,
		Detail: addr,
	}}
}

func checkServer(serverURL string) []Check {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(serverURL + "/health")
	if err != nil {
		return []Check{{
			Name:   "Garund API reachable",
			Passed: false,
			Detail: "Server not running (optional check)",
			Remedy: "Run: garund local",
		}}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	passed := resp.StatusCode == http.StatusOK

	return []Check{{
		Name:   "Garund API reachable",
		Passed: passed,
		Detail: strings.TrimSpace(string(body)),
		Remedy: "Run: garund local",
	}}
}

// Format renders the report for terminal output.
func Format(r Report) string {
	var b strings.Builder
	b.WriteString("Garund Doctor\n\n")

	for _, c := range r.Checks {
		icon := "✓"
		if !c.Passed {
			icon = "✗"
		}
		b.WriteString(fmt.Sprintf("%s %s", icon, c.Name))
		if c.Detail != "" {
			b.WriteString(fmt.Sprintf("\n  %s", c.Detail))
		}
		if !c.Passed && c.Remedy != "" {
			b.WriteString(fmt.Sprintf("\n  Fix: %s", c.Remedy))
		}
		b.WriteString("\n\n")
	}

	if r.AllPassed() {
		b.WriteString("Everything looks good.\n")
	} else {
		b.WriteString("Some checks failed. See above for fixes.\n")
	}

	return b.String()
}
