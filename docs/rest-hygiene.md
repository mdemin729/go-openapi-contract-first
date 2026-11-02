# REST Hygiene: API Style Guide and Best Practices

This guide is the "REST hygiene" that [ADR-0001](adr/0001-rest-for-all-apis.md) commits us to. It applies to every service in every language. Where a rule can be checked by a machine, `redocly.yaml` checks it and CI fails on violations. Where it cannot, the review checklist at the end applies.

Keywords **MUST**, **SHOULD** and **MAY** are used as in RFC 2119.

## 1. Contract first

- The OpenAPI document (`api/openapi.yaml`, OpenAPI 3.0.3) is the single source of truth. Server stubs, models, clients and documentation are generated from it. **Nothing is generated from code into the contract.**
- Workflow: change the contract, regenerate, let the compiler show what to implement, test, open a pull request. CI regenerates and fails if committed generated code differs.
- One contract per service. Shared schemas (for example `Problem`) are copied, not referenced across repositories, so a service never breaks because another repository changed.
- The running service publishes its own contract at `GET /openapi.yaml` and `GET /openapi.json`, with Swagger UI at `GET /docs` and ReDoc at `GET /redoc`. A deployed service and its documentation can therefore never disagree.
- Generated code is committed, lives in a directory named `gen` or `oas`, carries a "do not edit" header, and is never edited by hand.

## 2. URLs and resources

| Rule | Example | Enforced by |
|---|---|---|
| Nouns, plural, kebab-case path segments | `/order-items`, `/orders/{orderId}/items` | `paths-kebab-case` |
| No trailing slash, no file extensions, no verbs | `/orders` not `/orders/`, `/getOrders`, `/orders.json` | `no-path-trailing-slash` |
| Path parameters are camelCase and declared | `/orders/{orderId}` | `path-params-defined` |
| Nest at most one level | `/orders/{orderId}/items` yes; `/customers/{id}/orders/{id}/items` no, use `/orders/{orderId}/items` | review |
| Actions that are not CRUD are sub-resources with POST | `POST /orders/{orderId}/ship`, `POST /reports/daily/exports` | review |
| Query strings only for filtering, sorting, paging, field selection | `?status=open&sort=-createdAt&limit=50` | `path-not-include-query` |
| Every operation has a unique camelCase `operationId` | `listOrders`, `getOrder`, `createOrder`, `shipOrder` | `operation-operationId*` |
| Every operation has a `summary` and a tag that is defined at the top level | | `operation-summary`, `operation-tag-defined` |

Naming the `operationId` matters: generators turn it into method names in every client (`client.ShipOrder(...)`, `api.POST("/orders/{orderId}/ship")`).

## 3. HTTP methods and status codes

| Method | Use | Idempotent | Success code |
|---|---|---|---|
| `GET` | Read a resource or collection. Never changes state. | Yes | `200` |
| `POST` | Create a resource, or run an action | No (unless `Idempotency-Key`) | `201` with `Location` for creation; `200` or `202` for actions |
| `PUT` | Replace a resource completely | Yes | `200` or `204` |
| `PATCH` | Partial update with JSON Merge Patch (`application/merge-patch+json`) | Yes by design of merge patch | `200` |
| `DELETE` | Remove a resource | Yes | `204`; a repeated DELETE returns `404` |

Status codes we use, and what they mean to a client:

