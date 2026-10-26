---
schema_version: 1
id: "iss-2610262043093810"
slug: "itd-109-and-itd-50-both-define-how-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1118 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-109-a-two-part-verification-suite-for-abcd.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-109, itd-50]
finding_ref: cfk-de2212f476ae8152
---

itd-109 and itd-50 both define how a human's verification of an intent's criteria is recorded before shipping, as a receipt and as not a receipt.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-109-a-two-part-verification-suite-for-abcd.md:23` — "a human walks them, and their verdicts land in the"
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:101` — "in the product thinker's words, not a separate receipt"

itd-109 has a human walk the criteria that cannot be automated, with 'their verdicts land in the same receipt through the same verb', and shipping an intent reads those receipts and refuses on a pending manual verdict. itd-50 rules that 'hand verification is a grounds entry in the product thinker's words, not a separate receipt', taken after the machine criteria read MET. Both records claim the same moment and the same act and record it in incompatible forms, so the ship gate cannot read both.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1118.
