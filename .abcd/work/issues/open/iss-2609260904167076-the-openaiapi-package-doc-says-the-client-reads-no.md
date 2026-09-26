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
---

The openaiapi package doc says the client reads no environment, but its nil Transport is net/http's DefaultTransport, which honours HTTPS_PROXY, HTTP_PROXY and NO_PROXY, so the doc misstates the adapter's network path.