| Code | Meaning | Body |
|---|---|---|
| `200` | OK | resource or collection |
| `201` | Created; `Location` header points at it | the created resource |
| `202` | Accepted; work continues asynchronously | a status resource with a `Location` to poll |
| `204` | Done, nothing to return | none |
| `304` | Not modified (conditional GET) | none |
| `400` | Request does not match the contract (syntax, types, enums, lengths, unknown fields) | Problem |
| `401` | Not authenticated | Problem, plus `WWW-Authenticate` |
| `403` | Authenticated but not allowed | Problem |
| `404` | No such route or resource; also used instead of 403 when existence must not leak | Problem |
| `405` | Method not supported on this path; `Allow` header lists the supported ones | Problem |
| `409` | Conflict with current state (duplicate, version mismatch, already shipped) | Problem |
| `412` | `If-Match` precondition failed (optimistic concurrency) | Problem |
| `415` | Unsupported `Content-Type` | Problem |
| `422` | Well-formed but semantically invalid (business rule) | Problem with `errors[]` |
| `429` | Rate limited; `Retry-After` header | Problem |
| `500` | Bug or unexpected failure; cause is logged, never returned | Problem |
| `503` | Dependency down, shutting down; `Retry-After` when known | Problem |

Rules:

- `GET` and `DELETE` have no request body. `POST`, `PUT`, `PATCH` bodies are JSON objects, never bare arrays or scalars (an object can grow; an array cannot).
- Never return `200` with an error inside. Never return `500` for a client mistake.
- Do not invent status codes or use rare ones (`418`, `451`) without a reason recorded in the contract description.

## 4. Errors: RFC 9457 Problem Details, always

Every 4xx and 5xx response, from every service, in every language, is `application/problem+json` with the `Problem` schema:

```json
{
  "type": "about:blank",
  "title": "Unprocessable Content",
  "status": 422,
  "detail": "cannot ship an order that has no items",
  "instance": "/orders/42/ship",
  "errors": [
    { "pointer": "/items", "detail": "must contain at least one item" }
  ]
}
```

- `title` and `status` are required. `title` is the same for every occurrence of a problem type; `detail` is specific to the occurrence.
- `type` is `about:blank` when the status code says everything. For errors a client must handle specially (insufficient stock, version conflict), define a problem type URI such as `/problems/insufficient-stock`, document it in the contract description, and keep the list short. Clients branch on `type`, never on `detail` text.
- `instance` is the request path. Servers fill it automatically.
- `errors[]` (extension member) carries field-level validation failures: `pointer` is a JSON Pointer (RFC 6901) into the request body, or `query:name` / `header:x-request-id` / `path:orderId` for parameters.
- `detail` never contains stack traces, SQL, file paths or internal hostnames. A `500` says "internal error" and the real cause goes to the log with a correlation id.
- Error responses carry `Cache-Control: no-store`.
- Go services use the shared `problem` package. Equivalent helpers exist in other stacks (ASP.NET `ProblemDetails`, Spring `ProblemDetail`, Symfony / API Platform `ApiProblem`, Laravel `spatie/laravel-problem-details` or a 30-line class).

Enforced by: the `rule/error-responses-are-problem-json` configurable rule in `redocly.yaml`; the contract test that validates real responses against the schema.

## 5. Request and response bodies

- JSON, UTF-8, `Content-Type: application/json`. Requests with another type get `415`.
- Property names are camelCase (`createdAt`, `orderId`). Enforced by `rule/schema-properties-camel-case`.
- Timestamps are RFC 3339 strings in UTC with a `Z` suffix: `2026-11-01T14:03:00Z`. Dates without time use `YYYY-MM-DD`. Never Unix epochs, never local time.
- Identifiers are strings in the contract even when the database uses integers, so they can change representation without breaking clients. Money is `{ "amount": "12.50", "currency": "EUR" }` with the amount as a decimal string, never a float.
- Enumerations are lowercase strings (`open`, `shipped`). Clients MUST treat unknown enum values as "other", because enums grow (see section 8).
- Request objects set `additionalProperties: false` so typos are rejected with `400`; response objects do not, so clients tolerate new fields.
- Nullability is explicit: a field that may be absent is optional; a field that is present but may be `null` is declared `nullable: true`. Avoid both at once.
- Booleans are `is`/`has`-free: `active`, not `isActive`.
- No envelopes around single resources (`{ "data": { ... } }`). Collections do have an envelope (next section).

## 6. Collections: pagination, filtering, sorting, field selection

Collections are returned as an object with `items` and paging metadata, so new metadata can be added later:

