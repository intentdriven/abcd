---
schema_version: 1
id: "iss-2610312044545639"
slug: "gitutil-runwithindex-and-gitutil"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "landing PR #1301 (run-2610310228179894, lane-1)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gitutil/index.go"
remedy: "Run both under a context with a bounded timeout, as bounded.go does, and have indexDigest fold a timed-out run into its uncopied value so the lane is handed back."
---

gitutil.RunWithIndex and gitutil.StreamWithIndex (internal/gitutil/index.go) run git with no deadline, unlike the bounded runner in internal/gitutil/bounded.go, so a stalled git on a scratch index holds the auto-plan's checkout digest, and the implement step taking it, with no limit.
