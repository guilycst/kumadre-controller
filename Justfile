# Kumadre Controller - Justfile
# Just is a command runner, alternative to make

# Default target
default := "help"

# Build
build:
    go build -o bin/controller ./cmd/controller

docker-build:
    docker build -t kumadre-controller:dev .

dev-build: build docker-build
    kind load docker-image kumadre-controller:dev --name kumadre-dev

# Test
test:
    go test ./... -v

test-unit:
    go test ./api/... ./pkg/... ./internal/... -v

test-integration:
    cd tests/integration && go test -v -count=1

test-verbose:
    go test ./... -v -race -coverprofile=coverage.out

# Deploy
deploy-crds:
    kubectl --context=kind-kumadre-dev apply -f config/crd/bases/

deploy: dev-build
    kubectl --context=kind-kumadre-dev apply -f config/rbac/
    kubectl --context=kind-kumadre-dev apply -f config/manager/

deploy-all: deploy-crds deploy

undeploy:
    kubectl --context=kind-kumadre-dev delete -f config/manager/ 2>/dev/null || true
    kubectl --context=kind-kumadre-dev delete -f config/rbac/ 2>/dev/null || true

# Development
setup-kind:
    kind create cluster --name kumadre-dev

setup-traefik:
    helm repo add traefik https://traefik.github.io/charts
    helm repo update
    helm install traefik traefik/traefik --namespace traefik --create-namespace \
        --set "ports.traefik.expose.nodePort=30080" \
        --set "providers.kubernetesIngress.enabled=true" \
        --set "rbac.enabled=true" \
        --kube-context=kind-kumadre-dev

setup-sample-apps:
    kubectl --context=kind-kumadre-dev apply -f hack/manifests/sample-apps.yaml
    kubectl --context=kind-kumadre-dev apply -f hack/manifests/ingress.yaml

setup-dev: setup-kind setup-traefik setup-sample-apps

port-forward-uptime:
    kubectl --context=kind-kumadre-dev port-forward -n uptime-kuma svc/uptime-kuma 3001:3001

port-forward-controller:
    kubectl --context=kind-kumadre-dev port-forward -n system svc/kumadre-controller 8080:8080

# Cleanup
clean:
    rm -rf bin/
    rm -f coverage.out

delete-kind:
    kind delete cluster --name kumadre-dev

# Generate
generate:
    go generate ./...

# Lint
lint:
    go vet ./...
    golangci-lint run || true

# Help
help:
    @echo "Kumadre Controller - Available targets:"
    @echo ""
    @echo "Build:"
    @echo "  build              Build the controller binary"
    @echo "  docker-build       Build Docker image"
    @echo "  dev-build          Build and load to kind"
    @echo ""
    @echo "Test:"
    @echo "  test               Run all tests"
    @echo "  test-unit          Run unit tests only"
    @echo "  test-integration   Run integration tests"
    @echo "  test-verbose       Run tests with verbose output"
    @echo ""
    @echo "Deploy:"
    @echo "  deploy-crds        Deploy CRDs to cluster"
    @echo "  deploy             Deploy controller to kind"
    @echo "  deploy-all         Deploy CRDs and controller"
    @echo "  undeploy           Remove controller from cluster"
    @echo ""
    @echo "Development:"
    @echo "  setup-kind         Create kind cluster"
    @echo "  setup-traefik      Install Traefik ingress"
    @echo "  setup-sample-apps  Deploy sample applications"
    @echo "  setup-dev          Full dev setup"
    @echo "  port-forward-uptime Port forward to Uptime Kuma"
    @echo ""
    @echo "Cleanup:"
    @echo "  clean              Remove build artifacts"
    @echo "  delete-kind        Delete kind cluster"