```json
{ "items": [ ... ], "nextCursor": "eyJpZCI6NDJ9", "total": 1234 }
```

- **Pagination** is cursor-based by default: `?limit=50&cursor=...`. `limit` has a documented default (50) and maximum (200); larger values are clamped, not rejected. `total` is optional and may be omitted when it is expensive. Offset pagination (`?offset=`) is allowed only for admin grids over small, stable tables, and the contract must say so.
- **Filtering** uses one query parameter per field: `?status=open&customerId=7`. Multiple values are repeated parameters (`?status=open&status=paid`). Range filters use suffixes: `?createdAtFrom=...&createdAtTo=...`. No query DSLs in query strings.
- **Sorting** uses `?sort=field` or `?sort=-field` for descending; several fields are comma-separated. The contract lists the sortable fields.
- **Field selection** (sparse fieldsets) uses `?fields=id,status,customer.name`; **embedding related resources** uses `?include=customer,items`. Both are optional per endpoint and must be declared in the contract when supported. These two parameters are our answer to the "over-fetching" argument and to composite admin screens; use them before creating a screen-specific endpoint.
- **Composite view endpoints** (`GET /views/order-dashboard`) are allowed when a screen needs data from several resources and `include=` would be slow. They are tagged `views`, documented as read-only, and counted against trigger T3 in ADR-0001.

## 7. Versioning and compatibility

- **Additive only.** Adding optional fields, optional parameters, new endpoints, new enum values and new problem types is allowed at any time. Clients MUST ignore unknown fields and unknown enum values.
- **Breaking changes are not made.** Removing or renaming a field, path or parameter, changing a type, making an optional input required, narrowing an enum, changing the meaning of a field. If one is unavoidable, add the new form alongside the old one, mark the old one `deprecated: true` with a `description` saying what replaces it and the earliest removal date, keep both for at least two release cycles, and remove the old one only after access logs show no callers.
- **No URL versioning for internal APIs** (`/v1/`). We deploy every client ourselves, backend first, UI second, so additive evolution is enough. If an API is ever exposed to third parties, that API gets a date-based or `/v{n}/` version, recorded in its own ADR.
- **Deploy order:** backend before frontend, always. A UI that asks for a field the backend does not have yet is the one failure mode additive evolution does not cover.
- Enforced by: `oasdiff breaking --fail-on ERR` against the base branch in CI. A pull request that needs to pass despite a reported breaking change must explain why in its description and get a second reviewer.

## 8. Generated clients only

- Other Go services call an API through the client generated from its contract (`gen.NewClientWithResponses`). The web UI uses TypeScript types generated with openapi-typescript and calls through openapi-fetch. .NET uses Kiota or NSwag, Java uses OpenAPI Generator or the Spring `RestClient` with generated models, PHP uses OpenAPI Generator, C++ uses OpenAPI Generator's `cpp-restsdk` target or a thin libcurl wrapper over generated models.
- No hand-written request construction (`fetch("/api/orders/" + id)`, `http.NewRequest(...)`) in application code. If the generated client cannot express a call, fix the contract or the generator config.
- Generated clients are regenerated in CI and compared with the committed copy, so a contract change that affects a caller is visible in the same pull request.
- Clients send `Accept: application/json`, a `User-Agent` naming the calling service, and propagate `traceparent` (W3C Trace Context) and a request id.

## 9. Security

- Every production contract declares a `securitySchemes` entry (bearer JWT from our identity provider for the UI, mTLS or service tokens between services) and applies it at the top level with `security:`. Health probes opt out with `security: []`. The `security-defined` lint rule is switched to `error` in real services; it is off only in this sample.
- Authorization is enforced per route (method + path) in middleware, so the authorization matrix is readable from the contract.
- Request validation against the contract runs before any handler, so handlers never see invalid input.
- Limits: request body size, string lengths (`maxLength` on every string), array lengths (`maxItems`), pagination `limit`. Every string in a request schema has a `maxLength`.
- Response headers: `Cache-Control` set explicitly on every endpoint (`no-store` for errors and personal data, `private, max-age=...` or `ETag` for cacheable reads), `X-Content-Type-Options: nosniff`.
- No secrets, tokens or personal data in URLs or query strings; they end up in logs.

