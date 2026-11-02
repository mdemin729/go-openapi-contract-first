# Hello World API: oapi-codegen

A contract-first "Hello world" REST service in Go (compare with [`../ogen`](../ogen)). `api/openapi.yaml` is the
source of truth. Models, the router and typed server stubs are generated from
it with [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen), and the
running service publishes the same contract with interactive docs.

## Layout

```
api/openapi.yaml            the contract (edit this first)
api/spec.go                 embeds the YAML into the binary
oapi-codegen.yaml           generator config: models + net/http + strict server
internal/gen/               GENERATED, do not edit (go generate ./...)
internal/handler/           business logic: implements gen.StrictServerInterface
internal/server/            wiring: request validation, error format, routes
internal/docs/              /openapi.yaml, /openapi.json, /docs, /redoc
cmd/server/                 main: http.Server with timeouts and graceful shutdown
```

## Workflow

1. Change `api/openapi.yaml`.
2. `go generate ./...` (oapi-codegen is pinned as a Go tool in `go.mod`, so there is nothing to install).
3. The build breaks until `internal/handler` implements any new operation.
4. `go test ./...`. `TestResponsesMatchContract` checks that real responses match the spec.

## Run

```bash
go run ./cmd/server
```

| URL | What |
|---|---|
| http://localhost:8080/docs | Swagger UI (try requests in the browser) |
| http://localhost:8080/redoc | ReDoc reference |
| http://localhost:8080/openapi.yaml | the contract as written |
| http://localhost:8080/openapi.json | the contract as JSON |
| http://localhost:8080/hello?name=Gopher | the API |

```bash
curl -X POST localhost:8080/greetings -H "Content-Type: application/json" -d "{\"name\":\"Ana\",\"language\":\"es\"}"
```

Requests that don't match the contract (bad enum, too long, unknown field,
missing required field) are rejected with `400 application/problem+json`
(RFC 9457) before they reach handler code.
