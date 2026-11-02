# go-openapi-contract-first: oapi-codegen vs ogen, plus REST hygiene

Reference implementation for [ADR-0001 "REST with OpenAPI for all service APIs"](docs/adr/0001-rest-for-all-apis.md), decided on 2026-10-30. It shows, end to end, what a contract-first Go service looks like when it follows the [REST hygiene guide](docs/rest-hygiene.md):

- one OpenAPI 3.0.3 contract as the single source of truth;
- server stubs, models and clients generated from it, with two generators compared side by side;
- RFC 9457 problem details for every error, through a shared package;
- lint, breaking-change detection, generated-code freshness and a generated TypeScript client, all enforced in CI.

**Recommendation for new Go services: oapi-codegen.** The reasoning and the measurements are in [ADR-0002](docs/adr/0002-oapi-codegen-for-go-services.md); the short version is at the end of this file.

## Layout

```
docs/adr/0001-rest-for-all-apis.md         the decision from the 2026-10-30 meeting and its revisit triggers
docs/adr/0002-oapi-codegen-for-go-services.md  generator choice with the comparison
docs/rest-hygiene.md                       the API style guide every service follows
redocly.yaml                               lint rules that enforce the style guide; also configures openapi-typescript
.github/workflows/ci.yml                   contract lint, breaking-change check, Go matrix, TypeScript client check
Makefile                                   the same checks for a laptop: make lint / breaking / changelog / test / ts-client
problem/                                   shared RFC 9457 package (standard library only), used by both samples
oapi-codegen/                              the contract implemented with oapi-codegen (recommended)   port 8080
ogen/                                      the same contract implemented with ogen (comparison baseline)   port 8081
clients/typescript/                        TypeScript types for the web UI, generated and committed
```

Each sample is its own Go module with the same layout, the same `api/openapi.yaml` (byte-identical, checked in CI), the same handler logic, the same error format, the same docs endpoints (`/docs`, `/redoc`, `/openapi.yaml`, `/openapi.json`), and **the same test suite, benchmarks and generated-client test**.

## Try it

```bash
go -C oapi-codegen run ./cmd/server -addr :8080
```

Open http://localhost:8080/docs, or:

```bash
curl -s -X POST localhost:8080/greetings -H "Content-Type: application/json" -d "{\"name\":\"Ana\",\"language\":\"xx\"}"
```

returns `400 application/problem+json`:

```json
{"type":"about:blank","title":"Bad Request","status":400,"detail":"request body has an error: doesn't match schema #/components/schemas/GreetingRequest: Error at \"/language\": value is not one of the allowed values [\"en\",\"es\",\"de\",\"fr\"]","instance":"/greetings"}
```

## The hygiene checks

| Check | Command | What fails |
|---|---|---|
| Contract follows the style guide | `make lint` | missing `operationId`, non-kebab path, error response that is not `application/problem+json`, property that is not camelCase, unused schema, … |
| No breaking change against `main` | `make breaking` | removed or renamed field, new required input, narrowed enum, changed type |
| Change list for the PR | `make changelog` | (informational) |
| Generated code is current | `make check-generated` | contract edited without `go generate`, or generated code edited by hand |
| Services behave as the contract says | `make test` | any response that does not validate against the contract; any error that is not a `Problem` |
| Web UI types are current | `make ts-client` then `git diff` | contract changed without regenerating `clients/typescript/hello-api.d.ts` |

