.PHONY: generate lint check-generated test run build

generate:          ## regenerate Go code from api/openapi.yaml
	go generate ./...

lint:              ## lint the contract (needs Node) and the Go code
	npx --yes @redocly/cli@1.34.0 lint api/openapi.yaml
	go vet ./...

check-generated: generate  ## fail if committed generated code is stale (use in CI)
	git diff --exit-code -- internal/gen

test:
	go test ./...

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server
