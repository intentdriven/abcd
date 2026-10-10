---
schema_version: 1
id: "iss-2608271858577043"
slug: "da001-first-cut-enforced-position-not"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "adversarial security review of the structural-review sweep (2026-08-27)"
found_at: "scripts/check-decisions-append.sh"
resolution: "the round's fixes rode the same change: DA002/DA003/DA004 and the reader/gate slug split close every reproduced bypass; merge_group BASE_SHA gap stays flagged in iss-form"
impact: internal
resolved_by:
  commit: "154af92a"
---

adversarial review round over the structural-review sweep (assembled diff, pre-merge): the DA001 append-only gate as first written enforced position but not preservation — a tail-reaching rewrite of committed decisions passed clean (reproduced on a copy of the live ledger), merge commits were skipped wholesale so a forged merge resolution landed unchecked (reproduced), and a non-canonical list bullet slipped the header exemption (reproduced); its refusal output echoed attacker bytes unsanitised, and seeding an empty ledger was refused. The new capture-side slug invariant was reader-only, so a drifted record turned into a silent skip instead of a red gate. Fixes ride the same change; flagged-not-fixed sibling: on merge_group events BASE_SHA is empty for ALL three diff-ranged gates (RS001/RS002/DA001), so the merge-queue entry does not run them although AGENTS.md claims it runs the lot — pre-existing pattern, needs its own decision.