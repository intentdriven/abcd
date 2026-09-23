---
schema_version: 1
id: "iss-2608221342509701"
slug: "the-screenshot-audit-installs-playwright-by-pinned-version-t"
severity: "nitpick"
category: "future-work-seed"
source: "user-observation"
found_during: "agent-finding"
found_at: ".github/workflows/site-screenshots.yml"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: a committed playwright lockfile needs the dependency gate and changes wrangler-action's package-manager inference). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

the screenshot audit installs playwright by pinned version through npx with no lockfile; a committed lockfile would harden it but changes wrangler-action's package-manager inference and needs the dependency gate