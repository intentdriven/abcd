---
schema_version: 1
id: "iss-2610262030315186"
slug: "context-presents-abcd-corpus-json-as-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 11 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:47"
remedy: "At 02-context.md:47 say that .abcd/corpus.json is a reserved name not yet in the tree and that the three repositories are the intended seed, as 04-naming.md and 02-disembark.md already say; carry the same note into the 'Validation corpus' section of 02-constraints/01-platform.md."
finding_ref: cfk-a3a3697cecc0f7aa
---

Context presents `.abcd/corpus.json` as an existing, seeded validation-corpus list; the naming page says the name is reserved and not yet in the tree

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:47` — "**Validation corpus:** user-maintained list in `.abcd/corpus.json`, seeded with:"
- End B: `.abcd/development/brief/02-constraints/04-naming.md:80` — "`corpus.json`, which is reserved here as a name and is not yet in the tree"

The context page, which the platform page names as the canonical validation-corpus source, says the corpus is a user-maintained list in `.abcd/corpus.json`, seeded with three repositories. The naming page says `corpus.json` is only a reserved name and not yet in the tree. A list cannot be seeded in a file that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 11.
