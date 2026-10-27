---
schema_version: 1
id: "iss-2610262035462032"
slug: "the-glossary-allows-bare-the-ledger-only"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 597 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/ledger.md:44"
remedy: "none (filed automatically)"
finding_ref: cfk-3602d99b571a1cc6
---

The glossary allows bare 'the ledger' only for the issue ledger and spells the source-provenance store 'the sources ledger'; the guard chapter writes 'source-ledger' and then bare 'the ledger' for it

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/ledger.md:44` — "Use bare "the ledger" only where the issue ledger is unambiguous"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:656` — "Flipping a source-ledger line to public citation is blocked, because the ledger records that a person chose to cite the source"

ledger.md says bare 'the ledger' is used only where the issue ledger is unambiguous, and every other ledger is named in full ('the sources ledger' for the corpus's source-to-decision provenance). The guard chapter calls it a 'source-ledger line' and then says 'the ledger records that a person chose to cite the source', so the bare form there means the sources ledger, not the issue ledger.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 597.
