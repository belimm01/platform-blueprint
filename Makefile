KUBECONFORM_VERSION := v0.6.7
KUBERNETES_SCHEMA_VERSION := 1.31.0

.PHONY: test verify verify-manifests render

test:
	go test -race ./...

verify: verify-manifests
	gofmt -w .
	git diff --exit-code
	go vet ./...
	go test -race ./...
	go run ./cmd/platformctl validate -file examples/catalog-service.json

verify-manifests:
	@manifest_dir="$$(mktemp -d)"; \
	trap 'rm -rf "$$manifest_dir"' EXIT; \
	go run ./cmd/platformctl render -file examples/catalog-service.json -out "$$manifest_dir/manifests.yaml"; \
	go run github.com/yannh/kubeconform/cmd/kubeconform@$(KUBECONFORM_VERSION) -kubernetes-version $(KUBERNETES_SCHEMA_VERSION) -strict -summary -ignore-missing-schemas "$$manifest_dir/manifests.yaml"

render:
	mkdir -p build
	go run ./cmd/platformctl render -file examples/catalog-service.json -out build/catalog.yaml
