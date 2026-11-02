# ADR-0002: oapi-codegen is the OpenAPI generator for Go services

- **Status:** Proposed
- **Date:** 2026-11-01
- **Deciders:** Software development team
- **Related:** [ADR-0001](0001-rest-for-all-apis.md), [root README comparison](../../README.md)

## Context

ADR-0001 requires every Go service to generate its server stubs, models and clients from an OpenAPI contract. Two mature generators exist for Go, and this repository implements the same "Hello World" contract with both so they can be compared on equal terms:

| | oapi-codegen v2.8 | ogen v1.24 |
|---|---|---|
| Design | Generates models, a thin router adapter and typed "strict" handlers for `net/http` (or Chi, Echo, Gin, Fiber). Request validation is a separate middleware backed by kin-openapi, which interprets the spec at runtime. | Generates everything: router, JSON encoders and decoders, validators, client, OpenTelemetry hooks. No runtime spec. Own JSON stack (`go-faster/jx`). |
| Generated code for the sample | 1 file, ~700 lines (including client) | 19 files, ~3,500 lines |
| Third-party modules linked into the binary | 11 | 27 (OpenTelemetry, zap, jx, regexp2, …) |
| Binary size (`-s -w`) | 9.9 MB | 8.7 MB |
| Optional fields | Pointers | `Opt*` wrapper types with `.Or(default)` |
| Response typing | One interface per operation; constructors per status code | Sum types per operation; returning an undeclared type does not compile |
| Validation errors | Human-readable kin-openapi messages | Terse generated messages |
| Throughput, trivial GET | ~5.3 µs | ~5.5 µs (even) |
| Throughput, POST with body | ~9 to 12 µs | ~8 to 9 µs |
| Throughput, rejected request | ~37 µs | ~19 µs |
| Handler returning an error | Hook on the strict handler; `*problem.Problem` keeps its status | `NewError` method required on the handler ("convenient errors"), only when every operation has a `default` response |
| OpenAPI 3.1 | Partial | Good |
| Router | Any; the sample uses Go 1.22+ `net/http.ServeMux` | Its own |
| Community | Largest Go OpenAPI generator by adoption | Smaller, active |

Both pass the same 14-case HTTP test suite, the response-versus-contract check and the generated-client test. Both produce RFC 9457 errors through the shared `problem` package.

## Decision

**New Go services use oapi-codegen**, configured as in `oapi-codegen/oapi-codegen.yaml`: `models`, `std-http-server`, `strict-server`, `client` and `embedded-spec`, with the kin-openapi request validator middleware in front of the router.

Reasons, in order of weight:

1. **Simplicity, which is the criterion ADR-0001 was decided on.** One generated file a reviewer can read, plain structs with pointers, the standard library router, and 11 dependencies instead of 27. A developer who knows `net/http` understands the whole stack in an hour.
2. **No framework lock-in.** Services keep `net/http` middleware, tracing and logging exactly as they are. ogen brings its own router, JSON encoder and OpenTelemetry wiring, which is a second way of doing things we already do.
3. **Runtime validation against the contract is a feature, not a cost, at our scale.** The validator rejects anything the contract does not allow before handler code runs, with messages a UI developer can read. The measured penalty (a few microseconds on the error path, roughly even on the happy path) is irrelevant for an admin UI and internal traffic.
4. **Generated clients are good enough and idiomatic.** `NewClientWithResponses` returns typed responses per status code, including the `Problem` body on errors, so callers switch on `JSON200` / `ApplicationproblemJSON400` instead of parsing.
5. **Broad adoption** means more answers, more examples and more contributors, which matters for a small team.

ogen is a strong tool and would win on strictness and raw decode speed. For a service with very high request rates, or a team that wants the compiler to enforce every response type, it is an acceptable exception, to be recorded in that service's README. OpenAPI 3.1 features (`type: [string, "null"]`, `const`, webhooks) are avoided in contracts until oapi-codegen supports them fully; contracts are written in OpenAPI 3.0.3.

## Consequences

- `go generate ./...` plus the `tool` directive in `go.mod` pins the generator version per service; nothing is installed globally.
- CI regenerates and fails on a diff, so generated code is always current and never edited by hand.
- Handlers implement `gen.StrictServerInterface`; a compile error is the signal that the contract changed.
- Every service imports `github.com/mdemin729/go-openapi-contract-first/problem` (or its successor in the shared module) for errors.
- The `ogen/` module stays in this repository as the comparison baseline; it is not a template for new services.
