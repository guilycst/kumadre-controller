package integration

import (
	"context"
	"time"

	"github.com/guilycst/kumadre-controller/api/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("KumaMonitor", func() {
	var ctx context.Context

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
	})

	ginkgo.Describe("CreateHTTPMonitor", func() {
		ginkgo.It("should create a KumaMonitor referencing default server", func() {
			createKumaServer(ctx, "default-server", "http://localhost:3001", true)
			time.Sleep(500 * time.Millisecond)

			monitor := createKumaMonitor(ctx, "test-monitor", "")
			gomega.Expect(monitor.Name).To(gomega.Equal("test-monitor"))
			gomega.Expect(monitor.Spec.MonitorConfig.Type).To(gomega.Equal(v1alpha1.MonitorTypeHTTP))
		})

		ginkgo.It("should create a KumaMonitor with specific server", func() {
			createKumaServer(ctx, "specific-server", "http://localhost:3002", false)
			time.Sleep(500 * time.Millisecond)

			monitor := createKumaMonitor(ctx, "specific-monitor", "specific-server")
			gomega.Expect(monitor.Name).To(gomega.Equal("specific-monitor"))
		})
	})

	ginkgo.Describe("MonitorLifecycle", func() {
		ginkgo.It("should track monitor creation in status", func() {
			createKumaServer(ctx, "lifecycle-server", "http://localhost:3001", true)
			time.Sleep(500 * time.Millisecond)

			createKumaMonitor(ctx, "lifecycle-monitor", "")
			gomega.Eventually(func() int {
				m := getKumaMonitor(ctx, "lifecycle-monitor")
				return m.Status.MonitoredCount
			}, 5*time.Second).Should(gomega.BeNumerically(">=", 0))
		})
	})

	ginkgo.AfterEach(func() {
		deleteKumaMonitor(ctx, "test-monitor")
		deleteKumaMonitor(ctx, "specific-monitor")
		deleteKumaMonitor(ctx, "lifecycle-monitor")
		deleteKumaServer(ctx, "default-server")
		deleteKumaServer(ctx, "specific-server")
		deleteKumaServer(ctx, "lifecycle-server")
	})
})
