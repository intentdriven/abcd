---
schema_version: 1
id: "iss-2608261206505385"
slug: "site-check-runs-in-no-gate-before-the"
severity: "major"
category: "process"
source: "user-observation"
found_during: "bughunt-a/round-8"
found_at: ".github/workflows/ci.yml"
resolution: "the Makefile site-render recipe and the ci.yml render step now run site check over the built output before a release can publish."
impact: internal
resolved_by:
  commit: "fd914d40"
---

site check runs in no gate before the release publishes