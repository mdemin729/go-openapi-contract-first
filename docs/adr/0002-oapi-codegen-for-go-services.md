# ADR-0002: oapi-codegen is the OpenAPI generator for Go services

- **Status:** Proposed
- **Date:** 2026-11-01, maintenance data added 2026-11-04
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
| OpenAPI 3.1 | "Initial" since v2.8.0 (2026-07-17); multi-type arrays become `any` | Parser accepts 3.0 to 3.2; multi-type arrays are an error |
| Router | Any; the sample uses Go 1.22+ `net/http.ServeMux` | Its own |
| Community | Largest Go OpenAPI generator by adoption | Smaller, active |

Both pass the same 14-case HTTP test suite, the response-versus-contract check and the generated-client test. Both produce RFC 9457 errors through the shared `problem` package.

### Maintenance and project health (measured 2026-11-04)

GitHub numbers were taken from the GitHub API on 2026-11-04. "6 months" means commits on the default branch since 2026-05-04, "12 months" since 2025-11-04.

| | oapi-codegen | ogen |
|---|---|---|
| Owner | `oapi-codegen` org (moved from `deepmap` in June 2024) | `ogen-go` org, part of the go-faster group |
| Core maintainers | Marcin Romaszewicz (founder) and Jamie Tanna; a governance repo states that only core maintainers review and merge | No CODEOWNERS or governance file; Aleksandr Razumov (ernado) carries the project, the original main author made 5 commits in 12 months |
| Backing and funding | Volunteer-run. Sponsors DevZero, Cybozu and Elastic at one hour per month each; OpenCollective about $2k per year; Tidelift | No sponsorship channel; informal go-faster backing; no company identified |
| Stars / forks | 8,642 / 1,065 | 2,153 / 196 |
| Open issues / open PRs | 236 / 74 | 116 / 32 |
| Latest release | v2.8.2, 2026-10-31 | v1.24.0, 2026-08-07 |
| Releases in 12 months | 7 | 11 |
| Human commits in 6 months | 131 by 24 people | 159 by 14 people |
| Concentration, 12 months | Two maintainers wrote 74% of human commits | One maintainer wrote 43% of human commits |
| Issue response since 2026-08-01 | 7 of 15 new issues answered, median about 34 hours | 1 of 12 new issues answered, after 68 days; maintainers respond through PRs |
| Minimum Go | 1.26 to build the tool on main (1.25 at v2.8.2); generated code needs 1.24 or 1.25 | 1.25 per `go.mod` |
| OpenAPI 2.0 | Not supported | Not supported |
| OpenAPI 3.0 | Full | Full |
| OpenAPI 3.1 | Initial since v2.8.0: webhooks, callbacks, `type: [T, "null"]`, `const`, `oneOf` with `const`. Multi-type arrays become `any` | 3.1 features gated by version; `type: [T, "null"]` since July 2026. No "unsupported features" page; gaps surface as not-implemented errors, skippable with `ignore_not_implemented` |
| Compatibility policy | Semver for generated code and CLI config; internal packages and template overrides are unstable; no release cadence | go-faster "stable": backward compatibility guaranteed, v1 since March 2024 |

Reading of these numbers:

- Both projects are volunteer-run with a bus factor of about two. Neither has a corporate owner. This is an accepted risk for both; the mitigation is that generated code is committed, so a service keeps building even if its generator stops being released.
- oapi-codegen has roughly four times the community, a written governance model, a funding stream and faster issue responses. Its main upstream risk is kin-openapi, which is pre-1.0 and may ship breaking changes; adopting 3.1 support forced the Go 1.25 floor.
- ogen has slightly more commit activity but its original main author has stepped back, it has no funding channel, and new issues mostly go unanswered.
- Neither generator handles multi-type arrays, so contracts stay on OpenAPI 3.0.3 regardless of generator. OpenAPI 2.0 is not an option with either.
- Support for `$schema` dialects, `unevaluatedProperties` and `$id`-based references is undocumented in both projects and should be treated as unsupported.

## Decision

**New Go services use oapi-codegen**, configured as in `oapi-codegen/oapi-codegen.yaml`: `models`, `std-http-server`, `strict-server`, `client` and `embedded-spec`, with the kin-openapi request validator middleware in front of the router.

Reasons, in order of weight:

1. **Simplicity, which is the criterion ADR-0001 was decided on.** One generated file a reviewer can read, plain structs with pointers, the standard library router, and 11 dependencies instead of 27. A developer who knows `net/http` understands the whole stack in an hour.
2. **No framework lock-in.** Services keep `net/http` middleware, tracing and logging exactly as they are. ogen brings its own router, JSON encoder and OpenTelemetry wiring, which is a second way of doing things we already do.
3. **Runtime validation against the contract is a feature, not a cost, at our scale.** The validator rejects anything the contract does not allow before handler code runs, with messages a UI developer can read. The measured penalty (a few microseconds on the error path, roughly even on the happy path) is irrelevant for an admin UI and internal traffic.
4. **Generated clients are good enough and idiomatic.** `NewClientWithResponses` returns typed responses per status code, including the `Problem` body on errors, so callers switch on `JSON200` / `ApplicationproblemJSON400` instead of parsing.
5. **Broad adoption and healthier project governance.** Four times the community, a governance repository, sponsors and faster issue responses (see "Maintenance and project health"), which matters for a small team that will depend on the tool for years.

ogen is a strong tool and would win on strictness and raw decode speed. For a service with very high request rates, or a team that wants the compiler to enforce every response type, it is an acceptable exception, to be recorded in that service's README. Contracts are written in OpenAPI 3.0.3: both generators gained practical 3.1 support in July 2026, but neither handles multi-type arrays, and 3.0.3 keeps every other tool in the chain (Redocly, oasdiff, openapi-typescript, clients in other languages) on their best-tested path.

## Consequences

- `go generate ./...` plus the `tool` directive in `go.mod` pins the generator version per service; nothing is installed globally.
- CI regenerates and fails on a diff, so generated code is always current and never edited by hand.
- Handlers implement `gen.StrictServerInterface`; a compile error is the signal that the contract changed.
- Every service imports `github.com/mdemin729/go-openapi-contract-first/problem` (or its successor in the shared module) for errors.
- The `ogen/` module stays in this repository as the comparison baseline; it is not a template for new services.

## Sources for the maintenance data

- https://github.com/oapi-codegen/oapi-codegen (README, CODEOWNERS, FUNDING.yml, releases)
- https://github.com/oapi-codegen/governance
- https://github.com/oapi-codegen/oapi-codegen/issues/373 and https://github.com/oapi-codegen/oapi-codegen/pull/2336 (OpenAPI 3.1)
- https://github.com/oapi-codegen/oapi-codegen/discussions/1606 (sustainability) and https://github.com/oapi-codegen/oapi-codegen/issues/1519 (release cadence)
- https://opencollective.com/oapi-codegen
- https://github.com/ogen-go/ogen (README, go.mod, releases) and https://github.com/ogen-go/ogen/pull/1708 (`type: [T, "null"]`)
- https://github.com/ogen-go/ogen/issues/907 and https://github.com/ogen-go/ogen/issues/1491 (known gaps)
- https://ogen.dev/docs/features and https://ogen.dev/docs/config
- https://go-faster.org/docs/projects/status
- GitHub REST and GraphQL APIs for stars, forks, issues, commits, contributors and sponsor listings
