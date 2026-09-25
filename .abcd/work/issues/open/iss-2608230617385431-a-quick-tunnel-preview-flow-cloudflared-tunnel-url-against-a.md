---
schema_version: 1
id: "iss-2608230617385431"
slug: "a-quick-tunnel-preview-flow-cloudflared-tunnel-url-against-a"
severity: "nitpick"
category: "future-work-seed"
source: "user-observation"
found_during: "user-observation"
found_at: "site-src"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Close the quick-tunnel preview protocol seed unless the need recurs, or plan it now?"
---

a quick-tunnel preview flow (cloudflared tunnel --url against a local site build) covers phone testing when the LAN route is blocked and multiple simultaneous testers during dev work; worth a documented protocol or make target if the need recurs — the URL is public while the tunnel runs, so the protocol must say what may be served through it