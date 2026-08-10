package analyzer

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type Finding struct {
	Severity string `json:"severity"`
	Reason   string `json:"reason"`
	Message  string `json:"message"`
	Evidence string `json:"evidence"`
}

func AnalyzePod(
	pod *corev1.Pod,
	events []corev1.Event,
) []Finding {

	findings := make([]Finding, 0)

	// Pod phase
	if pod.Status.Phase == corev1.PodFailed {
		findings = append(findings, Finding{
			Severity: "critical",
			Reason:   "PodFailed",
			Message:  "Pod is in Failed state",
			Evidence: string(pod.Status.Phase),
		})
	}

	if pod.Status.Phase == corev1.PodPending {
		findings = append(findings, Finding{
			Severity: "warning",
			Reason:   "PodPending",
			Message:  "Pod is still pending",
			Evidence: string(pod.Status.Phase),
		})
	}

	// Container state
	for _, container := range pod.Status.ContainerStatuses {

		if container.RestartCount > 0 {
			findings = append(findings, Finding{
				Severity: "warning",
				Reason:   "ContainerRestarts",
				Message:  "Container has restarted",
				Evidence: container.Name,
			})
		}

		if container.State.Waiting != nil {

			reason := container.State.Waiting.Reason

			switch reason {

			case "CrashLoopBackOff":
				findings = append(findings, Finding{
					Severity: "critical",
					Reason:   "CrashLoopBackOff",
					Message:  "Container repeatedly crashes after starting",
					Evidence: container.Name,
				})

			case "ImagePullBackOff":
				findings = append(findings, Finding{
					Severity: "critical",
					Reason:   "ImagePullBackOff",
					Message:  "Kubernetes cannot pull the container image",
					Evidence: container.Name,
				})

			case "ErrImagePull":
				findings = append(findings, Finding{
					Severity: "critical",
					Reason:   "ErrImagePull",
					Message:  "Container image pull failed",
					Evidence: container.Name,
				})

			case "CreateContainerConfigError":
				findings = append(findings, Finding{
					Severity: "critical",
					Reason:   "CreateContainerConfigError",
					Message:  "Container configuration could not be created",
					Evidence: container.Name,
				})
			}
		}
	}

	// Pod conditions
	for _, condition := range pod.Status.Conditions {

		if condition.Type == corev1.PodReady &&
			condition.Status != corev1.ConditionTrue {

			findings = append(findings, Finding{
				Severity: "warning",
				Reason:   "PodNotReady",
				Message:  "Pod readiness condition is not healthy",
				Evidence: condition.Message,
			})
		}

		if condition.Type == corev1.PodScheduled &&
			condition.Status == corev1.ConditionFalse {

			findings = append(findings, Finding{
				Severity: "critical",
				Reason:   "PodUnschedulable",
				Message:  "Scheduler could not place the pod on a node",
				Evidence: condition.Message,
			})
		}
	}

	// Kubernetes warning events
	for _, event := range events {

		if event.Type != corev1.EventTypeWarning {
			continue
		}

		findings = append(findings, Finding{
			Severity: "warning",
			Reason:   event.Reason,
			Message:  event.Message,
			Evidence: "Kubernetes Event",
		})
	}

	return findings
}

func AnalyzeDeployment(
	deployment *appsv1.Deployment,
	pods []corev1.Pod,
	events []corev1.Event,
) []Finding {

	findings := make([]Finding, 0)

	desired := int32(0)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}

	if deployment.Status.AvailableReplicas < desired {
		findings = append(findings, Finding{
			Severity: "critical",
			Reason:   "DeploymentUnavailable",
			Message:  "Deployment has fewer available replicas than desired",
			Evidence: fmt.Sprintf(
				"desired=%d available=%d",
				desired,
				deployment.Status.AvailableReplicas,
			),
		})
	}

	if deployment.Status.ReadyReplicas < desired {
		findings = append(findings, Finding{
			Severity: "warning",
			Reason:   "InsufficientReadyReplicas",
			Message:  "Not all desired replicas are ready",
			Evidence: fmt.Sprintf(
				"desired=%d ready=%d",
				desired,
				deployment.Status.ReadyReplicas,
			),
		})
	}

	for i := range pods {
		podFindings := AnalyzePod(
			&pods[i],
			events,
		)

		for _, finding := range podFindings {
			finding.Evidence =
				pods[i].Name + ": " + finding.Evidence

			findings = append(
				findings,
				finding,
			)
		}
	}

	return findings
}
