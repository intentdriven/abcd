---
schema_version: 1
id: "iss-2608221342509701"
slug: "the-screenshot-audit-installs-playwright-by-pinned-version-t"
severity: "nitpick"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-finding"
found_at: ".github/workflows/site-screenshots.yml"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Sign off a committed playwright lockfile (the dependency gate; it changes wrangler-action's package-manager inference)?"
---

the screenshot audit installs playwright by pinned version through npx with no lockfile; a committed lockfile would harden it but changes wrangler-action's package-manager inference and needs the dependency gate