---
schema_version: 1
id: "iss-2610251558471369"
slug: "docs-reference-terminology-md-58-maps"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "docs-currency review of docs/, 2026-10-24, reader D3"
origin: researcher-authored
production_mode: hand-written
found_at: "docs/reference/terminology.md"
remedy: "Rewrite the Memory position to cover both halves (memory notes recalled on a matching prompt and kept on the machine until promoted; the library's store of cited pages consulted on demand with library ask), and drop or ground the curator clause; run abcd lint docs."
resolution: "The crosswalk's Memory row now leads with memory notes (recalled on a matching prompt, kept on the machine until a person promotes one into .abcd/memory.json), then the library's cited pages consulted on demand with abcd library ask; the ungrounded 'single writing curator' clause is replaced by the host session distilling and abcd library ingest writing. abcd lint docs is clean."
impact: fix
resolved_by:
  commit: "5c842acd425a2c772409bcab818a803ebc5203a2"
---

docs/reference/terminology.md:58 maps Memory onto the library's .abcd/memory/ store only and omits memory notes, the abcd feature closest to the established meaning: abcd memory add keeps a lesson with its recall words on the machine or, once promoted, in .abcd/memory.json (internal/core/memnote/layers.go:13), and the rules loader brings it back into a session when a prompt matches (internal/core/rules/inject.go:40-43). The glossary gives 'memory' as memory-note's alias (.abcd/development/brief/glossary/core/memory-note.md), so the row's use of the word disagrees with the canonical vocabulary. Its 'single writing curator' names no shipped component: the host session distils the pages and abcd library ingest writes them (commands/library.md:310-313), and principle-distiller, the curator the brief names, feeds disembark principles (agents/principle-distiller.md:3).

## Grounds

- pursued: the row agrees with the glossary's memory-note sense and names only shipped components; TestTerminologyMemoryRowCoversBothHalves would fail if the row dropped either half, put the page store first, or named a curator again
