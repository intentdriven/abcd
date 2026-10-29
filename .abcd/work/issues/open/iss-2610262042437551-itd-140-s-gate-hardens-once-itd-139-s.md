---
schema_version: 1
id: "iss-2610262042437551"
slug: "itd-140-s-gate-hardens-once-itd-139-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1089 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:57"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Is itd-140 rule 3 (no repo-agnostic claim for the site build until a second sparse managed instance runs in CI) kept, with the fixture instance re-homed from the superseded itd-139 to itd-2610091010348892, or given up as decision 13 of itd-2610032150577708 ruled on 2026-10-04? Recommended: keep it and add the fixture instance as a criterion of itd-2610091010348892, because the 2026-10-14 ruling post-dates the give-up and promises the site for every opted-in project."
related_intents: [itd-140, itd-2610032150577708]
finding_ref: cfk-4ff1767f612a3683
---

itd-140's gate hardens once itd-139's fixture exists; the dashboard intent replaced itd-139 and gave the genericity promise up

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-140-the-generic-specific-boundary-of-the.md:57` — "hardens once itd-139's fixture instance exists"
- End B: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47` — "give up the promise. The record site and the dashboard are promised for abcd's own record only; itd-140 carries the ruling."

itd-140's armed-detector rung waits on itd-139's fixture instance, which it names as the genericity demonstration. itd-2610032150577708 rules that the dashboard replaces itd-139 and that the record site is promised for abcd's own record only, saying itd-140 carries that ruling, which itd-140 as written does not. The rung waits on a record that will not be built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1089.
