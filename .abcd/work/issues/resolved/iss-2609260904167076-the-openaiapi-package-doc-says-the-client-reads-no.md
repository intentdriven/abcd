---
schema_version: 1
id: "iss-2609260904167076"
slug: "the-openaiapi-package-doc-says-the-client-reads-no"
severity: "nitpick"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/client.go"
resolution: "The package doc and the adapters chapter say the client honours net/http's proxy variables and trust roots, and why the key stays inside TLS; the transport is left as it is, since no recorded decision forbids a proxy."
impact: internal
resolved_by:
  commit: "ed08d993"
---

The openaiapi package doc says the client reads no environment, but its nil Transport is net/http's DefaultTransport, which honours HTTPS_PROXY, HTTP_PROXY and NO_PROXY, so the doc misstates the adapter's network path.

## Grounds

- pursued: the adapter's documented network path matches its transport; shown wrong if the client is given a transport that ignores the proxy variables, or a Decision rules the adapter must not be proxied, while the doc still says otherwise
