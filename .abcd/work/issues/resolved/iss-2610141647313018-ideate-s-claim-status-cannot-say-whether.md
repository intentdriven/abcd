---
schema_version: 1
id: "iss-2610141647313018"
slug: "ideate-s-claim-status-cannot-say-whether"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "lab lab-261014142726-9c4d759 (a hand-run of /abcd:ideate on one external write-up, 2026-10-14)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ideate/ideate.go"
remedy: "Deliver itd-2610141646454347: claim status separates quoted from verified, and the primary source has a checked shape."
related_intents: [itd-2610141646454347]
resolution: "claim status separates quoted from verified, a verified claim resting only on the idea's own origin source is refused, and the primary source has a checked shape"
impact: breaking
resolved_by:
  intent: "itd-2610141646454347"
  spec: "spc-2610142138047122"
  commit: "81baf3f6a"
---

ideate's claim status cannot say whether a claim was verified as quoted or as true: a write-up's own experience can only be checked as quotation, and the three statuses carry no such distinction; any non-empty string is accepted as a primary source (lab finding F-9).
