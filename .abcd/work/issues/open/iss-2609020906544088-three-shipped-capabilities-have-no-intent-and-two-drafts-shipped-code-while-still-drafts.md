---
schema_version: 1
id: "iss-2609020906544088"
slug: "three-shipped-capabilities-have-no-intent-and-two-drafts-shipped-code-while-still-drafts"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "phase-8-planning-2026-09-02"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents"
deferred_after: "v0.11.1"
deferral_reason: "lane owed (lapsed-deferral triage, run A 2026-09-29): a record-ceremony lane of its own: ship itd-85 by hand and file shipped intents for the generated CLI reference, the surface_coverage rule and the brief-to-surface crosscheck, each noting it was delivered before the ceremony existed (the 2026-09-23 ruling carried by iss-240's wontfix). Still owed to the product thinker: may the run write the retrospective specs shipped/ requires, or will you run their planning step? itd-60 has since reached planned/ with spc-2609020903498198."
remedy: "Waits on the M5 follow-up (may the run write the retrospective specs shipped/ requires, or will the product thinker run the planning step): either way, file one shipped intent each for the generated CLI reference, the surface_coverage rule and the brief-to-surface crosscheck, and move itd-85 from drafts/ to shipped/ through a closed spec, each carrying an Audit Notes line saying it was delivered before the ceremony existed and naming the issue ids it shipped under. Prove it with record-lint clean and the board listing the four as shipped; itd-60 closes through its own spec, spc-2609020903498198."
---

Found while readying Phase 8: the generated CLI reference (docs/reference/cli/commands.md), the surface_coverage record-lint rule and the brief-to-surface crosscheck gate were all shipped under issue ids with no intent record, and itd-60 and itd-85 each shipped substantial code while sitting in drafts/ (itd-60's deterministic docs lint is live as abcd docs lint; itd-85's delivery is on main). Phase 8's expectation is that the record is the shipped state, and these are shipped capabilities the record does not name as intents, so a product thinker reading the brief and the intents together still cannot see them as delivered. Directions, none adopted: file a shipped intent per capability by hand with an audit note saying it was delivered before the ceremony existed; or list them in the Phase 8 file as pre-ceremony deliveries and leave the intents as is; or resolve the gap through the doc-fidelity gate itself once it exists, since it will name every surface the brief describes without an intent.

## Remedy grounds (2026-09-29)

- Why: ruling M5 (2026-09-23) chose to ship by hand, as iss-240's wontfix records; only who writes the specs is open, so the fix is the same under both answers.
- Rejected: listing the deliveries in the Phase 8 file alone, which leaves the intent store, the thing the board and the changelog read, silent about them.
