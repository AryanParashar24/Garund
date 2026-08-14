# Garund

Garund is a unified, cloud-native Kubernetes SRE observability and reliability control-plane application. Built for **Public Clouds** (EKS, GKE, AKS), **Private Clouds & On-Premises** (OpenShift, Rancher, Bare-Metal), and **Local/Edge Environments** (K3s, Kind, Minikube), Garund provides real-time workload discovery, interactive resource topology mapping, synthetic reliability probing, SLI/SLO/SLA management, PromQL-based reliability evaluation, error budget calculations, and multi-channel alert routing.

---

## Key Features & Capabilities

### 1. Hybrid, Multi-Cloud & On-Premises Infrastructure Support
Garund runs seamlessly across any Kubernetes environment without vendor lock-in:
* **Public Cloud Kubernetes**: Native support for **AWS EKS**, **Google Cloud GKE**, **Azure AKS**, **DigitalOcean DOKS**, and **Civo**.
* **Private Cloud & On-Premises**: Complete observability for **Red Hat OpenShift**, **SUSE Rancher**, **VMware Tanzu**, and **Bare-Metal** Kubernetes deployments.
* **Local & Edge Clusters**: Lightweight footprint optimized for **K3s**, **Kind**, **Minikube**, and **MicroK8s**.
* **Secure In-Cluster Agent (Private & Air-Gapped Networks)**: Connect on-premises and private cloud clusters securely using the lightweight outbound-only `garund-agent`, eliminating the need to expose Kubernetes API servers to public ingress or modify firewall rules.

### 2.  Interactive Resource Topology View
Garund dynamically constructs a live 2D dependency graph of your Kubernetes clusters:
* **Full Hierarchy Mapping**: Visualizes relationships across **Services → Deployments → ReplicaSets → Pods** in real time.
* **Instant Resource Inspection**: Click on any workload node in the graph to view live pod logs, container status, configuration manifests, and warning events.
* **Global Search Integration**: Press `/` to search for any resource across namespaces; selecting a result automatically highlights and centers the node in the topology graph.
* **Cascading Failure Isolation**: Visually track degraded pods and failing ReplicaSets to trace root-cause incidents across microservice boundaries.

### 3. Live Cluster Event Stream & Warning Flagging
Garund continuously monitors and flags cluster events in real time:
* **Real-Time Event Stream**: Streams live Kubernetes cluster audit and state events across all namespaces.
* **Incident Flagging**: Automatically highlights critical failure patterns, including `CrashLoopBackOff`, `OOMKilled`, `ImagePullBackOff`, `Unhealthy` probes, and `FailedScheduling`.
* **Actionable Diagnostics**: Displays precise event timestamps, namespace scopes, resource actions, and raw failure context for rapid incident response.

### 4. Automated Resource Health Detection & Timestamp Tracking
Comprehensive workload health scoring and lifecycle tracking:
* **Dynamic Cluster Health Score**: Continuously evaluates a unified Cluster Health Score (0–100%) calculated dynamically from workload crash loops, pod restarts, and event severities.
* **Timestamped Lifecycles**: Tracks creation timestamps, age/uptime durations, last-seen timestamps, and restart counts across all Pods, Deployments, ReplicaSets, and Services.
* **Status Categorization**: Automatically categorizes resources into healthy, degraded, and failed states for immediate visual identification.

### 5. PromQL Integration & Guided SLI Builder (No PromQL Knowledge Required)
Garund integrates directly with Prometheus telemetry to measure workload reliability, while making Service Level Indicators (SLIs) accessible to every team member:
* **No PromQL Required**: If you don't know PromQL, Garund's **Guided SLI Builder** allows you to configure reliability targets visually. Simply select your service, desired measurement type (*Availability %*, *Error Rate %*, *Latency ms*, *Throughput req/s*), and percentile (*p50*, *p90*, *p95*, *p99*). Garund handles the query generation automatically.
* **Advanced PromQL Support**: For teams with existing Prometheus expertise, input custom PromQL queries with live syntax validation and test execution.
* **Query Explainability**: Every generated or custom PromQL query includes a plain-English explainability drawer, breaking down exactly how metrics and rates are calculated.
* **Truthful Telemetry Guarantees**: Garund never fabricates health data or defaults to fake green indicators. If telemetry is unreachable or returned empty, Garund reports state as `N/A (Unavailable)`.

### 6. Telemetry & OpenTelemetry (OTel) Integration
Unified telemetry pipeline for infrastructure and application performance:
* **Prometheus & OTel Collectors**: Directly ingests metrics from Prometheus endpoints and OpenTelemetry collectors.
* **Telemetry Correlation**: Links cluster infrastructure metrics directly with synthetic HTTP/TCP/gRPC probes and application-level SLI/SLO measurements.

