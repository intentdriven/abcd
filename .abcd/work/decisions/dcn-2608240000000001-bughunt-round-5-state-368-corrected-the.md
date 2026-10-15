---
id: dcn-2608240000000001
slug: bughunt-round-5-state-368-corrected-the
date: 2026-08-24
---
2026-08-24 — bughunt round 5 (state #368): corrected the shipped-intents
bucket README drift — the false "directory is empty" claim (18 shipped
intents present; the parent index already reads populated) and the retired
`intent-fidelity-reviewer` name (→ `intent-auditor`, spc-28; the sibling
front-door the iss-334 sweep of the parent index missed).
iss-2608240815458212, iss-2608240815453131. A concurrent second firing of the
hunt-a routine took the `bughunt-a/round-5` branch and PR #480 (planned/
bucket README) at the same time; these disjoint shipped/ findings ship on
`bughunt-a/round-5-2` rather than clobber a live peer's branch. The persona
nitpick in the abcdev-site prompt was dropped (that firing logged it
considered-and-rejected as a frozen historical artefact); the one code
candidate — `frontmatter.Fields` duplicate-key first-wins — was refuted
(every lint status read shares the one first-wins map, first-wins is the
documented contract, the exemption was already audited in iss-39).
