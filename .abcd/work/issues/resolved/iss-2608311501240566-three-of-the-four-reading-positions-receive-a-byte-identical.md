---
schema_version: 1
id: "iss-2608311501240566"
slug: "three-of-the-four-reading-positions-receive-a-byte-identical"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "end-to-end reading rehearsal, cold-reading cycle 1"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/include.go"
resolution: "Each position is handed its own committed entry, and the entry is the one configuration surface for a position's object set, its kinds and its window (maintainer's ruling of 2026-09-02). The three assembling positions' default item sets are pinned by digest and asserted distinct, and two assemblies of one entry produce byte-identical bundles, so what a position reads moves by a commit to the preset file and by nothing else."
impact: fix
resolved_by:
  intent: "itd-2609020625400445"
  spec: "spc-2609020626048722"
---

Three of the four reading positions receive a byte-identical item set. Comparing the manifests of a real assembly at one commit, widening, comparative and detection each carry the same 951 items with the same fields, and only entailment differs, by the 346 draft and planned items the drafts asymmetry admits. That asymmetry is itd-183's and it works exactly as designed. The other three do not. The definitions state four distinct objects: widening reads the brief, glossary, disciplines, specs and shipped tree; comparative reads the candidate set, which is the widening reading's pre-admission output, against the declared selection criteria; detection reads the shipped tree against the claim record. What the assembler actually passes at comparative and at detection is the widening object unchanged. The comparative case is the sharper one, because its object is not a subset of the repository at all but the output of a previous reading, and the assembler has no channel for it -- spc-64 says as much in passing, that the in-cycle candidate set arrives by whatever channel spc-61 defines and the eval makes no claim about it, but nothing states that the position therefore receives the wrong object. No criterion measures cross-position differentiation, and the evals run on a fixture too small for the difference to show.

## Grounds

- pursued: we expect one committed entry per position, carrying that position's own object set and kinds, to give each position an item set of its own, because the byte-identical sets came from one admission serving four near-identical rows; two positions whose pinned digests coincided again, or a digest that moved without a commit to the entry, would show it wrong.
