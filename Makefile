# Repository-wide targets. Each sample module also has its own Makefile.
# Versions are the same ones .github/workflows/ci.yml pins.
REDOCLY_VERSION    ?= 2.72.1
OASDIFF_VERSION    ?= v1.35.0
OPENAPI_TS_VERSION ?= 7.13.0
BASE_REF           ?= origin/main

MODULES := problem oapi-codegen ogen
CONTRACT := oapi-codegen/api/openapi.yaml

.PHONY: help lint breaking changelog contract-sync generate check-generated test ts-client ci

help:              ## list targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-16s %s\n", $$1, $$2}'

lint:              ## lint every contract against redocly.yaml
	npx --yes @redocly/cli@$(REDOCLY_VERSION) lint

breaking:          ## fail if the contract breaks clients compared with $(BASE_REF)
	git show $(BASE_REF):$(CONTRACT) > .base-openapi.yaml
	go run github.com/oasdiff/oasdiff@$(OASDIFF_VERSION) breaking .base-openapi.yaml $(CONTRACT) --fail-on ERR
	@rm -f .base-openapi.yaml

changelog:         ## human-readable list of contract changes since $(BASE_REF)
	git show $(BASE_REF):$(CONTRACT) > .base-openapi.yaml
	go run github.com/oasdiff/oasdiff@$(OASDIFF_VERSION) changelog .base-openapi.yaml $(CONTRACT)
	@rm -f .base-openapi.yaml

contract-sync:     ## copy the contract into the ogen module (they must stay identical)
	cp $(CONTRACT) ogen/api/openapi.yaml

generate:          ## regenerate Go code in both sample modules
	go -C oapi-codegen generate ./...
	go -C ogen generate ./...

check-generated: generate ## fail if committed generated code is stale
	git diff --exit-code -- oapi-codegen/internal ogen/internal

test:              ## vet and test every module
	@for m in $(MODULES); do go -C $$m vet ./... && go -C $$m test ./... || exit 1; done

ts-client:         ## regenerate the TypeScript types the web UI compiles against
	npx --yes openapi-typescript@$(OPENAPI_TS_VERSION)   # input and output come from redocly.yaml

ci: lint check-generated test ## what CI runs (breaking needs a base branch)
