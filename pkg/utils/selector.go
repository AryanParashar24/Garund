package util

import (
	appsv1 "k8s.io/api/apps/v1" // Import the appv1 package from the k8s-operator API which helps in defining the custom resource definitions (CRDs) for the application.
	corev1 "k8s.io/api/core/v1" // Import the corev1 package from the Kubernetes API which provides the core resources like Pods, Services, etc.
)

func LabelsMatch(selector map[string]string, labels map[string]string) bool {

	if len(selector) == 0 {
		return false
	}

	for key, value := range selector {

		if labels[key] != value {
			return false
		}
	}

	return true
}

func MatchServiceToPods(
	service corev1.Service,
	pods []corev1.Pod,
) []corev1.Pod {

	matched := []corev1.Pod{}

	for _, pod := range pods {

		if pod.Namespace != service.Namespace {
			continue
		}

		if LabelsMatch(service.Spec.Selector, pod.Labels) {
			matched = append(matched, pod)
		}
	}

	return matched
}

func MatchDeploymentToPods(
	deployment appsv1.Deployment,
	pods []corev1.Pod,
) []corev1.Pod {

	matched := []corev1.Pod{}

	selector := deployment.Spec.Selector.MatchLabels

	for _, pod := range pods {

		if pod.Namespace != deployment.Namespace {
			continue
		}

		if LabelsMatch(selector, pod.Labels) {
			matched = append(matched, pod)
		}
	}

	return matched
}

func FindDeploymentForReplicaSet(
	rs appsv1.ReplicaSet,
	deployments []appsv1.Deployment,
) *appsv1.Deployment {

	for _, owner := range rs.OwnerReferences {

		if owner.Kind != "Deployment" {
			continue
		}

		for _, dep := range deployments {

			if dep.Name == owner.Name &&
				dep.Namespace == rs.Namespace {

				return &dep
			}
		}
	}

	return nil
}

func ReplicaSetNameFromPod(
	pod corev1.Pod,
) string {

	for _, owner := range pod.OwnerReferences {

		if owner.Kind == "ReplicaSet" {
			return owner.Name
		}
	}

	return ""
}

func DeploymentNameFromPod(
	pod corev1.Pod,
	deployments []appsv1.Deployment,
) string {

	for _, dep := range deployments {

		if dep.Namespace != pod.Namespace {
			continue
		}

		if LabelsMatch(dep.Spec.Selector.MatchLabels, pod.Labels) {
			return dep.Name
		}
	}

	return ""
}
