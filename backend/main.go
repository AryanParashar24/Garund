package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"garund-backend/actions"
	analyzer "garund-backend/analyzer"
	utils "garund-backend/utils"
	watcher "garund-backend/watcher"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/gorilla/websocket"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

type SearchResult struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Status    string `json:"status,omitempty"`
	UID       string `json:"uid,omitempty"`
	Score     int    `json:"score,omitempty"`
}

type searchResourceType struct {
	kind  string
	score int
}

type ReliabilityMetric struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Value       *float64 `json:"value"`
	Target      float64  `json:"target"`
	Unit        string   `json:"unit"`
	GoodEvents  int64    `json:"goodEvents"`
	TotalEvents int64    `json:"totalEvents"`
	Window      string   `json:"window"`
	Status      string   `json:"status"`
}

type ReliabilitySLO struct {
	Name                 string   `json:"name"`
	Service              string   `json:"service"`
	Namespace            string   `json:"namespace"`
	Target               float64  `json:"target"`
	Window               string   `json:"window"`
	SLIType              string   `json:"sliType"`
	Current              *float64 `json:"current"`
	ErrorBudget          float64  `json:"errorBudget"`
	ErrorBudgetRemaining float64  `json:"errorBudgetRemaining"`
	Status               string   `json:"status"`
}

type ReliabilitySLA struct {
	Name               string   `json:"name"`
	Service            string   `json:"service"`
	Namespace          string   `json:"namespace"`
	AvailabilityTarget *float64 `json:"availabilityTarget,omitempty"`
	LatencyTarget      *float64 `json:"latencyTarget,omitempty"`
	Window             string   `json:"window"`
	Status             string   `json:"status"`
}

var searchAliases = map[string]string{
	"po":   "Pod",
	"pod":  "Pod",
	"pods": "Pod",

	"svc":      "Service",
	"service":  "Service",
	"services": "Service",

	"dep":         "Deployment",
	"deploy":      "Deployment",
	"deployment":  "Deployment",
	"deployments": "Deployment",

	"rs":          "ReplicaSet",
	"replicaset":  "ReplicaSet",
	"replicasets": "ReplicaSet",

	"ns":         "Namespace",
	"namespace":  "Namespace",
	"namespaces": "Namespace",

	"no":    "Node",
	"node":  "Node",
	"nodes": "Node",

	"sts":          "StatefulSet",
	"statefulset":  "StatefulSet",
	"statefulsets": "StatefulSet",

	"ds":         "DaemonSet",
	"daemonset":  "DaemonSet",
	"daemonsets": "DaemonSet",
}

func reliabilityStatus(
	current float64,
	target float64,
) string {

	if current >= target {
		return "healthy"
	}

	difference := target - current

	if difference <= 0.5 {
		return "warning"
	}

	return "critical"
}

func calculateErrorBudget(
	target float64,
	current float64,
) float64 {

	if target <= 0 {
		return 0
	}

	if current >= target {
		return 100
	}

	return ((current) / target) * 100
}

func normalizeSearchQuery(query string) string {
	query = strings.TrimSpace(
		strings.ToLower(query),
	)

	query = strings.TrimPrefix(
		query,
		"/",
	)

	return strings.TrimSpace(query)
}

func parseSearchQuery(query string) (string, string) {
	query = normalizeSearchQuery(query)

	if query == "" {
		return "", ""
	}

	parts := strings.Fields(query)

	if len(parts) == 1 {
		return "", parts[0]
	}

	resourceType := searchAliases[parts[0]]

	if resourceType == "" {
		return "", query
	}

	return resourceType, strings.Join(parts[1:], " ")
}

func searchMatches(
	name string,
	namespace string,
	kind string,
	query string,
) bool {

	query = strings.ToLower(
		strings.TrimSpace(query),
	)

	if query == "" {
		return true
	}

	name = strings.ToLower(name)
	namespace = strings.ToLower(namespace)
	kind = strings.ToLower(kind)

	return strings.Contains(name, query) ||
		strings.Contains(namespace, query) ||
		strings.Contains(kind, query)
}

func searchScore(
	name string,
	namespace string,
	kind string,
	query string,
) int {

	query = strings.ToLower(
		strings.TrimSpace(query),
	)

	name = strings.ToLower(name)
	namespace = strings.ToLower(namespace)
	kind = strings.ToLower(kind)

	if query == "" {
		return 1
	}

	if name == query {
		return 100
	}

	if strings.HasPrefix(name, query) {
		return 80
	}

	if strings.Contains(name, query) {
		return 60
	}

	if strings.Contains(namespace, query) {
		return 40
	}

	if strings.Contains(kind, query) {
		return 20
	}

	return 10
}

func podHealth(
	pod corev1.Pod,
) string {

	switch pod.Status.Phase {

	case corev1.PodRunning:

		for _, container := range pod.Status.ContainerStatuses {

			if !container.Ready {
				return "warning"
			}

			if container.RestartCount >= 5 {
				return "warning"
			}
		}

		return "healthy"

	case corev1.PodPending:
		return "warning"

	case corev1.PodFailed:
		return "critical"

	case corev1.PodUnknown:
		return "critical"

	default:
		return "warning"
	}
}

