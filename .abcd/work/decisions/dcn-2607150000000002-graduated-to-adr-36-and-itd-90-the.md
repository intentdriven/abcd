---
id: dcn-2607150000000002
slug: graduated-to-adr-36-and-itd-90-the
date: 2026-07-15
---
2026-07-15 (post-M2 gate design) — Graduated to adr-36 and itd-90. The coverage
gate raised a lifecycle question the plan implied but never named: the coverage
report knows what to ask, but nothing said who answers a blank, when, or where —
and the person with the tool (facilitator) is rarely the person with the answer
(product thinker). Decision (adr-36): a blank is a durable, fillable object, not
a fill-now-or-lose snapshot; answering is decoupled from disembark/embark and runs
as its own async, environment-agnostic step over the coverage JSON; blanks carry a
`kind` (`extractable` = coverage debt abcd can fix, vs `human-owned` = personas/
mental-model, never derivable and framed as a prompt not a failure — the durable
form of the "personas is manual" gate call); and a filled blank is marked
`authored-by` (a person + date), structurally distinct from a grounded section's
`extracted-from` (a file), so an opinion never launders into a fact. Coverage
schema grows to v2 (fillable object); mapping.go gains a per-section `Kind`. itd-90
specifies the product-thinker-facing interview (draft). Boundary: distinct from
itd-86 cold-reading (which reviews for contradictions, denied context; the interview
answers questions, fed context — opposite direction).
