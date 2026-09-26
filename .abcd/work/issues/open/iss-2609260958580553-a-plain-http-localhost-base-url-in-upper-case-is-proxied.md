---
schema_version: 1
id: "iss-2609260958580553"
slug: "a-plain-http-localhost-base-url-in-upper-case-is-proxied"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: review2-apiadapter"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/openaiapi/client.go"
---

The OpenAI-compatible client admits a plain-http base URL to LOCALHOST (or any case other than lower) through its case-insensitive loopback check, but net/http's proxy exclusion compares the host with localhost case-sensitively, so with HTTP_PROXY set the call is proxied and the bearer key crosses the proxy in cleartext, against the package doc and the adapters chapter, which say a call to this machine is never proxied.
