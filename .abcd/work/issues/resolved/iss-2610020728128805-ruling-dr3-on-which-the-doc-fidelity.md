---
schema_version: 1
id: "iss-2610020728128805"
slug: "ruling-dr3-on-which-the-doc-fidelity"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-60"
origin: researcher-authored
production_mode: hand-written
remedy: "Append a dated line to .abcd/work/DECISIONS.md restating DR3 verbatim ('itd-60 docs-fidelity verdict: FOUND AUTOMATICALLY by spec close and launch ship (a saved verdict labelled with the reviewed code's fingerprint; match -> proceed; none/stale -> refuse run the docs review first), like the preflight receipt') with the date and the interviewer that relayed it, never editing the 2026-09-30 line in place (DA002); the same local file holds CB1, CC1, CD1, CD2, DQ2b and CJ1b, which a peer review already flagged as owed. Grounds: lane-common's rule that every question the record does not answer is answered by the recorded decisions, and AGENTS.md's append-only decision log."
resolution: "DECISIONS.md carries a dated 2026-10-07 entry naming the issue that restates ruling DR3 verbatim from the local-tier rulings file, with its date (2026-09-29) and relaying interviewer (abcd-23 [b1e81b]), appended without editing any earlier line; the 2026-10-02 entry of lane rd2 (commit 0ed0e8a51) had already restated DR3, CB1, CC1, CD1 and CD2, and DQ2b and CJ1b have 2026-09-30 entries, so no ruling the issue names is still unrecorded."
impact: internal
resolved_by:
  commit: "427708676a94bd53d6993c83031114c49f9a66f2"
---

Ruling DR3, on which the doc-fidelity gate's shape rests (itd-60, ac-2: the semantic layer is a saved receipt for HEAD that spec close and launch ship find automatically, rather than a review invoked at the close), is cited by three source files (internal/core/docfidelity/store.go:4, internal/core/docfidelity/docfidelity.go:14, internal/core/lint/gatereceipt.go:10) but recorded nowhere in the committed record: the 2026-09-30 entry of .abcd/work/DECISIONS.md restates DR1, DR2, DR4, DR5 and DR6 and skips DR3, whose text lives only in a local-tier rulings file this checkout cannot cite. The fidelity audit of itd-60 therefore grades ac-2 as a signed-off divergence on the strength of an uncommitted ruling; a reader of the record alone cannot see why the gate reads a receipt instead of running the reviewer.

## Grounds

- pursued: a reader of the committed record finds DR3's text in DECISIONS.md, in an entry naming this issue; shown wrong if the entry's quote differs from line 31 of rulings-answered-2609-29-b.md, or if decisions-append (DA001-DA004) refuses the commit.
