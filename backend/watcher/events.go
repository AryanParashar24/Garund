package watcher

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func WatchEvents(
	clientset *kubernetes.Clientset,
) {

	watcher, err := clientset.CoreV1().
		Events("").
		Watch(
			context.Background(),
			metav1.ListOptions{},
		)

	if err != nil {
		log.Println(err)
		return
	}

	for event := range watcher.ResultChan() {

		evt := event.Object.(*corev1.Event)

		Broadcast(map[string]any{
			"resource":  "Event",
			"action":    string(event.Type),
			"reason":    evt.Reason,
			"message":   evt.Message,
			"namespace": evt.Namespace,
		})
	}

}
