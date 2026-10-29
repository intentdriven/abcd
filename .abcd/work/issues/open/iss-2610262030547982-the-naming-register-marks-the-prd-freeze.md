---
schema_version: 1
id: "iss-2610262030547982"
slug: "the-naming-register-marks-the-prd-freeze"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 60 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:162"
remedy: "Remove, or mark as superseded design, the 'grill step as the PRD-producing sub-verb' and 'later-phase PRD-validating + freezing front end' sections of 05-intent.md (about lines 478-500); they rest on superseded itd-27 and predecessor spc-3 machinery."
finding_ref: cfk-f263035fa0d035d2
---

The naming register marks the PRD-freeze vocabulary as machinery abcd neither ships nor plans; the intent chapter specifies it as a later-phase plan front end

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:162` — "Some names entered this register from the retired predecessor implementation and describe machinery abcd neither ships nor plans"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:485` — "The later-phase PRD-freeze front end for planning runs this ordered sequence"

04-naming marks frozen_content_hash, planning_attempt_id, prd_grandfathered and the planning-attempt modes as predecessor vocabulary for machinery abcd neither ships nor plans, while 05-intent sets out a later-phase PRD-validating and freezing front end for plan that computes frozen_content_hash and writes planning_attempt_id. The same machinery cannot be both unplanned and a later phase.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 60.
