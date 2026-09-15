---
schema_version: 1
id: "iss-2609012037440084"
slug: "the-docs-cite-refresh-fetcher-in-cite-fetch-go-re-guards-the"
severity: "minor"
category: "security"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/cite/fetch.go"
---

The docs cite refresh fetcher in cite/fetch.go re-guards the host per redirect hop through urlguard.CheckHostWith but pins no scheme, so an https link that redirects to plaintext http is followed — the same shape GHSA-35fj-9w6f-7h62 closes in memory ingest. It fetches documented links for liveness and stores no content, so it sits outside that advisory and is recorded here instead of fixed. A fix would refuse a hop whose URL scheme is not https before the host guard, as update.go and memory ingest do.
