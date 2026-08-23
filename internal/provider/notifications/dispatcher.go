package notifications

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesDispatcher struct {
	Client     client.Client
	HTTPClient *http.Client
}

func (d KubernetesDispatcher) Notify(ctx context.Context, request NotificationRequest) error {
	if d.Client == nil {
		return nil
	}
	project := &cicdv1alpha1.Project{}
	if err := d.Client.Get(ctx, client.ObjectKey{Namespace: request.Namespace, Name: request.Project}, project); err != nil {
		return fmt.Errorf("load notification project: %w", err)
	}
	config := project.Spec.Notifications
	if config == nil || !config.Enabled || !matches(*config, request) {
		return nil
	}
	if config.SecretRef == nil || config.SecretRef.Name == "" {
		return errors.New("notification configuration invalid: endpoint secret is not configured")
	}
	secret := &corev1.Secret{}
	if err := d.Client.Get(ctx, client.ObjectKey{Namespace: request.Namespace, Name: config.SecretRef.Name}, secret); err != nil {
		return fmt.Errorf("load notification endpoint secret: %w", err)
	}
	key := config.SecretRef.Key
	if key == "" {
		key = "url"
	}
	endpoint := string(secret.Data[key])
	if endpoint == "" {
		return errors.New("notification configuration invalid: endpoint key is missing")
	}
	var selected NotificationProvider
	switch config.Provider {
	case "webhook":
		selected = GenericWebhook{Endpoint: endpoint, Client: d.HTTPClient}
	case "slack", "teams", "email":
		selected = Skeleton{ProviderName: config.Provider}
	default:
		return fmt.Errorf("unsupported notification provider %q", config.Provider)
	}
	return selected.Send(ctx, request)
}

func matches(config cicdv1alpha1.ProjectNotificationSpec, request NotificationRequest) bool {
	if len(config.Events) > 0 && !slices.Contains(config.Events, string(request.Event)) {
		return false
	}
	filters := config.Filters
	return (filters.Project == "" || filters.Project == request.Project) &&
		(filters.Repository == "" || filters.Repository == request.Repository) &&
		(filters.Environment == "" || filters.Environment == request.Environment) &&
		(filters.Phase == "" || filters.Phase == request.Phase)
}
