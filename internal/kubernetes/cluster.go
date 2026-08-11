package kubernetes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterStatus string

const (
	StatusConnected    ClusterStatus = "CONNECTED"
	StatusDegraded     ClusterStatus = "DEGRADED"
	StatusDisconnected ClusterStatus = "DISCONNECTED"
	StatusAuthError    ClusterStatus = "AUTH_ERROR"
	StatusUnknown      ClusterStatus = "UNKNOWN"
)

type ConnectionMode string

const (
	ModeLocalKubeconfig      ConnectionMode = "local_kubeconfig"
	ModeAgent                ConnectionMode = "agent"
	ModeServiceAccountToken  ConnectionMode = "service_account_token"
)

type CapabilitySet struct {
	CanReadWorkloads  bool `json:"canReadWorkloads"`
	CanReadLogs       bool `json:"canReadLogs"`
	CanReadEvents     bool `json:"canReadEvents"`
	CanReadTelemetry  bool `json:"canReadTelemetry"`
	CanOperateWorkloads bool `json:"canOperateWorkloads"`
	CanAdminister     bool `json:"canAdminister"`
}

type ClusterConnection struct {
	ID                string         `json:"id"`
	Name              string         `json:"name"`
	Environment       string         `json:"environment"`
	Provider          string         `json:"provider"`
	ClusterType       string         `json:"clusterType"`
	ConnectionMode    ConnectionMode `json:"connectionMode"`
	Status            ClusterStatus  `json:"status"`
	Endpoint          string         `json:"endpoint"`
	KubernetesVersion string         `json:"kubernetesVersion"`
	AgentVersion      string         `json:"agentVersion,omitempty"`
	LastHeartbeat     time.Time      `json:"lastHeartbeat,omitempty"`
	LatencyMs         int64          `json:"latencyMs"`
	NodeCount         int            `json:"nodeCount"`
	NamespaceCount    int            `json:"namespaceCount"`
	Capabilities      CapabilitySet  `json:"capabilities"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`

	// Security: Sensitive internal fields NOT serialized to JSON
	KubeconfigPath string `json:"-"`
	KubeContext    string `json:"-"`
	BearerToken    string `json:"-"`
	AgentToken     string `json:"-"`
}

type ClusterManager struct {
	mu          sync.RWMutex
	clusters    map[string]*ClusterConnection
	clientsets  map[string]kubernetes.Interface
	activeID    string
	agentTunnel map[string]interface{} // Reserved for agent proxy connection state
}

var (
	globalManager *ClusterManager
	once          sync.Once
)

func GetManager() *ClusterManager {
	once.Do(func() {
		globalManager = NewClusterManager()
	})
	return globalManager
}

func NewClusterManager() *ClusterManager {
	cm := &ClusterManager{
		clusters:    make(map[string]*ClusterConnection),
		clientsets:  make(map[string]kubernetes.Interface),
		agentTunnel: make(map[string]interface{}),
	}
	return cm
}

func (cm *ClusterManager) RegisterCluster(conn *ClusterConnection, client kubernetes.Interface) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = time.Now()
	}
	conn.UpdatedAt = time.Now()

	cm.clusters[conn.ID] = conn
	if client != nil {
		cm.clientsets[conn.ID] = client
	}

	if cm.activeID == "" {
		cm.activeID = conn.ID
	}
}

func (cm *ClusterManager) ListClusters() []*ClusterConnection {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var list []*ClusterConnection
	for _, conn := range cm.clusters {
		// Return copy to prevent mutation
		cp := *conn
		list = append(list, &cp)
	}
	return list
}

func (cm *ClusterManager) GetCluster(id string) (*ClusterConnection, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	conn, exists := cm.clusters[id]
	if !exists {
		return nil, false
	}
	cp := *conn
	return &cp, true
}

func (cm *ClusterManager) GetClient(id string) (kubernetes.Interface, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if id == "" || id == "active" || id == "current" {
		id = cm.activeID
	}

	client, exists := cm.clientsets[id]
	if !exists || client == nil {
		return nil, fmt.Errorf("kubernetes client not found for cluster id '%s'", id)
	}
	return client, nil
}

