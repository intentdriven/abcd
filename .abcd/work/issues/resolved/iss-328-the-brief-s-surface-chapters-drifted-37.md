---
schema_version: 1
id: "iss-328"
slug: "the-brief-s-surface-chapters-drifted-37"
severity: "minor"
category: "drift"
source: "user-observation"
found_during: "v0.6.1 release gate"
found_at: ".abcd/development/brief/04-surfaces"
resolution: "Chapter-by-chapter brief-drift sweep: all 37 receipt findings verified against the current tree and binary, then fixed across the 04-surfaces chapters, the surfaces README, and the 01-agents/03-configuration/08-skills internals chapters"
impact: internal
resolved_by:
  commit: "d475190"
---

The brief's surface chapters drifted 37 findings from the shipped reality (v0.6.1 release-gate crosscheck, full tier): 14 false claims, 12 undocumented surfaces, 10 stale counts, 1 fictional layout — including a pre-rename 'intent review ingest' spelling in 05-intent.md, the ahoy dry-run envelope missing its banlist key, an undocumented --allow-stale-binary install flag, and the launch dry-run's citation gate absent from its gate roster. The full where/claim/reality list lives in the iss35 receipt committed beside the v0.6.1 cut under .abcd/work/reviews/; a sweep should fix them chapter by chapter, citing the receipt