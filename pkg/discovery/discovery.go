package discovery

import (
	"context"
	"fmt"

	"github.com/guilycst/kumadre-controller/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type Target struct {
	Name      string
	URL       string
	Namespace string
	Type      string
	Host      string
	Port      int
}

type Discoverer interface {
	Discover(ctx context.Context, c client.Client, recorder record.EventRecorder) ([]Target, error)
}

type LazyDiscoverer struct{}

func NewLazyDiscoverer() *LazyDiscoverer {
	return &LazyDiscoverer{}
}

func (d *LazyDiscoverer) Discover(ctx context.Context, c client.Client, recorder record.EventRecorder) ([]Target, error) {
	var targets []Target

	var ingressList networkingv1.IngressList
	if err := c.List(ctx, &ingressList); err != nil {
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	for _, ing := range ingressList.Items {
		ann := ing.Annotations
		enabled := ann["kumadre.controller/enabled"]
		if enabled != "true" {
			continue
		}

		monitorType := ann["kumadre.controller/monitor-type"]
		if monitorType == "" {
			monitorType = "http"
		}

		path := ann["kumadre.controller/path"]
		if path == "" {
			path = "/"
		}

		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}

			host := rule.Host
			if host == "" {
				host = "*"
			}

			// Build URL
			var url string
			customURL := ann["kumadre.controller/custom-url"]
			if customURL != "" {
				url = customURL
			} else {
				scheme := "http"
				if monitorType == "https" {
					scheme = "https"
				}
				url = fmt.Sprintf("%s://%s%s", scheme, host, path)
			}

			targets = append(targets, Target{
				Name:      fmt.Sprintf("%s-%s", ing.Name, host),
				URL:       url,
				Namespace: ing.Namespace,
				Type:      monitorType,
				Host:      host,
			})

			klog.Infof("Discovered lazy target: %s (URL: %s)", targets[len(targets)-1].Name, url)
		}
	}

	return targets, nil
}

type EagerDiscoverer struct {
	WatchResources []v1alpha1.ResourceKind
}

func NewEagerDiscoverer(watchResources []v1alpha1.ResourceKind) *EagerDiscoverer {
	if len(watchResources) == 0 {
		watchResources = []v1alpha1.ResourceKind{
			v1alpha1.ResourceKindIngress,
		}
	}
	return &EagerDiscoverer{
		WatchResources: watchResources,
	}
}

func (d *EagerDiscoverer) Discover(ctx context.Context, c client.Client, recorder record.EventRecorder) ([]Target, error) {
	var targets []Target

	for _, kind := range d.WatchResources {
		switch kind {
		case v1alpha1.ResourceKindIngress:
			ingTargets, err := d.discoverIngress(ctx, c)
			if err != nil {
				klog.Errorf("Failed to discover ingress targets: %v", err)
				continue
			}
			targets = append(targets, ingTargets...)

		case v1alpha1.ResourceKindHTTPRoute:
			routeTargets, err := d.discoverHTTPRoute(ctx, c)
			if err != nil {
				klog.Errorf("Failed to discover httproute targets: %v", err)
				continue
			}
			targets = append(targets, routeTargets...)

		case v1alpha1.ResourceKindService:
			svcTargets, err := d.discoverService(ctx, c)
			if err != nil {
				klog.Errorf("Failed to discover service targets: %v", err)
				continue
			}
			targets = append(targets, svcTargets...)

		case v1alpha1.ResourceKindNode:
			nodeTargets, err := d.discoverNode(ctx, c)
			if err != nil {
				klog.Errorf("Failed to discover node targets: %v", err)
				continue
			}
			targets = append(targets, nodeTargets...)
		}
	}

	return targets, nil
}

