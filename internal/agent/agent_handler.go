package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/garund/garund/internal/kubernetes"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type AgentHeartbeatMessage struct {
	ClusterID         string `json:"clusterId"`
	AgentVersion      string `json:"agentVersion"`
	KubernetesVersion string `json:"kubernetesVersion"`
	NodeCount         int    `json:"nodeCount"`
	NamespaceCount    int    `json:"namespaceCount"`
	LatencyMs         int64  `json:"latencyMs"`
	Status            string `json:"status"`
}

type AgentRegistry struct {
	mu          sync.RWMutex
	connections map[string]*websocket.Conn
	tokens      map[string]string // clusterID -> agentToken
}

var globalAgentRegistry = &AgentRegistry{
	connections: make(map[string]*websocket.Conn),
	tokens:      make(map[string]string),
}

func GetAgentRegistry() *AgentRegistry {
	return globalAgentRegistry
}

func (r *AgentRegistry) RegisterToken(clusterID, token string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[clusterID] = token
}

func (r *AgentRegistry) ValidateToken(clusterID, token string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	expectedToken, exists := r.tokens[clusterID]
	if !exists {
		// Allow token match if cluster connection exists with matching agent token
		if conn, found := kubernetes.GetManager().GetCluster(clusterID); found {
			return conn.AgentToken == "" || conn.AgentToken == token
		}
		return true
	}
	return expectedToken == token
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func HandleAgentWebSocket(c *gin.Context) {
	clusterID := c.Param("id")
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("X-Garund-Agent-Token")
	}

	if !globalAgentRegistry.ValidateToken(clusterID, token) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid agent token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Agent websocket upgrade error for cluster %s: %v", clusterID, err)
		return
	}

	globalAgentRegistry.mu.Lock()
	globalAgentRegistry.connections[clusterID] = conn
	globalAgentRegistry.mu.Unlock()

	defer func() {
		globalAgentRegistry.mu.Lock()
		delete(globalAgentRegistry.connections, clusterID)
		globalAgentRegistry.mu.Unlock()
		conn.Close()

		kubernetes.GetManager().UpdateStatus(
			clusterID,
			kubernetes.StatusDisconnected,
			"",
			0,
			0,
			0,
		)
	}()

	kubernetes.GetManager().UpdateStatus(
		clusterID,
		kubernetes.StatusConnected,
		"",
		0,
		0,
		0,
	)

	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var hb AgentHeartbeatMessage
		if err := json.Unmarshal(message, &hb); err == nil {
			status := kubernetes.StatusConnected
			if hb.Status == "degraded" {
				status = kubernetes.StatusDegraded
			}

			kubernetes.GetManager().UpdateStatus(
				clusterID,
				status,
				hb.KubernetesVersion,
				hb.NodeCount,
				hb.NamespaceCount,
				hb.LatencyMs,
			)
		}
	}
}

func GenerateAgentRBACManifest(clusterID, serverURL, agentToken string) string {
	return fmt.Sprintf(`---
apiVersion: v1
kind: Namespace
metadata:
  name: garund-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: garund-agent
  namespace: garund-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: garund-agent-role
rules:
- apiGroups: [""]
  resources: ["pods", "services", "namespaces", "nodes", "events", "endpoints"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "replicasets", "statefulsets", "daemonsets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["pods/log"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: garund-agent-binding
subjects:
- kind: ServiceAccount
  name: garund-agent
  namespace: garund-system
roleRef:
  kind: ClusterRole
  name: garund-agent-role
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: garund-agent
  namespace: garund-system
  labels:
    app.kubernetes.io/name: garund-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: garund-agent
  template:
    metadata:
      labels:
        app.kubernetes.io/name: garund-agent
    spec:
      serviceAccountName: garund-agent
      containers:
      - name: garund-agent
        image: garund/agent:v1.0.0
        env:
        - name: GARUND_SERVER_URL
          value: "%s"
        - name: GARUND_CLUSTER_ID
          value: "%s"
        - name: GARUND_AGENT_TOKEN
          value: "%s"
        resources:
          limits:
            cpu: 200m
            memory: 256Mi
          requests:
            cpu: 50m
            memory: 64Mi
`, serverURL, clusterID, agentToken)
}
