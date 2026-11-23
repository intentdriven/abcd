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
remedy: "At 17-guard.md line 656 write 'sources ledger line' and 'the sources ledger' instead of 'source-ledger line' and 'the ledger'."
finding_ref: cfk-3602d99b571a1cc6
resolution: "The guard chapter (.abcd/development/brief/04-surfaces/17-guard.md) now names the corpus's source-to-decision store in full, as the glossary's ledger entry requires. The remedy's sentence (now at lines 1017-1018, not 656) reads 'Flipping a sources ledger line to public citation is blocked, because the sources ledger records that a person chose to cite the source'. Line 216, the same act in the same hyphenated spelling, now reads 'flipping a sources ledger line', so the page uses one spelling. The backticked rule ids abcd-source-ledger-flip-any-program and abcd-source-ledger-flip-retired-any-program are identifiers and are unchanged. No other page changed."
impact: internal
resolved_by:
  commit: "56e728a552b8c56ec15ba56d80feec203c304187"
---

The glossary allows bare 'the ledger' only for the issue ledger and spells the source-provenance store 'the sources ledger'; the guard chapter writes 'source-ledger' and then bare 'the ledger' for it

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/ledger.md:44` — "Use bare "the ledger" only where the issue ledger is unambiguous"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:656` — "Flipping a source-ledger line to public citation is blocked, because the ledger records that a person chose to cite the source"

ledger.md says bare 'the ledger' is used only where the issue ledger is unambiguous, and every other ledger is named in full ('the sources ledger' for the corpus's source-to-decision provenance). The guard chapter calls it a 'source-ledger line' and then says 'the ledger records that a person chose to cite the source', so the bare form there means the sources ledger, not the issue ledger.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 597.

## Grounds

- pursued: glossary/core/ledger.md's Senses table spells this store 'the sources ledger', and line 44 allows bare 'the ledger' only for the issue ledger or inside 07-library.md. The sentence describes abcd library ledger --flip (internal/core/guard/personacts.go lines 71-82), which is the sources ledger. A grep of the page for the hyphenated spelling outside a backticked id, and for 'because the ledger records', hit lines 216, 1017 and 1018 before the edit and nothing after; 'sources ledger' now hits 216, 1017 and 1018. It would be shown wrong by a prose 'source-ledger' or a bare 'the ledger' for this store left on the page, by a changed rule id, or by a glossary that allows the bare form outside the library chapter.