func (d *EagerDiscoverer) discoverIngress(ctx context.Context, c client.Client) ([]Target, error) {
	var targets []Target

	var ingressList networkingv1.IngressList
	if err := c.List(ctx, &ingressList); err != nil {
		return nil, err
	}

	for _, ing := range ingressList.Items {
		// Skip if explicitly disabled via annotation
		if ing.Annotations["kumadre.controller/enabled"] == "false" {
			continue
		}

		for _, rule := range ing.Spec.Rules {
			if rule.HTTP == nil {
				continue
			}

			host := rule.Host
			if host == "" {
				host = "*"
			}

			// Get first path
			var defaultPath string
			for _, p := range rule.HTTP.Paths {
				defaultPath = p.Path
				break
			}

			if defaultPath == "" {
				defaultPath = "/"
			}

			scheme := "http"
			if ing.Spec.TLS != nil && len(ing.Spec.TLS) > 0 {
				scheme = "https"
			}

			url := fmt.Sprintf("%s://%s%s", scheme, host, defaultPath)

			target := Target{
				Name:      fmt.Sprintf("ingress-%s-%s", ing.Namespace, ing.Name),
				URL:       url,
				Namespace: ing.Namespace,
				Type:      "http",
				Host:      host,
			}

			targets = append(targets, target)
			klog.Infof("Discovered eager target (ingress): %s (URL: %s)", target.Name, url)
		}
	}

	return targets, nil
}

func (d *EagerDiscoverer) discoverHTTPRoute(ctx context.Context, c client.Client) ([]Target, error) {
	var targets []Target

	var routeList gatewayv1.HTTPRouteList
	if err := c.List(ctx, &routeList); err != nil {
		return nil, err
	}

	for _, route := range routeList.Items {
		for _, rule := range route.Spec.Rules {
			for _, match := range rule.Matches {
				path := "/"
				if match.Path != nil && *match.Path.Value != "" {
					path = *match.Path.Value
				}

				// Try to resolve backend
				for _, ref := range rule.BackendRefs {
					// Get the service name from the ref
					if ref.Kind != nil && *ref.Kind == "Service" {
						serviceName := string(ref.Name)

						// Try to resolve port
						var port int
						if ref.Port != nil {
							port = int(*ref.Port)
						} else {
							port = 80
						}

						// We need the actual cluster IP - for now use a placeholder
						// In real implementation, we'd need to resolve the service
						host := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, route.Namespace)

						url := fmt.Sprintf("http://%s:%d%s", host, port, path)

						target := Target{
							Name:      fmt.Sprintf("httproute-%s-%s", route.Namespace, route.Name),
							URL:       url,
							Namespace: route.Namespace,
							Type:      "http",
							Host:      host,
							Port:      port,
						}

						targets = append(targets, target)
						klog.Infof("Discovered eager target (httproute): %s (URL: %s)", target.Name, url)
					}
				}
			}
		}
	}

	return targets, nil
}

func (d *EagerDiscoverer) discoverService(ctx context.Context, c client.Client) ([]Target, error) {
	var targets []Target

	// This would require more complex logic to determine which services to monitor
	// For now, we skip this - services need explicit annotation to be monitored
	return targets, nil
}

func (d *EagerDiscoverer) discoverNode(ctx context.Context, c client.Client) ([]Target, error) {
	var targets []Target

	var nodeList corev1.NodeList
	if err := c.List(ctx, &nodeList); err != nil {
		return nil, err
	}

	for _, node := range nodeList.Items {
		// Get node internal IP
		var internalIP string
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				internalIP = addr.Address
				break
			}
		}

		if internalIP == "" {
			continue
		}

		// Skip nodes that are not ready
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status != corev1.ConditionTrue {
				continue
			}
		}

		target := Target{
			Name:      fmt.Sprintf("node-%s", node.Name),
			URL:       fmt.Sprintf("http://%s:9100/metrics", internalIP),
			Namespace: "",
			Type:      "ping",
			Host:      internalIP,
		}

		targets = append(targets, target)
		klog.Infof("Discovered eager target (node): %s (URL: %s)", target.Name, target.URL)
	}

	return targets, nil
}

func GetDiscoverer(mode v1alpha1.DiscoveryMode, watchResources []v1alpha1.ResourceKind) Discoverer {
	switch mode {
	case v1alpha1.DiscoveryModeLazy:
		return NewLazyDiscoverer()
	case v1alpha1.DiscoveryModeEager:
		return NewEagerDiscoverer(watchResources)
	default:
		return NewEagerDiscoverer(watchResources)
	}
}
