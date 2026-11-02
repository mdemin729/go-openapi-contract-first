# TypeScript client for the web UI

`hello-api.d.ts` is generated from `oapi-codegen/api/openapi.yaml` by
[openapi-typescript](https://openapi-ts.dev/) and committed. CI regenerates it
and fails if it differs, so the UI always compiles against the current contract.

Regenerate after changing the contract:

```bash
make ts-client
```

## Using it in the UI

Pair the types with [openapi-fetch](https://openapi-ts.dev/openapi-fetch/), a
2 kB typed `fetch` wrapper. Paths, parameters, bodies and every declared status
code are type-checked; a renamed field is a compile error in the UI, not a
runtime surprise.

```ts
import createClient from "openapi-fetch";
import type { paths } from "./hello-api";

const api = createClient<paths>({ baseUrl: "/api" });

const { data, error, response } = await api.POST("/greetings", {
  body: { name: "Ana", language: "es" },
});

if (error) {
  // `error` is typed as the Problem schema (RFC 9457) from the contract
  showToast(`${error.title}: ${error.detail ?? ""}`);
  for (const e of error.errors ?? []) highlightField(e.pointer, e.detail);
} else {
  // `data` is typed as Greeting
  console.log(data.message, response.status);
}
```

With TanStack Query:

```ts
useQuery({
  queryKey: ["hello", name],
  queryFn: () => api.GET("/hello", { params: { query: { name } } }).then(r => r.data!),
});
```

Rules (from [docs/rest-hygiene.md](../../docs/rest-hygiene.md)):

- Never hand-write `fetch("/api/...")` calls. If a path or field is not in
  the generated types, change the contract first.
- Parse errors with the `Problem` type only; do not special-case status codes
  with string matching on the body.
