---
schema_version: 1
id: "iss-2610120921425348"
slug: "running-a-source-checkout-s-abcd-through"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "cloud-candidate survey, 2026-10-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/staleusage.go"
remedy: "Before naming a PATH copy stale, recognise a binary built from a source checkout of abcd (go run's build cache, or the working directory being such a checkout) and say the flag is not in this build, without update advice"
resolution: "An unstamped dev build outside every plugin root (go run of a source checkout) is told the plugin surface documents the thing and this build lacks it, with no update advice; only a release-stamped copy keeps the PATH-copy remedy."
impact: fix
resolved_by:
  commit: "033e7cd914c54f35ecfdf6f7ee11abd8cbadebbc"
---

Running a source checkout's abcd through go run and typing a flag the verb does not have (go run ./cmd/abcd build next --dry-run) says 'this binary predates --dry-run the plugin surface it was provisioned from documents; this PATH copy is stale; run abcd update'. The binary is a fresh go-run build of the checkout, not a PATH copy, and abcd update would change nothing: staleUsageNote (internal/surface/cli/staleusage.go) takes its default branch for any executable outside the plugin root.
