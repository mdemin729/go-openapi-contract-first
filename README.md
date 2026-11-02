# go-openapi-contract-first: oapi-codegen vs ogen

One OpenAPI 3.0.3 contract (a "Hello world" API with `GET /hello`,
`POST /greetings` and `GET /healthz`), implemented contract-first twice:

| Folder | Generator | Port |
|---|---|---|
| [`oapi-codegen/`](oapi-codegen) | [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) v2: `net/http` strict server + kin-openapi validation middleware | 8080 |
| [`ogen/`](ogen) | [ogen](https://github.com/ogen-go/ogen) v1.24: everything generated, no runtime spec | 8081 |

Each folder is its own Go module with the same layout, the same
`api/openapi.yaml` (byte-identical), the same handler logic, the same
RFC 9457 error format, the same docs endpoints (`/docs`, `/redoc`,
`/openapi.yaml`, `/openapi.json`), and **the same test suite and benchmarks**.

```bash
go -C oapi-codegen test ./...
```

```bash
go -C ogen test ./...
```

## Results

Measured on Go 1.27.3, windows/amd64. For binary size, the server was built with `-trimpath -ldflags "-s -w"`.

| | oapi-codegen | ogen |
|---|---|---|
| Generated files | 1 | 19 |
| Generated lines | 705 | 3,479 (677 of them are the client) |
| Hand-written lines (handler, server, docs, main) | 234 | 220 |
| Extra wiring needed | validation middleware, spec loading, route-error mapping | error handler, 404/405 hooks |
| Third-party modules linked into the binary | 11 | 27 (OpenTelemetry, zap, jx, regexp2 …) |
| Binary size | 9.9 MB | 8.7 MB |
| `go generate` | ~0.5 s | ~0.7 s |
| Shared test suite (14 HTTP cases + response-vs-contract check) | pass | pass |

### Behaviour on invalid input

Both reject the same requests with the same status codes. Only the messages differ:

| Request | oapi-codegen (kin-openapi) | ogen |
|---|---|---|
| `name` fails `pattern` | `parameter "name" in query has an error: string doesn't match the regular expression …` | `operation GetHello: decode params: query: "name": string: no regex match: …` |
| `name` > 50 chars | `… maximum string length is 50` | `… len 51 greater than maximum 50` |
| bad enum | `Error at "/language": value is not one of the allowed values ["en","es","de","fr"]` | `validate: invalid: language (invalid value: xx)` |
| unknown field | `property "extra" is unsupported` | `unexpected field "extra"` |
| missing required | `property "name" is missing` | `invalid: name (field required)` |
| wrong method | 405 | 405 (+ `Allow` header) |

Other differences in output:

- **`date-time` precision:** oapi-codegen writes `2026-11-01T23:54:51.9252408Z`
  (it uses `encoding/json`), while ogen writes `2026-11-01T23:54:47Z`. Both are valid RFC 3339.
- **`/openapi.json` key order:** oapi-codegen serialises kin-openapi's parsed
  document. The ogen version converts YAML through a Go map, so keys come out
  sorted. Swagger UI and ReDoc load `/openapi.yaml`, so this doesn't affect them.

### Throughput

These numbers come from an in-process `httptest` benchmark (`go test -bench . ./internal/server/`) running 3 times on a 32-thread CPU. They are approximate.

| Benchmark | oapi-codegen | ogen |
|---|---|---|
| `GET /hello` | ~5.3 µs, 54 allocs | ~5.5 µs, 54 allocs |
| `POST /greetings` (valid) | ~9–12 µs, 75 allocs | ~8–9 µs, 60 allocs |
| `POST /greetings` (rejected) | ~37 µs, 139 allocs | ~19 µs, 88 allocs |

For a trivial GET, the cost of building an `httptest` request dominates, so the two
are even. ogen pulls ahead when bodies are decoded and validated, and especially on the
error path: kin-openapi validates against a schema that is interpreted at runtime,
while ogen runs generated code.

## Which one to pick

**oapi-codegen** suits teams that want the least magic: small, readable generated code;
plain structs with pointers for optional fields; and a choice of router (`net/http`, Chi, Echo, Gin,
Fiber …). Validation is a separate, optional middleware. Its OpenAPI 3.1 support
is partial.

**ogen** suits teams that want the contract enforced by the compiler: sum-type
responses, `Opt*` types, built-in validation, a generated client and
OpenTelemetry. The cost is more generated code, more dependencies, its own router
and JSON stack, and some opinions, such as "convenient errors" requiring a
`default` response on every operation. It supports OpenAPI 3.1 well.

## Run both side by side

```bash
go -C oapi-codegen run ./cmd/server -addr :8080
```

```bash
go -C ogen run ./cmd/server -addr :8081
```

Then compare http://localhost:8080/docs with http://localhost:8081/docs.
