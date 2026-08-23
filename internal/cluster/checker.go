package cluster

import (
	"context"
	"fmt"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Result struct{ Version string }

type Checker interface {
	Check(context.Context, *cicdv1alpha1.ClusterTarget) (Result, error)
}

type KubernetesChecker struct {
	Reader             client.Reader
	LocalConfig        *rest.Config
	DiscoveryForConfig func(*rest.Config) (discovery.DiscoveryInterface, error)
}

func (c KubernetesChecker) Check(ctx context.Context, target *cicdv1alpha1.ClusterTarget) (Result, error) {
	config, err := c.config(ctx, target)
	if err != nil {
		return Result{}, err
	}
	factory := c.DiscoveryForConfig
	if factory == nil {
		factory = func(config *rest.Config) (discovery.DiscoveryInterface, error) {
			return discovery.NewDiscoveryClientForConfig(config)
		}
	}
	discoveryClient, err := factory(config)
	if err != nil {
		return Result{}, fmt.Errorf("create discovery client: %w", err)
	}
	version, err := discoveryClient.ServerVersion()
	if err != nil {
		return Result{}, fmt.Errorf("query Kubernetes version: %w", err)
	}
	return Result{Version: version.GitVersion}, nil
}

func (c KubernetesChecker) config(ctx context.Context, target *cicdv1alpha1.ClusterTarget) (*rest.Config, error) {
	ref := target.Spec.KubeconfigSecretRef
	if ref == nil {
		if c.LocalConfig == nil {
			return nil, fmt.Errorf("local cluster REST config is unavailable")
		}
		return rest.CopyConfig(c.LocalConfig), nil
	}
	if c.Reader == nil {
		return nil, fmt.Errorf("Secret reader is unavailable")
	}
	secret := &corev1.Secret{}
	if err := c.Reader.Get(ctx, client.ObjectKey{Namespace: target.Namespace, Name: ref.Name}, secret); err != nil {
		return nil, fmt.Errorf("read kubeconfig Secret %s/%s: %w", target.Namespace, ref.Name, err)
	}
	key := ref.Key
	if key == "" {
		key = "kubeconfig"
	}
	data, ok := secret.Data[key]
	if !ok {
		return nil, fmt.Errorf("kubeconfig Secret %s/%s does not contain key %q", target.Namespace, ref.Name, key)
	}
	config, err := clientcmd.Load(data)
	if err != nil {
		return nil, fmt.Errorf("parse kubeconfig Secret %s/%s: %w", target.Namespace, ref.Name, err)
	}
	overrides := &clientcmd.ConfigOverrides{}
	if target.Spec.Context != "" {
		overrides.CurrentContext = target.Spec.Context
	}
	restConfig, err := clientcmd.NewDefaultClientConfig(*config, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("select kubeconfig context: %w", err)
	}
	return restConfig, nil
}
