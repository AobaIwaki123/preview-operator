package k8s

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Scaler manages replica scaling for preview deployments.
type Scaler struct {
	client    *kubernetes.Clientset
	namespace string
	deploy    string
}

// NewScaler creates a new Scaler instance.
func NewScaler(client *kubernetes.Clientset, namespace, deploy string) *Scaler {
	return &Scaler{
		client:    client,
		namespace: namespace,
		deploy:    deploy,
	}
}

// GetReplicas returns the current replica count of the target deployment.
func (s *Scaler) GetReplicas(ctx context.Context) (int32, error) {
	scale, err := s.client.AppsV1().Deployments(s.namespace).GetScale(ctx, s.deploy, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get scale for %s/%s: %w", s.namespace, s.deploy, err)
	}
	return scale.Spec.Replicas, nil
}

// ScaleTo updates the replica count of the target deployment.
func (s *Scaler) ScaleTo(ctx context.Context, replicas int32) error {
	scale, err := s.client.AppsV1().Deployments(s.namespace).GetScale(ctx, s.deploy, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get scale for %s/%s: %w", s.namespace, s.deploy, err)
	}

	if scale.Spec.Replicas == replicas {
		return nil
	}

	log.Printf("[Scaler] Updating replicas for %s/%s: %d -> %d", s.namespace, s.deploy, scale.Spec.Replicas, replicas)
	scale.Spec.Replicas = replicas
	_, err = s.client.AppsV1().Deployments(s.namespace).UpdateScale(ctx, s.deploy, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update scale for %s/%s: %w", s.namespace, s.deploy, err)
	}
	return nil
}
