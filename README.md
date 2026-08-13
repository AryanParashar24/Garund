# Garund

Garund is a Kubernetes SRE observability and reliability control-plane application. It provides real-time workload discovery, topology mapping, synthetic reliability probing, SLI/SLO/SLA management, PromQL-based reliability evaluation, error budget calculations, and multi-channel alert routing.

---

## Core Experience

Install Garund and launch it directly from your terminal:

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

### Installation Script

Install the latest pre-compiled binary for your system (`linux`, `darwin`, `windows`):

```bash
curl -fsSL https://raw.githubusercontent.com/AryanParashar24/Garund/master/scripts/install.sh | sh
```

### Build & Install from Source

```bash
git clone https://github.com/AryanParashar24/Garund.git
cd Garund

# Build binary locally into bin/garund
make build

# Install binary to system PATH (~/.local/bin/garund)
make install
```

---

## CLI Usage

Garund includes a supervisor CLI for managing process lifecycles and diagnosing cluster connectivity.

| Command | Description |
| :--- | :--- |
| `garund start` | Launch the Garund server and embedded dashboard |
| `garund status` | Show process status, PID, address, and cluster connection |
| `garund doctor` | Run full environment, permissions, port, and API diagnostics |
| `garund logs` | Display sanitized application log output |
| `garund restart` | Gracefully restart the Garund server |
| `garund stop` | Stop running Garund processes cleanly |
| `garund version` | Output build version, commit hash, and platform details |

### Configuration Flags (`garund start`)

* `--host` (default: `127.0.0.1`) - Host address to bind
* `--port` (default: `8080`) - HTTP port to listen on
* `--kubeconfig` (default: `~/.kube/config`) - Path to Kubernetes configuration file
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
        supervisor               config
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

### Testing & Verification

```bash
make test    # Run Go unit and integration tests
make build   # Build production single binary
make release # Cross-compile distribution binaries into dist/
```

---

## Security Guarantees

* **Localhost Binding**: Garund binds to `127.0.0.1` by default to prevent unauthorized network exposure.
* **SSRF Protection**: Strict URL validation and IP blocklists prevent server-side request forgery during webhook and alert routing.
* **Secret Redaction**: Kubeconfig credentials, auth headers, and API tokens are automatically sanitized in log output.