func deploymentHealth(
	deployment appsv1.Deployment,
) string {

	desired := int32(0)

	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	if deployment.Status.UnavailableReplicas > 0 {
		return "critical"
	}

	if deployment.Status.ReadyReplicas < desired {
		return "warning"
	}

	return "healthy"
}

func replicaSetHealth(
	rs appsv1.ReplicaSet,
) string {

	desired := int32(0)

	if rs.Spec.Replicas != nil {
		desired = *rs.Spec.Replicas
	}

	if rs.Status.ReadyReplicas < desired {
		return "warning"
	}

	return "healthy"
}

func nodeHealth(
	node corev1.Node,
) string {

	for _, condition := range node.Status.Conditions {

		if condition.Type ==
			corev1.NodeReady {

			if condition.Status ==
				corev1.ConditionTrue {

				return "healthy"
			}

			return "critical"
		}
	}

	return "warning"
}

func namespaceHealth(
	namespace corev1.Namespace,
) string {

	if namespace.Status.Phase ==
		corev1.NamespaceTerminating {

		return "warning"
	}

	return "healthy"
}

func statefulSetHealth(
	sts appsv1.StatefulSet,
) string {

	desired := int32(0)

	if sts.Spec.Replicas != nil {
		desired = *sts.Spec.Replicas
	}

	if sts.Status.ReadyReplicas < desired {
		return "warning"
	}

	return "healthy"
}

func daemonSetHealth(
	ds appsv1.DaemonSet,
) string {

	if ds.Status.NumberUnavailable > 0 {
		return "warning"
	}

	if ds.Status.DesiredNumberScheduled !=
		ds.Status.NumberReady {

		return "warning"
	}

	return "healthy"
}

func healthMatches(
	health string,
	status string,
	query string,
) bool {

	query = strings.ToLower(
		strings.TrimSpace(query),
	)

	switch query {

	case "healthy":
		return health == "healthy"

	case "warning":
		return health == "warning"

	case "critical":
		return health == "critical"

	case "running":
		return strings.EqualFold(
			status,
			"Running",
		)

	case "pending":
		return strings.EqualFold(
			status,
			"Pending",
		)

	case "failed":
		return strings.EqualFold(
			status,
			"Failed",
		)

	default:
		return false
	}
}

func addSearchResult(
	results *[]gin.H,
	kind string,
	name string,
	namespace string,
	status string,
	health string,
	uid string,
	query string,
) {

	*results = append(
		*results,
		gin.H{
			"kind":      kind,
			"name":      name,
			"namespace": namespace,
			"status":    status,
			"health":    health,
			"uid":       uid,
			"score": searchScore(
				name,
				namespace,
				kind,
				query,
			),
		},
	)
}

func InitMeterProvider(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	exporter, err :=
		otlpmetricgrpc.New(
			ctx,
			otlpmetricgrpc.WithEndpoint(
				"localhost:4317",
			),
			otlpmetricgrpc.WithInsecure(),
		)

	if err != nil {
		return nil, err
	}

	provider :=
		sdkmetric.NewMeterProvider(
			sdkmetric.WithReader(
				sdkmetric.NewPeriodicReader(
					exporter,
				),
			),
		)

	otel.SetMeterProvider(provider)

	return provider, nil
}

func podIsHealthy(pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	if len(pod.Status.ContainerStatuses) == 0 {
		return false
	}

	for _, container := range pod.Status.ContainerStatuses {
		if !container.Ready {
			return false
		}
	}

	return true
}

func deploymentIsHealthy(
	deployment appsv1.Deployment,
) bool {
	desired := int32(0)

	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	return deployment.Status.ReadyReplicas == desired &&
		deployment.Status.AvailableReplicas == desired &&
		deployment.Status.UnavailableReplicas == 0
}

func replicaSetIsHealthy(
	replicaSet appsv1.ReplicaSet,
) bool {
	desired := int32(0)

	if replicaSet.Spec.Replicas != nil {
		desired = *replicaSet.Spec.Replicas
	}

	return replicaSet.Status.ReadyReplicas == desired &&
		replicaSet.Status.AvailableReplicas >= desired
}

func namespaceIsHealthy(
	namespace corev1.Namespace,
) bool {
	return namespace.Status.Phase ==
		corev1.NamespaceActive
}

func serviceIsHealthy(
	service corev1.Service,
	endpoints corev1.Endpoints,
) bool {

	// ExternalName services don't have Kubernetes endpoints.
	// We can at least verify that the service has an
	// external name configured.
	if service.Spec.Type ==
		corev1.ServiceTypeExternalName {

		return service.Spec.ExternalName != ""
	}

	for _, subset := range endpoints.Subsets {

		if len(subset.Addresses) > 0 {
			return true
		}
	}

	return false
}

