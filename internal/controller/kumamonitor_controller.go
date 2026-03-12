package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kumadrev1alpha1 "github.com/guilycst/kumadre-controller/api/v1alpha1"
	"github.com/guilycst/kumadre-controller/internal/uptimekuma"
	"github.com/guilycst/kumadre-controller/pkg/discovery"
)

type KumaMonitorReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *KumaMonitorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&kumadrev1alpha1.KumaMonitor{}).
		Complete(r)
}

func (r *KumaMonitorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	klog.Infof("Reconciling KumaMonitor %s", req.NamespacedName)

	var monitor kumadrev1alpha1.KumaMonitor
	if err := r.Get(ctx, req.NamespacedName, &monitor); err != nil {
		klog.Errorf("Failed to get KumaMonitor: %v", err)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get the KumaServer to use
	kumaServer, err := r.getKumaServer(ctx, monitor.Spec.ServerRef)
	if err != nil {
		klog.Errorf("Failed to get KumaServer: %v", err)
		monitor.Status.Connected = false
		r.Status().Update(ctx, &monitor)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	klog.Infof("Creating UptimeKuma client for %s", kumaServer.Spec.URL)
	uKumaClient, err := r.createUptimeKumaClient(ctx, kumaServer)
	if err != nil {
		klog.Errorf("Failed to create UptimeKuma client: %v", err)
		monitor.Status.Connected = false
		r.Status().Update(ctx, &monitor)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	defer uKumaClient.Disconnect()

	monitor.Status.Connected = true
	klog.Infof("UptimeKuma client created successfully")

	discoverer := discovery.GetDiscoverer(monitor.Spec.DiscoveryMode, monitor.Spec.WatchResources)
	targets, err := discoverer.Discover(ctx, r.Client, r.Recorder)
	if err != nil {
		klog.Errorf("Failed to discover targets: %v", err)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	klog.Infof("Discovered %d targets", len(targets))

	existingMonitors, err := uKumaClient.GetMonitors(ctx)
	if err != nil {
		klog.Errorf("Failed to get monitors from UptimeKuma: %v", err)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, err
	}

	created := 0
	for _, target := range targets {
		monitorConfig := monitor.Spec.MonitorConfig
		if monitorConfig.Type == "" {
			monitorConfig.Type = kumadrev1alpha1.MonitorTypeHTTP
		}
		if monitorConfig.Interval == 0 {
			monitorConfig.Interval = 60
		}

		exists := false
		for _, m := range existingMonitors {
			if m.Name == target.Name {
				exists = true
				break
			}
		}

		if !exists {
			var newMonitor *uptimekuma.Monitor
			var err error

			if target.Type == "ping" {
				newMonitor, err = uKumaClient.CreatePingMonitor(ctx, target.Name, target.Host, monitorConfig.Interval)
			} else {
				newMonitor, err = uKumaClient.CreateHTTPMonitor(ctx, target.Name, target.URL, monitorConfig.Interval)
			}

			if err != nil {
				klog.Errorf("Failed to create monitor for %s: %v", target.Name, err)
				continue
			}

			_ = uKumaClient.AddTag(ctx, newMonitor.ID, "kumadre/managed", "true")
			_ = uKumaClient.AddTag(ctx, newMonitor.ID, "kumadre/namespace", target.Namespace)

			created++
			klog.Infof("Created monitor for %s", target.Name)
		}
	}

	monitor.Status.MonitoredCount = len(targets)
	monitor.Status.LastSync = time.Now().Format(time.RFC3339)

	if err := r.Status().Update(ctx, &monitor); err != nil {
		klog.Errorf("Failed to update status: %v", err)
		return ctrl.Result{}, err
	}

	klog.Infof("Sync complete. Created %d new monitors, total monitored: %d", created, len(targets))

	return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

func (r *KumaMonitorReconciler) getKumaServer(ctx context.Context, serverRef string) (*kumadrev1alpha1.KumaServer, error) {
	var serverList kumadrev1alpha1.KumaServerList
	if err := r.List(ctx, &serverList); err != nil {
		return nil, fmt.Errorf("failed to list KumaServers: %w", err)
	}

	if len(serverList.Items) == 0 {
		return nil, fmt.Errorf("no KumaServer found")
	}

	if serverRef != "" {
		for _, s := range serverList.Items {
			if s.Name == serverRef {
				return &s, nil
			}
		}
		return nil, fmt.Errorf("KumaServer %q not found", serverRef)
	}

	for _, s := range serverList.Items {
		if s.Spec.Default {
			return &s, nil
		}
	}

	if len(serverList.Items) == 1 {
		return &serverList.Items[0], nil
	}

	return nil, fmt.Errorf("multiple KumaServers found, please specify one via serverRef")
}

func (r *KumaMonitorReconciler) createUptimeKumaClient(ctx context.Context, server *kumadrev1alpha1.KumaServer) (*uptimekuma.Client, error) {
	config := server.Spec
	var username, password string

	if config.Auth.UsernamePassword != nil {
		username = config.Auth.UsernamePassword.Username
		secretName := config.Auth.UsernamePassword.PasswordSecretRef.Name
		secretKey := config.Auth.UsernamePassword.PasswordSecretRef.Key

		var secret corev1.Secret
		if err := r.Get(ctx, client.ObjectKey{Namespace: server.Namespace, Name: secretName}, &secret); err != nil {
			return nil, err
		}

		password = string(secret.Data[secretKey])
	} else if config.Auth.APIToken != nil {
		username = "api"
		secretName := config.Auth.APIToken.TokenSecretRef.Name
		secretKey := config.Auth.APIToken.TokenSecretRef.Key

		var secret corev1.Secret
		if err := r.Get(ctx, client.ObjectKey{Namespace: server.Namespace, Name: secretName}, &secret); err != nil {
			return nil, err
		}

		password = string(secret.Data[secretKey])
	} else {
		return nil, fmt.Errorf("no auth method configured")
	}

	uKumaClient, err := uptimekuma.NewClient(ctx, config.URL, username, password, config.InsecureSkipVerify)
	if err != nil {
		return nil, err
	}

	return uKumaClient, nil
}
