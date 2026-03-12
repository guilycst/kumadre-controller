package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	kumadrev1alpha1 "github.com/guilycst/kumadre-controller/api/v1alpha1"
	"github.com/guilycst/kumadre-controller/internal/controller"
)

var (
	kubeClient    client.Client
	kubeClientset *kubernetes.Clientset
	testEnv       *envtest.Environment
	ctx           context.Context
	cancel        context.CancelFunc
	testNamespace string
)

func TestIntegration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Integration Test Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("Setting up test environment")

	ctx, cancel = context.WithCancel(context.Background())

	testNamespace = "test-kumadre-" + fmt.Sprintf("%d", time.Now().Unix())

	gomega.Expect(setupTestEnvironment()).To(gomega.Succeed())
	gomega.Expect(setupTestNamespace()).To(gomega.Succeed())
})

var _ = ginkgo.AfterSuite(func() {
	ginkgo.By("Tearing down test environment")

	if cancel != nil {
		cancel()
	}

	if testEnv != nil {
		ginkgo.By("Stopping test environment")
		testEnv.Stop()
	}

	if kubeClientset != nil {
		ginkgo.By("Deleting test namespace")
		_ = kubeClientset.CoreV1().Namespaces().Delete(ctx, testNamespace, metav1.DeleteOptions{})
	}
})

func setupTestEnvironment() error {
	ginkgo.GinkgoHelper()

	// First setup the scheme
	gomega.Expect(setupScheme()).To(gomega.Succeed())

	// Get the CRD directory path
	crdPath := "/Users/guilhermecastro/repos/kumadre-controller/config/crd/bases"

	// Use local control plane via envtest with CRDs
	testEnv = &envtest.Environment{
		BinaryAssetsDirectory: "/tmp/kubebuilder-bin/k8s/1.30.3-darwin-arm64",
		CRDDirectoryPaths:     []string{crdPath},
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return fmt.Errorf("failed to start test environment: %w", err)
	}

	kubeClientset, err = kubernetes.NewForConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kube clientset: %w", err)
	}

	kubeClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return fmt.Errorf("failed to create kube client: %w", err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}

	err = (&controller.KumaMonitorReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to setup controller: %w", err)
	}

	go func() {
		defer ginkgo.GinkgoRecover()
		err := mgr.Start(ctx)
		gomega.Expect(err).ToNot(gomega.HaveOccurred(), "manager started")
	}()

	return nil
}

func setupTestNamespace() error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: testNamespace},
	}
	_, err := kubeClientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	return err
}

func setupScheme() error {
	gomega.Expect(kumadrev1alpha1.AddToScheme(scheme.Scheme)).To(gomega.Succeed())
	return nil
}

func boolPtr(b bool) *bool {
	return &b
}

func createKumaServer(ctx context.Context, name string, url string, isDefault bool) *kumadrev1alpha1.KumaServer {
	server := &kumadrev1alpha1.KumaServer{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kumadrev1alpha1.KumaServerSpec{
			URL:     url,
			Default: isDefault,
			Auth: kumadrev1alpha1.KumaServerAuth{
				UsernamePassword: &kumadrev1alpha1.UsernamePasswordAuth{
					Username: "admin",
					PasswordSecretRef: kumadrev1alpha1.SecretRef{
						Name: "test-creds",
						Key:  "password",
					},
				},
			},
		},
	}
	gomega.Expect(kubeClient.Create(ctx, server)).To(gomega.Succeed())
	return server
}

func createKumaMonitor(ctx context.Context, name string, serverRef string) *kumadrev1alpha1.KumaMonitor {
	monitor := &kumadrev1alpha1.KumaMonitor{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kumadrev1alpha1.KumaMonitorSpec{
			ServerRef:      serverRef,
			DiscoveryMode:  kumadrev1alpha1.DiscoveryModeEager,
			WatchResources: []kumadrev1alpha1.ResourceKind{kumadrev1alpha1.ResourceKindIngress},
			MonitorConfig: kumadrev1alpha1.MonitorConfig{
				Type:     kumadrev1alpha1.MonitorTypeHTTP,
				Interval: 60,
			},
		},
	}
	gomega.Expect(kubeClient.Create(ctx, monitor)).To(gomega.Succeed())
	return monitor
}

func getKumaServer(ctx context.Context, name string) *kumadrev1alpha1.KumaServer {
	var server kumadrev1alpha1.KumaServer
	gomega.Expect(kubeClient.Get(ctx, types.NamespacedName{Name: name}, &server)).To(gomega.Succeed())
	return &server
}

func getKumaMonitor(ctx context.Context, name string) *kumadrev1alpha1.KumaMonitor {
	var monitor kumadrev1alpha1.KumaMonitor
	gomega.Expect(kubeClient.Get(ctx, types.NamespacedName{Name: name}, &monitor)).To(gomega.Succeed())
	return &monitor
}

func deleteKumaServer(ctx context.Context, name string) {
	var server kumadrev1alpha1.KumaServer
	err := kubeClient.Get(ctx, types.NamespacedName{Name: name}, &server)
	if err == nil {
		gomega.Expect(kubeClient.Delete(ctx, &server)).To(gomega.Succeed())
	}
}

func deleteKumaMonitor(ctx context.Context, name string) {
	var monitor kumadrev1alpha1.KumaMonitor
	err := kubeClient.Get(ctx, types.NamespacedName{Name: name}, &monitor)
	if err == nil {
		gomega.Expect(kubeClient.Delete(ctx, &monitor)).To(gomega.Succeed())
	}
}
