---
schema_version: 1
id: "iss-2610262032478338"
slug: "the-matrix-s-multi-repo-corpus-check"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 317 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:216"
remedy: "In 06-delivery/02-verification-matrix.md mark the Multi-repo corpus row (staged): .abcd/corpus.json does not exist, so the row has no runnable subject."
finding_ref: cfk-a766bed60e2b5e3d
---

the matrix's multi-repo corpus check needs a validation corpus; disembark says the corpus manifest is not yet in the tree

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:216` — "The corpus manifest (`.abcd/corpus.json`) is not yet in the tree."
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:45` — "Every milestone acceptance gate passes on each corpus repo, with documented exemptions"

02-verification-matrix carries an unstaged row requiring every acceptance gate to pass on each corpus repo with documented exemptions. 02-disembark says acceptance is checked across a validation corpus but that the corpus manifest (.abcd/corpus.json) is not yet in the tree. The matrix's check depends on a corpus definition the surface chapter says does not exist, so it cannot be run as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 317.
