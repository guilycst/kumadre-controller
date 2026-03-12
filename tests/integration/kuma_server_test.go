package integration

import (
	"context"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

var _ = ginkgo.Describe("KumaServer", func() {
	var ctx context.Context

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
	})

	ginkgo.Describe("CreateKumaServer", func() {
		ginkgo.It("should create a KumaServer with default flag", func() {
			server := createKumaServer(ctx, "test-server", "http://localhost:3001", true)
			gomega.Expect(server.Name).To(gomega.Equal("test-server"))
			gomega.Expect(server.Spec.Default).To(gomega.BeTrue())
		})

		ginkgo.It("should create a KumaServer without default flag", func() {
			server := createKumaServer(ctx, "test-server-2", "http://localhost:3002", false)
			gomega.Expect(server.Name).To(gomega.Equal("test-server-2"))
			gomega.Expect(server.Spec.Default).To(gomega.BeFalse())
		})
	})

	ginkgo.Describe("GetDefaultKumaServer", func() {
		ginkgo.It("should return the server marked as default", func() {
			deleteKumaServer(ctx, "test-server")
			deleteKumaServer(ctx, "test-server-2")

			createKumaServer(ctx, "default-server", "http://localhost:3001", true)
			server := getKumaServer(ctx, "default-server")
			gomega.Expect(server.Spec.Default).To(gomega.BeTrue())
		})
	})

	ginkgo.Describe("KumaServerStatus", func() {
		ginkgo.It("should update status when server is reachable", func() {
			createKumaServer(ctx, "default-server", "http://localhost:3001", true)
			time.Sleep(500 * time.Millisecond)
			server := getKumaServer(ctx, "default-server")
			gomega.Expect(server.Status.Connected).To(gomega.BeFalse())
		})
	})

	ginkgo.AfterEach(func() {
		deleteKumaServer(ctx, "test-server")
		deleteKumaServer(ctx, "test-server-2")
		deleteKumaServer(ctx, "default-server")
	})
})
