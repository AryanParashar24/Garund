package agent

import (
	"fmt"
)

// Run starts the Garund cluster agent (Phase 4 stub).
// The agent will maintain an outbound gRPC connection to Garund Cloud
// and proxy Kubernetes API access without exposing the cluster to the internet.
func Run(serverURL, enrollmentToken string) error {
	return fmt.Errorf(
		"garund agent is not yet implemented — coming in Phase 4 (multi-cluster)\n\n"+
			"  Planned: outbound TLS/gRPC to %s\n"+
			"  For now, use: garund local",
		serverURL,
	)
}