func main() {
	router := gin.Default()
	router.Use(cors.Default())

	router.Use(
		otelgin.Middleware("garund"),
	)

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err.Error())
	}

	kubeconfig := filepath.Join(
		home,
		".kube",
		"config",
	)

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	prometheusURL := os.Getenv("PROMETHEUS_URL")

	if prometheusURL == "" {
		prometheusURL = "http://localhost:9090"
	}

	prometheusClient := NewPrometheusClient(
		prometheusURL,
	)

	go watcher.WatchPods(clientset)
	go watcher.WatchDeployments(clientset)
	go watcher.WatchEvents(clientset)
	go watcher.WatchServices(clientset)
	var upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	err = InitTracer()
	if err != nil {
		panic(err)
	}

	meterProvider, err :=
		InitMeterProvider(
			context.Background(),
		)

	if err != nil {
		panic(err)
	}

	defer func() {
		ctx, cancel :=
			context.WithTimeout(
				context.Background(),
				5*time.Second,
			)

		defer cancel()

		if err :=
			meterProvider.Shutdown(ctx); err != nil {
			println(
				"metric provider shutdown:",
				err.Error(),
			)
		}
	}()

	registerReliabilityRoutes(
		router,
		clientset,
		prometheusClient,
	)

	router.GET("/pods", func(c *gin.Context) {

		namespace := c.Query("namespace")
		tracer := otel.Tracer("garund")

		ctx, span := tracer.Start(
			context.Background(),
			"list-pods",
		)

		defer span.End()

		pods, err := clientset.CoreV1().
			Pods(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		var podData []gin.H

		for _, pod := range pods.Items {

			podData = append(podData, gin.H{
				"name":      pod.Name,
				"namespace": pod.Namespace,
				"status":    pod.Status.Phase,
				"node":      pod.Spec.NodeName,
			})
		}

		c.JSON(http.StatusOK, podData)
	})

	router.GET("/namespaces", func(c *gin.Context) {

		namespaces, err := clientset.CoreV1().
			Namespaces().
			List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		var namespaceData []gin.H

		for _, ns := range namespaces.Items {

			namespaceData = append(namespaceData, gin.H{
				"name":   ns.Name,
				"status": ns.Status.Phase,
			})
		}

		c.JSON(http.StatusOK, namespaceData)
	})

	router.GET("/deployments", func(c *gin.Context) {

		namespace := c.Query("namespace")
		deployments, err := clientset.AppsV1().
			Deployments(namespace).
			List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		var deploymentData []gin.H

		for _, deploy := range deployments.Items {

			replicas := int32(0)

			if deploy.Spec.Replicas != nil {
				replicas = *deploy.Spec.Replicas
			}

			deploymentData = append(deploymentData, gin.H{
				"name":      deploy.Name,
				"namespace": deploy.Namespace,
				"replicas":  replicas,
			})
		}

		c.JSON(http.StatusOK, deploymentData)
	})

	router.GET("/overview", func(c *gin.Context) {

		tracer := otel.Tracer("garund")

		ctx, span := tracer.Start(
			context.Background(),
			"list-overview",
		)

		defer span.End()

		namespace := c.Query("namespace")

		// --------------------------------------------------
		// Pods
		// --------------------------------------------------

		pods, err := clientset.CoreV1().
			Pods(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// Deployments
		// --------------------------------------------------

		deployments, err := clientset.AppsV1().
			Deployments(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// ReplicaSets
		// --------------------------------------------------

		replicaSets, err := clientset.AppsV1().
			ReplicaSets(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// Nodes
		// --------------------------------------------------

		nodes, err := clientset.CoreV1().
			Nodes().
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// Services
		// --------------------------------------------------

		services, err := clientset.CoreV1().
			Services(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// Events
		// --------------------------------------------------

		events, err := clientset.CoreV1().
			Events(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// Namespaces
		// --------------------------------------------------

		namespaces, err := clientset.CoreV1().
			Namespaces().
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		// --------------------------------------------------
		// POD HEALTH
		// --------------------------------------------------

		podHealthy := 0

		for _, pod := range pods.Items {

			if podIsHealthy(pod) {
				podHealthy++
			}
		}

		podUnhealthy :=
			len(pods.Items) - podHealthy

		// --------------------------------------------------
		// DEPLOYMENT HEALTH
		// --------------------------------------------------

		deploymentHealthy := 0

		for _, deployment := range deployments.Items {

			if deploymentIsHealthy(deployment) {
				deploymentHealthy++
			}
		}

		deploymentUnhealthy :=
			len(deployments.Items) - deploymentHealthy

		// --------------------------------------------------
		// REPLICASET HEALTH
		// --------------------------------------------------

		replicaSetHealthy := 0

		for _, replicaSet := range replicaSets.Items {

			if replicaSetIsHealthy(replicaSet) {
				replicaSetHealthy++
			}
		}

		replicaSetUnhealthy :=
			len(replicaSets.Items) - replicaSetHealthy

		// --------------------------------------------------
		// NAMESPACE HEALTH
		// --------------------------------------------------

		namespaceHealthy := 0

		namespaceCount := 0

		for _, ns := range namespaces.Items {

			// If a namespace is selected,
			// only evaluate that namespace.
			if namespace != "" &&
				ns.Name != namespace {
				continue
			}

			namespaceCount++

			if namespaceIsHealthy(ns) {
				namespaceHealthy++
			}
		}

		namespaceUnhealthy :=
			namespaceCount - namespaceHealthy

		// --------------------------------------------------
		// SERVICE HEALTH
		// --------------------------------------------------

		endpoints, err := clientset.CoreV1().
			Endpoints(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		endpointMap :=
			make(map[string]corev1.Endpoints)

		for _, endpoint := range endpoints.Items {

			key :=
				endpoint.Namespace +
					"/" +
					endpoint.Name

			endpointMap[key] = endpoint
		}

		serviceHealthy := 0

		for _, service := range services.Items {

			key :=
				service.Namespace +
					"/" +
					service.Name

			endpoint :=
				endpointMap[key]

			if serviceIsHealthy(
				service,
				endpoint,
			) {
				serviceHealthy++
			}
		}

		serviceUnhealthy :=
			len(services.Items) - serviceHealthy

		// --------------------------------------------------
		// RESPONSE
		// --------------------------------------------------

		c.JSON(http.StatusOK, gin.H{

			"pods": len(pods.Items),

			"namespaces": namespaceCount,

			"deployments": len(deployments.Items),

			"replicaSets": len(replicaSets.Items),

			"nodes": len(nodes.Items),

			"services": len(services.Items),

			"events": len(events.Items),

			"health": gin.H{

				"pods": gin.H{
					"healthy":   podHealthy,
					"unhealthy": podUnhealthy,
					"total":     len(pods.Items),
				},

				"namespaces": gin.H{
					"healthy":   namespaceHealthy,
					"unhealthy": namespaceUnhealthy,
					"total":     namespaceCount,
				},

				"deployments": gin.H{
					"healthy":   deploymentHealthy,
					"unhealthy": deploymentUnhealthy,
					"total":     len(deployments.Items),
				},

				"replicaSets": gin.H{
					"healthy":   replicaSetHealthy,
					"unhealthy": replicaSetUnhealthy,
					"total":     len(replicaSets.Items),
				},

				"services": gin.H{
					"healthy":   serviceHealthy,
					"unhealthy": serviceUnhealthy,
					"total":     len(services.Items),
				},
			},
		})
	})

	router.GET("/services", func(c *gin.Context) {

		namespace := c.Query("namespace")
		tracer := otel.Tracer("garund")

		ctx, span := tracer.Start(
			context.Background(),
			"list-services",
		)

		defer span.End()
		services, err := clientset.CoreV1().
			Services(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		var data []gin.H

		for _, svc := range services.Items {

			data = append(data, gin.H{
				"name":      svc.Name,
				"namespace": svc.Namespace,
				"type":      string(svc.Spec.Type),
			})
		}
		c.JSON(http.StatusOK, data)
	})

	router.GET("/replicasets", func(c *gin.Context) {
		namespace := c.Query("namespace")

		replicaSets, err := clientset.AppsV1().
			ReplicaSets(namespace).
			List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		data := make([]gin.H, 0)

		for _, rs := range replicaSets.Items {
			data = append(data, gin.H{
				"name":      rs.Name,
				"namespace": rs.Namespace,
				"replicas":  rs.Status.Replicas,
				"ready":     rs.Status.ReadyReplicas,
				"available": rs.Status.AvailableReplicas,
			})
		}

		c.JSON(http.StatusOK, data)
	})

	router.GET("/nodes", func(c *gin.Context) {

		nodes, err := clientset.CoreV1().
			Nodes().
			List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		var data []gin.H

		for _, node := range nodes.Items {

			status := "Unknown"

			for _, condition := range node.Status.Conditions {

				if condition.Type == "Ready" {
					status = string(condition.Status)
				}
			}

			data = append(data, gin.H{
				"name":   node.Name,
				"status": status,
			})
		}

		c.JSON(http.StatusOK, data)
	})

	router.GET("/events", func(c *gin.Context) {
		tracer := otel.Tracer("garund")

		ctx, span := tracer.Start(
			context.Background(),
			"list-events",
		)
		defer span.End()

		namespace := c.Query("namespace")

		events, err := clientset.CoreV1().
			Events(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		data := make([]gin.H, 0, len(events.Items))

		for _, event := range events.Items {
			eventTime := event.EventTime.Time

			// Older Kubernetes events may not have EventTime populated.
			if eventTime.IsZero() {
				eventTime = event.LastTimestamp.Time
			}

			if eventTime.IsZero() {
				eventTime = event.FirstTimestamp.Time
			}

			data = append(data, gin.H{
				"type":      event.Type,
				"reason":    event.Reason,
				"namespace": event.Namespace,
				"message":   event.Message,

				"name":  event.Name,
				"count": event.Count,

				"firstSeen": event.FirstTimestamp,
				"lastSeen":  event.LastTimestamp,
				"eventTime": eventTime,

				"source": gin.H{
					"component": event.Source.Component,
					"host":      event.Source.Host,
				},

				"involvedObject": gin.H{
					"kind":       event.InvolvedObject.Kind,
					"name":       event.InvolvedObject.Name,
					"namespace":  event.InvolvedObject.Namespace,
					"uid":        event.InvolvedObject.UID,
					"apiVersion": event.InvolvedObject.APIVersion,
				},
			})
		}

		c.JSON(http.StatusOK, data)
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	router.GET("/topology", func(c *gin.Context) {

		tracer := otel.Tracer("garund")

		ctx, span := tracer.Start(
			context.Background(),
			"list-topology",
		)

		defer span.End()

		namespace := c.Query("namespace")

		services, err := clientset.CoreV1().
			Services(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		deployments, err := clientset.AppsV1().
			Deployments(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		replicaSets, err := clientset.AppsV1().
			ReplicaSets(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		pods, err := clientset.CoreV1().
			Pods(namespace).
			List(ctx, metav1.ListOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		nodes := make([]gin.H, 0)
		edges := make([]gin.H, 0)

		for i, svc := range services.Items {

			matchedPods := utils.MatchServiceToPods(
				svc,
				pods.Items,
			)

			nodes = append(nodes, gin.H{
				"id":   "service-" + svc.Namespace + "-" + svc.Name,
				"type": "service",

				"position": gin.H{
					"x": 100,
					"y": i * 120,
				},

				"data": gin.H{
					"label":     svc.Name,
					"name":      svc.Name,
					"kind":      "Service",
					"namespace": svc.Namespace,

					"svcType":   string(svc.Spec.Type),
					"clusterIP": svc.Spec.ClusterIP,
					"selector":  svc.Spec.Selector,

					"backends": len(matchedPods),

					"health": func() string {
						endpoints, err := clientset.CoreV1().
							Endpoints(svc.Namespace).
							Get(
								ctx,
								svc.Name,
								metav1.GetOptions{},
							)

						if err != nil {
							return "critical"
						}

						if serviceIsHealthy(svc, *endpoints) {
							return "healthy"
						}

						return "critical"
					}(),
				},
			})
		}

		for i, dep := range deployments.Items {

			replicas := int32(0)
			if dep.Spec.Replicas != nil {
				replicas = *dep.Spec.Replicas
			}

			nodes = append(nodes, gin.H{
				"id":   "deployment-" + dep.Namespace + "-" + dep.Name,
				"type": "deployment",

				"position": gin.H{
					"x": 400,
					"y": i * 120,
				},

				"data": gin.H{
					"label":               dep.Name,
					"name":                dep.Name,
					"kind":                "Deployment",
					"namespace":           dep.Namespace,
					"replicas":            replicas,
					"readyReplicas":       dep.Status.ReadyReplicas,
					"availableReplicas":   dep.Status.AvailableReplicas,
					"updatedReplicas":     dep.Status.UpdatedReplicas,
					"unavailableReplicas": dep.Status.UnavailableReplicas,
					"observedGeneration":  dep.Status.ObservedGeneration,
					"generation":          dep.Generation,
					"health": func() string {
						if deploymentIsHealthy(dep) {
							return "healthy"
						}

						return "critical"
					}(),
				},
			})
		}

		for i, rs := range replicaSets.Items {

			nodes = append(nodes, gin.H{
				"id":   "replicaset-" + rs.Namespace + "-" + rs.Name,
				"type": "replicaset",

				"position": gin.H{
					"x": 700,
					"y": i * 120,
				},

				"data": gin.H{
					"label":         rs.Name,
					"name":          rs.Name,
					"kind":          "ReplicaSet",
					"namespace":     rs.Namespace,
					"replicas":      rs.Status.Replicas,
					"readyReplicas": rs.Status.ReadyReplicas,
					"health": func() string {
						if replicaSetIsHealthy(rs) {
							return "healthy"
						}

						return "critical"
					}(),
				},
			})
		}

		for _, rs := range replicaSets.Items {

			dep := utils.FindDeploymentForReplicaSet(
				rs,
				deployments.Items,
			)

			if dep == nil {
				continue
			}

			edges = append(edges, gin.H{
				"id": "deployment-rs-" + dep.Namespace + "-" + dep.Name + "-" + rs.Name,

				"source": "deployment-" + dep.Namespace + "-" + dep.Name,
				"target": "replicaset-" + rs.Namespace + "-" + rs.Name,

				"animated": true,
				"type":     "smoothstep",
				"style": gin.H{
					"stroke":      "#3b82f6",
					"strokeWidth": 2,
				},
			})
		}

		for _, svc := range services.Items {

			matchedPods := utils.MatchServiceToPods(
				svc,
				pods.Items,
			)

			for _, pod := range matchedPods {

				edges = append(edges, gin.H{
					"id":       "service-pod-" + svc.Namespace + "-" + svc.Name + "-" + pod.Name,
					"source":   "service-" + svc.Namespace + "-" + svc.Name,
					"target":   "pod-" + pod.Namespace + "-" + pod.Name,
					"animated": true,
					"type":     "smoothstep",
					"backends": len(matchedPods),
					"style": gin.H{
						"stroke":      "#a855f7",
						"strokeWidth": 2,
					},
				})
			}
		}

		for i, pod := range pods.Items {

			restarts := int32(0)

			for _, container := range pod.Status.ContainerStatuses {
				restarts += container.RestartCount
			}

			health := podHealth(pod)

			nodes = append(nodes, gin.H{
				"id":   "pod-" + pod.Namespace + "-" + pod.Name,
				"type": "pod",

				"position": gin.H{
					"x": 1000,
					"y": i * 100,
				},

				"data": gin.H{
					"label":     pod.Name,
					"name":      pod.Name,
					"kind":      "Pod",
					"namespace": pod.Namespace,

					"status": string(pod.Status.Phase),
					"health": health,

					"node":     pod.Spec.NodeName,
					"podIP":    pod.Status.PodIP,
					"restarts": restarts,
					"age": time.Since(
						pod.CreationTimestamp.Time,
					).Round(time.Minute).String(),
				},
			})
		}

		for _, pod := range pods.Items {

			rsName := utils.ReplicaSetNameFromPod(pod)

			if rsName == "" {
				continue
			}

			color := "#22c55e"

			if pod.Status.Phase != corev1.PodRunning {
				color = "#ef4444"
			}

			edges = append(edges, gin.H{
				"id": "rs-pod-" + pod.Namespace + "-" + rsName + "-" + pod.Name,

				"source": "replicaset-" + pod.Namespace + "-" + rsName,
				"target": "pod-" + pod.Namespace + "-" + pod.Name,

				"animated": true,
				"type":     "smoothstep",

				"style": gin.H{
					"stroke":      color,
					"strokeWidth": 2,
				},
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"nodes": nodes,
			"edges": edges,

			"stats": gin.H{
				"services":    len(services.Items),
				"deployments": len(deployments.Items),
				"replicaSets": len(replicaSets.Items),
				"pods":        len(pods.Items),
				"edges":       len(edges),
			},
		})
	})

	router.GET("/search", func(c *gin.Context) {

		query := normalizeSearchQuery(
			c.Query("q"),
		)

		resourceType, searchTerm :=
			parseSearchQuery(query)

		/*
		* Special status searches:
		*
		* /running
		* /pending
		* /failed
		* /warning
		* /critical
		 */

		statusSearch := map[string]bool{
			"running":  true,
			"pending":  true,
			"failed":   true,
			"warning":  true,
			"critical": true,
		}

		isStatusSearch :=
			statusSearch[searchTerm]

		results := make([]gin.H, 0)

		/*
		* --------------------------------------------------
		* PODS
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "Pod" {

			pods, err :=
				clientset.CoreV1().
					Pods("").
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, pod := range pods.Items {

					health :=
						podHealth(pod)

					status :=
						string(
							pod.Status.Phase,
						)

					if isStatusSearch &&
						!healthMatches(
							health,
							status,
							searchTerm,
						) {

						continue
					}

					if !isStatusSearch &&
						!searchMatches(
							pod.Name,
							pod.Namespace,
							"Pod",
							searchTerm,
						) {

						continue
					}

					addSearchResult(
						&results,
						"Pod",
						pod.Name,
						pod.Namespace,
						status,
						health,
						string(pod.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* --------------------------------------------------
		* SERVICES
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "Service" {

			services, err :=
				clientset.CoreV1().
					Services("").
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, svc := range services.Items {

					if isStatusSearch {
						continue
					}

					if !searchMatches(
						svc.Name,
						svc.Namespace,
						"Service",
						searchTerm,
					) {
						continue
					}

					addSearchResult(
						&results,
						"Service",
						svc.Name,
						svc.Namespace,
						string(svc.Spec.Type),
						"healthy",
						string(svc.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* --------------------------------------------------
		* DEPLOYMENTS
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "Deployment" {

			deployments, err :=
				clientset.AppsV1().
					Deployments("").
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, deployment := range deployments.Items {

					health :=
						deploymentHealth(
							deployment,
						)

					status :=
						fmt.Sprintf(
							"%d/%d ready",
							deployment.Status.ReadyReplicas,
							deployment.Spec.Replicas,
						)

					if isStatusSearch &&
						!healthMatches(
							health,
							status,
							searchTerm,
						) {

						continue
					}

					if !isStatusSearch &&
						!searchMatches(
							deployment.Name,
							deployment.Namespace,
							"Deployment",
							searchTerm,
						) {

						continue
					}

					addSearchResult(
						&results,
						"Deployment",
						deployment.Name,
						deployment.Namespace,
						status,
						health,
						string(deployment.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* --------------------------------------------------
		* REPLICASETS
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "ReplicaSet" {

			replicaSets, err :=
				clientset.AppsV1().
					ReplicaSets("").
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, rs := range replicaSets.Items {

					health :=
						replicaSetHealth(rs)

					status :=
						fmt.Sprintf(
							"%d/%d ready",
							rs.Status.ReadyReplicas,
							rs.Spec.Replicas,
						)

					if isStatusSearch &&
						!healthMatches(
							health,
							status,
							searchTerm,
						) {

						continue
					}

					if !isStatusSearch &&
						!searchMatches(
							rs.Name,
							rs.Namespace,
							"ReplicaSet",
							searchTerm,
						) {

						continue
					}

					addSearchResult(
						&results,
						"ReplicaSet",
						rs.Name,
						rs.Namespace,
						status,
						health,
						string(rs.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* --------------------------------------------------
		* NAMESPACES
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "Namespace" {

			namespaces, err :=
				clientset.CoreV1().
					Namespaces().
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, namespace := range namespaces.Items {

					if isStatusSearch {
						continue
					}

					if !searchMatches(
						namespace.Name,
						"",
						"Namespace",
						searchTerm,
					) {
						continue
					}

					health :=
						namespaceHealth(
							namespace,
						)

					addSearchResult(
						&results,
						"Namespace",
						namespace.Name,
						"",
						string(namespace.Status.Phase),
						health,
						string(namespace.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* --------------------------------------------------
		* NODES
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "Node" {

			nodes, err :=
				clientset.CoreV1().
					Nodes().
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, node := range nodes.Items {

					health :=
						nodeHealth(node)

					status := "Unknown"

					for _, condition := range node.Status.Conditions {

						if condition.Type ==
							corev1.NodeReady {

							if condition.Status ==
								corev1.ConditionTrue {

								status = "Ready"

							} else {

								status = "NotReady"
							}
						}
					}

					if isStatusSearch &&
						!healthMatches(
							health,
							status,
							searchTerm,
						) {

						continue
					}

					if !isStatusSearch &&
						!searchMatches(
							node.Name,
							"",
							"Node",
							searchTerm,
						) {

						continue
					}

					addSearchResult(
						&results,
						"Node",
						node.Name,
						"",
						status,
						health,
						string(node.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* --------------------------------------------------
		* STATEFULSETS
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "StatefulSet" {

			statefulSets, err :=
				clientset.AppsV1().
					StatefulSets("").
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, sts := range statefulSets.Items {

					health :=
						statefulSetHealth(sts)

					status :=
						fmt.Sprintf(
							"%d/%d ready",
							sts.Status.ReadyReplicas,
							sts.Status.Replicas,
						)

					if isStatusSearch &&
						!healthMatches(
							health,
							status,
							searchTerm,
						) {

						continue
					}

					if !isStatusSearch &&
						!searchMatches(
							sts.Name,
							sts.Namespace,
							"StatefulSet",
							searchTerm,
						) {

						continue
					}

					addSearchResult(
						&results,
						"StatefulSet",
						sts.Name,
						sts.Namespace,
						status,
						health,
						string(sts.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* --------------------------------------------------
		* DAEMONSETS
		* --------------------------------------------------
		 */

		if resourceType == "" ||
			resourceType == "DaemonSet" {

			daemonSets, err :=
				clientset.AppsV1().
					DaemonSets("").
					List(
						context.Background(),
						metav1.ListOptions{},
					)

			if err == nil {

				for _, ds := range daemonSets.Items {

					health :=
						daemonSetHealth(ds)

					status :=
						fmt.Sprintf(
							"%d/%d ready",
							ds.Status.NumberReady,
							ds.Status.DesiredNumberScheduled,
						)

					if isStatusSearch &&
						!healthMatches(
							health,
							status,
							searchTerm,
						) {

						continue
					}

					if !isStatusSearch &&
						!searchMatches(
							ds.Name,
							ds.Namespace,
							"DaemonSet",
							searchTerm,
						) {

						continue
					}

					addSearchResult(
						&results,
						"DaemonSet",
						ds.Name,
						ds.Namespace,
						status,
						health,
						string(ds.UID),
						searchTerm,
					)
				}
			}
		}

		/*
		* Sort best matches first.
		 */

		sort.SliceStable(
			results,
			func(i, j int) bool {

				left, _ :=
					results[i]["score"].(int)

				right, _ :=
					results[j]["score"].(int)

				return left > right
			},
		)

		/*
		* Limit result count so searching a large
		* cluster doesn't dump thousands of objects
		* into the browser.
		 */

		const maxResults = 100

		if len(results) > maxResults {
			results = results[:maxResults]
		}

		c.JSON(
			http.StatusOK,
			gin.H{
				"query":   query,
				"results": results,
				"count":   len(results),
			},
		)
	})

	router.GET("/health-score", func(c *gin.Context) {

		namespace := c.Query("namespace")

		events, _ := clientset.CoreV1().
			Events(namespace).
			List(context.TODO(), metav1.ListOptions{})

		score := 100

		for _, event := range events.Items {

			if event.Type == "Warning" {
				score -= 10
			}
		}

		if score < 0 {
			score = 0
		}

		c.JSON(http.StatusOK, gin.H{
			"score": score,
		})
	})

	router.GET("/namespace-list", func(c *gin.Context) {

		namespaces, err := clientset.CoreV1().
			Namespaces().
			List(context.TODO(), metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		data := make([]string, 0)

		for _, ns := range namespaces.Items {
			data = append(data, ns.Name)
		}

		c.JSON(http.StatusOK, data)
	})

	router.GET("/otel-status", func(c *gin.Context) {

		c.JSON(http.StatusOK, gin.H{
			"collector": "running",
			"grpc":      4317,
			"http":      4318,
		})
	})

	router.GET("/resource", func(c *gin.Context) {
		kind := c.Query("kind")
		namespace := c.Query("namespace")
		name := c.Query("name")

		if kind == "" || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "kind and name are required",
			})
			return
		}

		switch kind {

		case "Pod":
			resource, err := clientset.CoreV1().
				Pods(namespace).
				Get(c.Request.Context(), name, metav1.GetOptions{})

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, resource)

		case "Service":
			resource, err := clientset.CoreV1().
				Services(namespace).
				Get(c.Request.Context(), name, metav1.GetOptions{})

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, resource)

		case "Deployment":
			resource, err := clientset.AppsV1().
				Deployments(namespace).
				Get(c.Request.Context(), name, metav1.GetOptions{})

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, resource)

		case "ReplicaSet":
			resource, err := clientset.AppsV1().
				ReplicaSets(namespace).
				Get(c.Request.Context(), name, metav1.GetOptions{})

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, resource)

		default:
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "unsupported resource kind",
			})
		}
	})

	router.GET("/logs", func(c *gin.Context) {
		namespace := c.Query("namespace")
		podName := c.Query("pod")
		container := c.Query("container")

		if namespace == "" || podName == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace and pod are required",
			})
			return
		}

		options := &v1.PodLogOptions{
			TailLines: func() *int64 {
				n := int64(200)
				return &n
			}(),
		}

		if container != "" {
			options.Container = container
		}

		logs, err := clientset.CoreV1().
			Pods(namespace).
			GetLogs(podName, options).
			Do(c.Request.Context()).
			Raw()

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"pod":       podName,
			"namespace": namespace,
			"container": container,
			"logs":      string(logs),
		})
	})

	router.GET("/resource-events", func(c *gin.Context) {
		namespace := c.Query("namespace")
		name := c.Query("name")
		kind := c.Query("kind")

		if namespace == "" || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace and name are required",
			})
			return
		}

		selector := "involvedObject.name=" + name
		if kind != "" {
			selector += ",involvedObject.kind=" + kind
		}

		events, err := clientset.CoreV1().
			Events(namespace).
			List(
				c.Request.Context(),
				metav1.ListOptions{
					FieldSelector: selector,
				},
			)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		data := make([]gin.H, 0)

		for _, event := range events.Items {
			eventTime := event.EventTime.Time
			if eventTime.IsZero() {
				eventTime = event.LastTimestamp.Time
			}
			if eventTime.IsZero() {
				eventTime = event.FirstTimestamp.Time
			}
			data = append(data, gin.H{
				"uid":       event.UID,
				"type":      event.Type,
				"reason":    event.Reason,
				"message":   event.Message,
				"count":     event.Count,
				"namespace": event.Namespace,
				"eventTime": eventTime,
				"involvedObject": gin.H{
					"kind":      event.InvolvedObject.Kind,
					"name":      event.InvolvedObject.Name,
					"namespace": event.InvolvedObject.Namespace,
				},
			})
		}

		c.JSON(http.StatusOK, data)
	})

	router.GET("/analyze/pod", func(c *gin.Context) {

		namespace := c.Query("namespace")
		name := c.Query("name")

		if namespace == "" || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace and name are required",
			})
			return
		}

		ctx := c.Request.Context()

		pod, err := clientset.CoreV1().
			Pods(namespace).
			Get(ctx, name, metav1.GetOptions{})

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		events, err := clientset.CoreV1().
			Events(namespace).
			List(
				ctx,
				metav1.ListOptions{
					FieldSelector: "involvedObject.name=" + name,
				},
			)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		findings := analyzer.AnalyzePod(
			pod,
			events.Items,
		)

		healthy := len(findings) == 0

		RecordAnalysis(
			ctx,
			"pod",
			pod.Namespace,
			pod.Name,
			healthy,
			findings,
		)

		c.JSON(http.StatusOK, gin.H{
			"kind":      "Pod",
			"name":      pod.Name,
			"namespace": pod.Namespace,
			"healthy":   healthy,
			"findings":  findings,
		})
	})

	router.GET("/analyze/deployment", func(c *gin.Context) {

		namespace := c.Query("namespace")
		name := c.Query("name")

		if namespace == "" || name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace and name are required",
			})
			return
		}

		ctx := c.Request.Context()

		deployment, err := clientset.AppsV1().
			Deployments(namespace).
			Get(ctx, name, metav1.GetOptions{})

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": err.Error(),
			})
			return
		}

		selector, err := metav1.LabelSelectorAsSelector(
			deployment.Spec.Selector,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		pods, err := clientset.CoreV1().
			Pods(namespace).
			List(ctx, metav1.ListOptions{
				LabelSelector: selector.String(),
			})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		events, err := clientset.CoreV1().
			Events(namespace).
			List(ctx, metav1.ListOptions{})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		findings := analyzer.AnalyzeDeployment(
			deployment,
			pods.Items,
			events.Items,
		)

		healthy := len(findings) == 0

		RecordAnalysis(
			ctx,
			"deployment",
			deployment.Namespace,
			deployment.Name,
			healthy,
			findings,
		)

		c.JSON(http.StatusOK, gin.H{
			"kind":      "Deployment",
			"name":      deployment.Name,
			"namespace": deployment.Namespace,
			"healthy":   healthy,
			"pods":      len(pods.Items),
			"findings":  findings,
		})
	})

	router.GET("/ws", func(c *gin.Context) {

		conn, err := upgrader.Upgrade(
			c.Writer,
			c.Request,
			nil,
		)

		if err != nil {
			return
		}

		watcher.AddClient(conn)
		defer watcher.RemoveClient(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})

	router.POST("/pod/restart", func(c *gin.Context) {

		type Request struct {
			Namespace string `json:"namespace"`
			Name      string `json:"name"`
		}

		var req Request

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		err := actions.RestartPod(
			clientset,
			req.Namespace,
			req.Name,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Pod restarted successfully",
		})

	})

	router.DELETE("/pod", func(c *gin.Context) {

		namespace := c.Query("namespace")
		name := c.Query("name")

		if namespace == "" || name == "" {

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "namespace and name are required",
			})

			return
		}

		err := actions.DeletePod(
			clientset,
			namespace,
			name,
		)

		if err != nil {

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})

			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Pod deleted successfully",
		})

	})

	probeCtx, probeCancel := context.WithCancel(context.Background())
	defer probeCancel()

	if err := StartSyntheticReliabilityProbe(probeCtx); err != nil {
		fmt.Printf(
			"failed to start synthetic reliability probe: %v\n",
			err,
		)
	}
	router.Run(":8080")
}