CI ([`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs all of them on every pull request. Tool versions are pinned in the Makefile and the workflow. `make` is not required: every target is a one-line command you can copy.

## How errors work

Both samples import [`problem`](problem/problem.go). Contract validation failures, unknown routes, wrong methods (with an `Allow` header) and handler errors all come out as the `Problem` schema declared in the contract. A handler returns `problem.New(http.StatusConflict, "already shipped")` and the client receives exactly that status and body; any other error becomes a `500` with the cause logged, never sent.

The generated clients see errors as types: oapi-codegen's `ApplicationproblemJSON400`, ogen's `*oas.Problem` member of the response sum type, openapi-fetch's typed `error`. See the `client_test.go` in each sample and [`clients/typescript/README.md`](clients/typescript/README.md).

## Generator comparison

Measured on Go 1.27.3, windows/amd64. For binary size, the server was built with `-trimpath -ldflags "-s -w"`.

| | oapi-codegen v2.8 | ogen v1.24 |
|---|---|---|
| Generated files | 1 | 19 |
| Generated lines (incl. client) | ~700 | ~3,500 |
| Hand-written lines (handler, server, docs, main) | ~240 | ~240 |
| Extra wiring needed | validation middleware, spec loading, route-error mapping | error handler, 404/405 hooks, `NewError` on the handler |
| Third-party modules linked into the binary | 11 | 27 (OpenTelemetry, zap, jx, regexp2 …) |
| Binary size | 9.9 MB | 8.7 MB |
| `go generate` | ~0.5 s | ~0.7 s |
| Shared test suite (14 HTTP cases, contract check, generated client) | pass | pass |

### Behaviour on invalid input

Both reject the same requests with the same status codes and the same `Problem` shape. Only the `detail` text differs:

| Request | oapi-codegen (kin-openapi) | ogen |
|---|---|---|
| `name` fails `pattern` | `parameter "name" in query has an error: string doesn't match the regular expression …` | `operation GetHello: decode params: query: "name": string: no regex match: …` |
| `name` > 50 chars | `… maximum string length is 50` | `… len 51 greater than maximum 50` |
| bad enum | `Error at "/language": value is not one of the allowed values ["en","es","de","fr"]` | `validate: invalid: language (invalid value: xx)` |
| unknown field | `property "extra" is unsupported` | `unexpected field "extra"` |
| missing required | `property "name" is missing` | `invalid: name (field required)` |
| wrong method | 405 + `Allow` | 405 + `Allow` |

Other differences:

- **`date-time` precision:** oapi-codegen writes `2026-11-01T23:54:51.9252408Z` (it uses `encoding/json`), ogen writes `2026-11-01T23:54:47Z`. Both are valid RFC 3339.
- **`/openapi.json` key order:** oapi-codegen serialises kin-openapi's parsed document; the ogen version converts YAML through a Go map, so keys come out sorted. Swagger UI and ReDoc load `/openapi.yaml`, so this does not affect them.
- **Handler errors:** oapi-codegen maps them in a hook on the strict handler. ogen requires a `NewError` method on the handler once every operation declares a `default` response ("convenient errors"), and generates the mapping to the `default` response type.

### Throughput

In-process `httptest` benchmark (`go test -bench . ./internal/server/`), 3 runs on a 32-thread CPU. Approximate.

| Benchmark | oapi-codegen | ogen |
|---|---|---|
| `GET /hello` | ~5.3 µs, 54 allocs | ~5.5 µs, 54 allocs |
| `POST /greetings` (valid) | ~9 to 12 µs, 75 allocs | ~8 to 9 µs, 60 allocs |
| `POST /greetings` (rejected) | ~37 µs, 139 allocs | ~19 µs, 88 allocs |

For a trivial GET, building the `httptest` request dominates, so the two are even. ogen pulls ahead when bodies are decoded and validated, and especially on the error path: kin-openapi validates against a schema interpreted at runtime, ogen runs generated code.

### Which one to pick

**oapi-codegen** (recommended, [ADR-0002](docs/adr/0002-oapi-codegen-for-go-services.md)): the least magic. One readable generated file, plain structs with pointers, the standard library router or any framework, validation as a separate middleware with readable messages, 11 dependencies. OpenAPI 3.1 support is partial, so contracts stay on 3.0.3.

**ogen**: the contract enforced by the compiler. Sum-type responses, `Opt*` types, generated validation, client and OpenTelemetry out of the box, faster on the error path. The cost is more generated code, 27 dependencies, its own router and JSON stack, and opinions such as `NewError`. Acceptable as a documented exception for a service that needs the strictness or the throughput.

## Run both side by side

```bash
go -C oapi-codegen run ./cmd/server -addr :8080
```

```bash
go -C ogen run ./cmd/server -addr :8081
```

Then compare http://localhost:8080/docs with http://localhost:8081/docs.