## 10. Operations

- `GET /healthz` (liveness, always `200` while the process runs) and `GET /readyz` (readiness, `503` with a Problem when a dependency is down). Both are unauthenticated and excluded from access logs if noisy.
- Every response carries the request id (`X-Request-Id`, echoed from the request or generated) and is logged with method, path template (`/orders/{orderId}`, not the concrete id), status, duration and request id.
- Timeouts on the server (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) and on every outgoing client call. Graceful shutdown on SIGTERM.
- Rate limiting and body-size limits at the edge (gateway or reverse proxy) by method and path.

## 11. Tooling and CI (what this repository implements)

| Check | Tool | Where |
|---|---|---|
| Contract is valid and follows this guide | Redocly CLI with `redocly.yaml` | `make lint`, CI job `contract` |
| No breaking change versus the base branch | oasdiff | `make breaking`, CI job `breaking` |
| Human-readable change list in the PR summary | oasdiff changelog | CI job `breaking` |
| Generated Go code is current and untouched | `go generate` + `git diff --exit-code` | `make check-generated`, CI job `go` |
| Real responses match the contract | kin-openapi response validation in tests | `go test`, CI job `go` |
| Errors are RFC 9457 | `problem` package tests + configurable lint rule | CI jobs `go` and `contract` |
| TypeScript client is current | openapi-typescript + `git diff --exit-code` | `make ts-client`, CI job `ts-client` |
| Both sample contracts identical | `cmp` | CI job `contract` |

Tool versions are pinned in `Makefile` and `.github/workflows/ci.yml` and bumped deliberately.

## 12. Pull request review checklist

Copy into the PR template.

- [ ] The change starts with the contract; generated code was produced by `go generate`, not edited.
- [ ] `make lint` passes with no new warnings. New endpoints have `operationId`, `summary`, tag, a `4xx` and a `default` response.
- [ ] `make breaking` reports no breaking change, or the PR explains why one is unavoidable and how clients were migrated.
- [ ] New request strings have `maxLength`; new arrays have `maxItems`; new collections use `limit` + cursor.
- [ ] Every error path returns a `Problem`; no `500` for client mistakes; no internal detail in `detail`.
- [ ] Timestamps are RFC 3339 UTC; ids are strings; money is decimal strings; enums are lowercase.
- [ ] Any new composite endpoint or `include=` relation is justified in the PR and counted against ADR-0001 trigger T3.
- [ ] Clients (Go services, TypeScript UI) were regenerated and compile.
- [ ] Docs at `/docs` show the new operation with a working example.

## References

- OpenAPI 3.0.3 specification: https://spec.openapis.org/oas/v3.0.3
- RFC 9457 Problem Details for HTTP APIs: https://www.rfc-editor.org/rfc/rfc9457
- RFC 9110 HTTP Semantics (methods, status codes, `Allow`): https://www.rfc-editor.org/rfc/rfc9110
- RFC 9111 HTTP Caching: https://www.rfc-editor.org/rfc/rfc9111
- RFC 7396 JSON Merge Patch: https://www.rfc-editor.org/rfc/rfc7396
- RFC 6901 JSON Pointer: https://www.rfc-editor.org/rfc/rfc6901
- Redocly CLI rules: https://redocly.com/docs/cli/rules
- oasdiff: https://github.com/oasdiff/oasdiff
- oapi-codegen: https://github.com/oapi-codegen/oapi-codegen
- openapi-typescript and openapi-fetch: https://openapi-ts.dev/
- Zalando RESTful API Guidelines (a mature public style guide many of these rules follow): https://opensource.zalando.com/restful-api-guidelines/
- Microsoft REST API Guidelines: https://github.com/microsoft/api-guidelines
