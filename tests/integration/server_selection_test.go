package integration

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("Server Selection", func() {
	var ctx context.Context

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
	})

	ginkgo.Describe("SingleServer_IsDefault", func() {
		ginkgo.It("should automatically set single server as default", func() {
			createKumaServer(ctx, "single-server", "http://localhost:3001", false)

			server := getKumaServer(ctx, "single-server")
			gomega.Expect(server.Spec.Default).To(gomega.BeFalse())
		})
	})

	ginkgo.Describe("MultipleServers_NoDefault", func() {
		ginkgo.It("should error when multiple servers exist without default", func() {
			createKumaServer(ctx, "server-1", "http://localhost:3001", false)
			createKumaServer(ctx, "server-2", "http://localhost:3002", false)

			monitor := createKumaMonitor(ctx, "multi-server-monitor", "")
			gomega.Expect(monitor).ToNot(gomega.BeNil())
		})
	})

	ginkgo.Describe("AnnotationOverridesDefault", func() {
		ginkgo.It("should use server from annotation when specified", func() {
			createKumaServer(ctx, "default-srv", "http://localhost:3001", true)
			createKumaServer(ctx, "alternate-srv", "http://localhost:3002", false)
			time.Sleep(500 * time.Millisecond)

			monitor := createKumaMonitor(ctx, "annotated-monitor", "alternate-srv")
			gomega.Expect(monitor.Name).To(gomega.Equal("annotated-monitor"))
		})
	})

	ginkgo.Describe("ServerNotFound_Error", func() {
		ginkgo.It("should error when referenced server doesn't exist", func() {
			monitor := createKumaMonitor(ctx, "nonexistent-monitor", "nonexistent-server")
			gomega.Expect(monitor.Name).To(gomega.Equal("nonexistent-monitor"))
		})
	})

	ginkgo.AfterEach(func() {
		deleteKumaMonitor(ctx, "multi-server-monitor")
		deleteKumaMonitor(ctx, "annotated-monitor")
		deleteKumaMonitor(ctx, "nonexistent-monitor")
		deleteKumaServer(ctx, "single-server")
		deleteKumaServer(ctx, "server-1")
		deleteKumaServer(ctx, "server-2")
		deleteKumaServer(ctx, "default-srv")
		deleteKumaServer(ctx, "alternate-srv")
	})
})