### 7. SLI/SLO/SLA Management & Error Budget Engine
* **Error Budget & Burn Rate**: Continuously calculates remaining error budgets and alert burn rates across 5-minute to 24-hour evaluation windows.
* **Multi-Channel Alerting**: Connects with Webhooks, Prometheus Alertmanager, and PagerDuty with strict SSRF protection.
* **Multi-Cluster Workspaces**: Manage local Kubernetes contexts and remote in-cluster agents from a unified control plane.

---

## Core Experience

Install Garund and launch it directly from your terminal from any directory:

```bash
# Start Garund
garund start
```

Open your browser at:

```
http://127.0.0.1:8080
```

No manual Node.js, npm, or Go development commands are required for running the packaged product.

---

## Installation

Choose one of the following installation methods:

### Option 1: Automated One-Line Installer Script

Install Garund directly via shell installer script (downloads pre-compiled release binary or automatically compiles from source fallback if releases are not yet published):

```bash
curl -fsSL https://raw.githubusercontent.com/AryanParashar24/Garund/master/scripts/install.sh | sh
```

Then export `~/.local/bin` to your current PATH session:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

---

### Option 2: Build & Install from Source (Git Clone)

Clone the repository and install the `garund` binary to `~/.local/bin`:

```bash
git clone https://github.com/AryanParashar24/Garund.git
cd Garund

# Build static frontend & Go binary
make build

# Install binary to ~/.local/bin/garund
make install

# Add to PATH for current session
export PATH="$HOME/.local/bin:$PATH"

# Launch Garund
garund start
```

### System-Wide Installation

To install system-wide to `/usr/local/bin`:

```bash
PREFIX=/usr/local make install
```

---

## Environment & PATH Setup

To ensure `garund` is available in every new terminal session, add `~/.local/bin` to your shell profile:

```bash
# For bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc

# For zsh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
```

Verify your installation:

```bash
garund version
garund doctor
```

---

## Uninstallation

To remove the installed Garund binary while preserving your configuration and telemetry data:

```bash
# Using Makefile
make uninstall

# Or using script
./scripts/uninstall.sh
```

To completely purge all local configuration, state, and logs (`~/.garund/`):

```bash
./scripts/uninstall.sh --purge
```

---

## CLI Usage Reference

Garund includes a supervisor CLI for managing process lifecycles and diagnosing cluster connectivity.

| Command | Description |
| :--- | :--- |
| `garund start` | Launch the Garund server and embedded dashboard |
| `garund status` | Show process status, PID, listen address, and Kubernetes connection state |
| `garund doctor` | Run full environment, permissions, port, and API diagnostics |
| `garund logs` | Display sanitized application log output |
| `garund restart` | Gracefully restart the Garund server |
| `garund stop` | Stop running Garund processes cleanly |
| `garund version` | Output build version, git commit hash, build date, and platform details |
| `garund help` | Display CLI usage help |

### Configuration Flags (`garund start` / `garund restart`)

* `--host` (default: `127.0.0.1`) - Host address to bind
* `--port` (default: `8080`) - HTTP port to listen on
* `--kubeconfig` (default: `~/.kube/config` or `$KUBECONFIG`) - Path to Kubernetes configuration file
* `--context` - Specific Kubernetes context to target
* `--prometheus-url` - Custom Prometheus endpoint for SLI evaluation

---

## System Architecture

```
                       USER
                         │
                         │
                    garund CLI
                         │
             ┌───────────┴───────────┐
             │                       │
        supervisor               config / state
             │                (~/.garund/)
             ▼
       Garund Runtime
       (127.0.0.1:8080)
        ┌────┴─────┐
        │          │
     Embedded   Go Backend API
     Frontend   (Gin Engine)
                   │
                   ├──► Kubernetes API (~/.kube/config)
                   └──► Prometheus / Telemetry
```

---

## Development Setup

For contributors working on Garund source code:

```bash
# Start backend API & frontend dev server simultaneously
make dev
```

* Backend runs at `http://127.0.0.1:8080`
* Frontend dev server runs at `http://127.0.0.1:3000`

### Useful Development Commands

```bash
make frontend   # Build static frontend output
make backend    # Build Go CLI binary to bin/garund
make run        # Build and run ./bin/garund start
make test       # Run Go unit and integration tests
make lint       # Run go vet static analysis
make clean      # Clean build artifacts (bin/, dist/, internal/web/dist/)
make release    # Cross-compile distribution binaries into dist/
```

---

## Security Guarantees

* **Localhost Binding**: Garund binds to `127.0.0.1` by default to prevent unauthorized network exposure.
* **SSRF Protection**: Strict URL validation and IP blocklists prevent server-side request forgery during webhook and alert routing.
* **Secret Redaction**: Kubeconfig credentials, auth headers, and API tokens are automatically sanitized in log output.
