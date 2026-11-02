# ADR-0001: REST with OpenAPI for all service APIs

- **Status:** Accepted
- **Date:** 2026-10-30
- **Deciders:** Software development team (team meeting, 2026-10-30)
- **Supersedes:** none
- **Related:** [ADR-0002](0002-oapi-codegen-for-go-services.md), [REST hygiene guide](../rest-hygiene.md)

## Context

We needed one API style for two kinds of traffic:

1. **Backend to backend**: services calling each other synchronously.
2. **Client to backend**: one administrator / operator web UI. There is no mobile client and no third-party consumer.

Backend services are written in PHP, Go, .NET, Java and C++. The product is at an early stage, the team is small, and the same team owns both the UI and the backend.

Two options were evaluated in a research pack prepared for the meeting (REST vs GraphQL, documents 00 to 09): REST described by OpenAPI, and GraphQL. Some team members proposed GraphQL for every API.

Key findings that drove the decision:

- GraphQL's defining benefit, letting the client choose the response shape without backend changes, pays off when many clients need many different data shapes and cannot cheaply change the backend. With one UI built by the same team, neither condition holds.
- Between services the data shape per call site is fixed, so GraphQL's benefit is zero while its per-request cost (parse, validate, per-field resolve, dynamic serialise) and its loss of HTTP semantics (method and path routing, status codes, caching, gateway policy) remain. GraphQL vendors themselves place GraphQL at the client edge and recommend gRPC or REST between services.
- REST is first-class in all five of our languages, including C++, where GraphQL support is thin. OpenAPI code generation is mature everywhere; GraphQL client generation outside TypeScript is uneven.
- REST endpoints have bounded cost by construction and get HTTP caching for free. GraphQL needs DataLoader, complexity limits, persisted queries and introspection control before the first production deploy.
- A GraphQL layer can be added over REST services later at roughly today's cost. The reverse is not true. REST is the reversible choice.

## Decision

1. **All synchronous APIs are REST, described by an OpenAPI 3.0 document that is the single source of truth.** This applies to backend-to-backend calls and to the web UI.
2. **Server stubs, models and clients are generated from the contract.** No hand-written HTTP clients. The web UI compiles against TypeScript types generated from the same document.
3. **Every service follows the [REST hygiene guide](../rest-hygiene.md)**: resource naming, HTTP method and status code usage, RFC 9457 problem details for every error, pagination and filtering conventions, additive-only evolution.
4. **CI enforces the guide**: Redocly lint with the shared ruleset, oasdiff breaking-change detection against the base branch, a check that committed generated code matches the contract, and a check that the generated TypeScript client is current.
5. **GraphQL is not adopted now.** A GraphQL layer over the REST services will be evaluated when two or more of the following triggers hold:
   - T1: a second client with materially different data needs (mobile app, partner portal, public API) ships or is committed;
   - T2: a separate frontend team exists and is measurably blocked waiting for backend endpoint changes;
   - T3: the REST API has grown past roughly 30 screen-specific or `include`-heavy composite endpoints that are hard to keep consistent;
   - T4: we expose the API to third parties whose data shapes we cannot predict;
   - T5: over-fetching or request waterfalls show up as a measured user-facing latency problem that `include=` / `fields=` parameters cannot fix.
6. **gRPC may be evaluated** for high-volume Go / .NET / Java / C++ service pairs. That evaluation gets its own ADR.

This ADR is reviewed at each planning cycle or when a trigger fires, whichever comes first.

## Consequences

Positive:

- One API style, one error format, one documentation tool, one set of CI checks for every service in every language.
- Any developer, operator or script can call any service with curl, the browser address bar or PowerShell.
- Authorization, rate limits and observability work by method and path in existing proxies and gateways.
- No new runtime infrastructure (GraphQL gateway, schema registry, GraphQL-aware observability).

Negative, and how we mitigate them:

- Composite screens may need several requests or purpose-built view endpoints. Mitigation: `include=` and `fields=` parameters defined once in the hygiene guide; composite "view" endpoints are allowed and counted against trigger T3.
- REST APIs drift in style without discipline. Mitigation: the lint ruleset and the review checklist in the hygiene guide.
- We do not get GraphQL's field-level usage analytics for deprecation decisions. Mitigation: additive-only changes, deprecation markers in OpenAPI, and access-log analysis by path.

## Reference implementation

The repository `go-openapi-contract-first` shows the full workflow for a Go service: contract, generation with two generators (compared in ADR-0002), RFC 9457 errors, lint and breaking-change checks in CI, generated Go and TypeScript clients.
