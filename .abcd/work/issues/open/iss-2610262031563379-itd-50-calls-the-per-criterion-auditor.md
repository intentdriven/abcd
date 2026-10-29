---
schema_version: 1
id: "iss-2610262031563379"
slug: "itd-50-calls-the-per-criterion-auditor"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 199 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25"
remedy: "In itd-50, replace the retired name intent-fidelity-reviewer with intent-auditor (press release, Why This Matters, Audit Notes placeholder, Related)."
related_intents: [itd-50]
finding_ref: cfk-8366953637717750
---

itd-50 calls the per-criterion auditor intent-fidelity-reviewer; the brief names the same role intent-auditor (Role 1).

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "the `intent-fidelity-reviewer` records a `NOT_MET` verdict"
- End B: `.abcd/development/brief/01-product/01-press-release.md:26` — "`/abcd:intent audit` (Role 1 of `intent-auditor`)"

itd-50's press release says 'the `intent-fidelity-reviewer` records a `NOT_MET` verdict'. The brief's press release names the same role as '`/abcd:intent audit` (Role 1 of `intent-auditor`)'. That is two names for one agent role, and work built from the planned intent would look for an agent the brief does not name.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 199.