func (cm *ClusterManager) GetActiveClusterID() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.activeID
}

func (cm *ClusterManager) SetActiveCluster(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.clusters[id]; !exists {
		return fmt.Errorf("cluster '%s' not registered", id)
	}
	cm.activeID = id
	return nil
}

func (cm *ClusterManager) UpdateStatus(id string, status ClusterStatus, version string, nodes, namespaces int, latency int64) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.clusters[id]; exists {
		conn.Status = status
		if version != "" {
			conn.KubernetesVersion = version
		}
		conn.NodeCount = nodes
		conn.NamespaceCount = namespaces
		conn.LatencyMs = latency
		conn.LastHeartbeat = time.Now()
		conn.UpdatedAt = time.Now()
	}
}

func (cm *ClusterManager) DeleteCluster(id string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if conn, exists := cm.clusters[id]; exists {
		if conn.ConnectionMode == ModeLocalKubeconfig && conn.ID == "local-dev" {
			return fmt.Errorf("cannot delete default local development cluster connection")
		}
		delete(cm.clusters, id)
		delete(cm.clientsets, id)
		if cm.activeID == id {
			cm.activeID = ""
			for remainingID := range cm.clusters {
				cm.activeID = remainingID
				break
			}
		}
		return nil
	}
	return fmt.Errorf("cluster '%s' not found", id)
}

func AutoDiscoverLocalContexts(kubeconfigPath string) error {
	cm := GetManager()

	if kubeconfigPath == "" {
		if path := os.Getenv("KUBECONFIG"); path != "" {
			kubeconfigPath = path
		} else {
			home, err := os.UserHomeDir()
			if err == nil {
				kubeconfigPath = filepath.Join(home, ".kube", "config")
			}
		}
	}

	if kubeconfigPath == "" {
		return fmt.Errorf("no kubeconfig path specified or found")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		// Create mock/standalone local fallback if config file is absent
		return fmt.Errorf("build config from kubeconfig %s: %w", kubeconfigPath, err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create clientset: %w", err)
	}

	// Ping cluster version to assess status
	versionInfo, err := clientset.Discovery().ServerVersion()
	status := StatusConnected
	version := "1.32.0"
	if err != nil {
		status = StatusDegraded
	} else {
		version = versionInfo.GitVersion
	}

	nodeCount := 0
	namespaceCount := 0
	if status == StatusConnected {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		if nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err == nil {
			nodeCount = len(nodes.Items)
		}
		if nss, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}); err == nil {
			namespaceCount = len(nss.Items)
		}
	}

	conn := &ClusterConnection{
		ID:                "local-dev",
		Name:              "Local Development",
		Environment:       "development",
		Provider:          "Local / Kubeconfig",
		ClusterType:       "Local Context",
		ConnectionMode:    ModeLocalKubeconfig,
		Status:            status,
		Endpoint:          config.Host,
		KubernetesVersion: version,
		NodeCount:         nodeCount,
		NamespaceCount:    namespaceCount,
		LatencyMs:         12,
		Capabilities: CapabilitySet{
			CanReadWorkloads:    true,
			CanReadLogs:         true,
			CanReadEvents:       true,
			CanReadTelemetry:    true,
			CanOperateWorkloads: true,
			CanAdminister:       true,
		},
		KubeconfigPath: kubeconfigPath,
	}

	cm.RegisterCluster(conn, clientset)
	return nil
}

// BuildClientFromToken creates a kubernetes.Interface using a scoped ServiceAccount token and host API server URL.
func BuildClientFromToken(apiServerHost string, bearerToken string, caData []byte) (kubernetes.Interface, error) {
	restConfig := &rest.Config{
		Host:        apiServerHost,
		BearerToken: bearerToken,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
			Insecure: len(caData) == 0,
		},
	}
	return kubernetes.NewForConfig(restConfig)
}
