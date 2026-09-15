---
schema_version: 1
id: "iss-2609012037433158"
slug: "ghsa-35fj-9w6f-7h62-cwe-319-cwe-311-isurl-admits-http-as-wel"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/memory/ingest.go"
resolution: "memory ingest now admits a local path or an https URL only: acquireSource refuses a URL-shaped source of any other scheme before the fetcher seam is reached, naming the scheme it wants and the one it got with userinfo masked, isURL classifies https alone, defaultFetch re-states the rule where the connection is made, and the redirect policy — lifted out as ingestRedirectPolicy so it is testable without a network — refuses a hop that leaves https ahead of the SSRF host guard, mirroring update.go newUpdater. TestIngestRefusesPlaintextHTTP pins the refusal, the unreached fetcher, the masked userinfo and both admitted controls (https and a local path); TestIngestRedirectMustStayHTTPS pins the per-hop pin, the masked redirect target, an https hop still followed, the hop budget and the SSRF guard. The http:// fixtures in TestIngestRefusesSSRFTargets and TestIngestEncodesHiddenRunesInFetchedOrigin moved to https:// so the SSRF guard and the origin encoder stay the thing under test rather than passing on the scheme refusal. Out of scope, named as siblings: cite/fetch.go's docs cite refresh fetcher re-guards the host per hop with no scheme pin (iss-2609012037440084); vintage/release.go never follows a redirect; lint/citations.go accepts http for link validation, not a fetch. The --allow-http admission alternative stays open as iss-2609012037443635."
impact: fix
---

GHSA-35fj-9w6f-7h62 (CWE-319, CWE-311): isURL admits http as well as https and acquireSource fetches either, and the CheckRedirect of defaultFetch re-checks only the redirect count and the SSRF host guard, so a hop from https to plaintext http is followed — whereas update.go newUpdater refuses a redirect that left https. A man-in-the-middle on a plaintext source can rewrite its text and plant a License: header the store copies into provenance. The fix must refuse a non-https source in acquireSource before any fetcher runs, with a refusal that names the scheme, pin the scheme per hop in the redirect policy the way abcd update does, and move the http:// test fixtures (TestIngestRefusesSSRFTargets, TestIngestEncodesHiddenRunesInFetchedOrigin) to https:// so the SSRF guard stays the thing under test rather than passing on the scheme refusal.

## Grounds

- pursued: the store copies a fetched source's text and the licence header lifted out of it verbatim into durable provenance, so a plaintext transport lets a man-in-the-middle choose what the store remembers and under what licence; https-only with no escape is the posture abcd update already holds, and an --allow-http escape is additive later
