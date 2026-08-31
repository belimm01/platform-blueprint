.PHONY: test verify render

test:
	go test -race ./...

verify:
	gofmt -w .
	git diff --exit-code
	go vet ./...
	go test -race ./...
	go run ./cmd/platformctl validate -file examples/catalog-service.json

render:
	go run ./cmd/platformctl render -file examples/catalog-service.json -out build/catalog.yaml
