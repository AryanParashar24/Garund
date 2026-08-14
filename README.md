# Garund

Garund is a Kubernetes SRE observability and reliability control-plane application. It provides real-time workload discovery, topology mapping, synthetic reliability probing, SLI/SLO/SLA management, PromQL-based reliability evaluation, error budget calculations, and multi-channel alert routing.

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
