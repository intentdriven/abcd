---
schema_version: 1
id: "iss-2608230748418054"
slug: "ideate-leg-2-grill-has-a-structural-blind-spot-for-id-less-r"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "ideate run abcd-research-verb"
found_at: "commands/ideate.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Which root cause to fix: an id family for research notes, a placement rule, or a grill-side sweep?"
---

Ideate leg-2 grill has a structural blind spot for id-less records: the 2026-08-23 abcd-research-verb run's grill failed to surface the same-day research note (2026-08-22-sota-research-protocol.md) that answers the submitted idea by name, because research notes carry no citable record id and the grill's output is id-cited hits. The nearest-citable-record note-field rule exists but nothing prompts a sweep of research/notes/ or DECISIONS.md, so a standing verdict recorded outside the id-bearing record families is invisible to the leg that exists to prevent re-litigation. The leg-3 evaluator caught it only by doing its own sweep.

Decision to make before fixing (maintainer's gate): should research notes and other id-less artefacts be pulled into record-bearing ids, so deterministic tooling can address them, or should the placement rule change instead? Three candidate resolutions, mutually exclusive at the root:

1. Mint an id family for research notes (rsn-N or similar). Pro: the grill becomes deterministic over them; typed cross-record links (supersedes/reverses/duplicates/refines, per itd-84) become possible for notes, which today cannot be typed-linked at all; standing verdicts and revisit conditions inside notes become mechanically discoverable. Con: id overhead on a deliberately cheap, high-volume genre (14 sota notes in 7 weeks); a fifth id family to wire through resolution, lint, and the status board.

2. Placement rule, not id scheme: a verdict or revisit condition that gates future action must not live only in a note — it gets promoted at write time into an id-bearing record (an issue wontfix, an ADR, or a DECISIONS pointer that the grill sweeps). Pro: no new family; targets exactly the failure (it was the *verdict* that was invisible, not the note). Con: relies on a discipline at write time, which is the same class of unenforced convention this ledger exists to catch; and DECISIONS.md pointers already existed for this very verdict yet the grill does not sweep DECISIONS.md either, so this option still requires widening the grill's sweep.

3. Grill-side only: prompt leg 2 to sweep research/notes/ and DECISIONS.md in addition to id-cited hits, reporting id-less findings in the note field of the nearest citable record. Pro: smallest change, no schema or discipline cost. Con: keeps the citation one-directional and untyped; the binary still cannot refuse an unresolvable note reference, so the determinism gain is partial.

Interim mitigation regardless of the decision: option 3's sweep goes into the leg-2 prompt now, since it is compatible with all three roots. Applied 2026-08-23 (commands/ideate.md leg-2 section + CHANGELOG entry). This issue stays open to park the root-cause decision between the three options above; resolving it means picking a root, not the applied sweep.