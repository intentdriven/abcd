---
schema_version: 1
id: "iss-2610262033533504"
slug: "the-board-chapter-calls-admissions-and"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 447 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/08-abcd.md:85"
remedy: "Add a note or rows to the record-families glossary page naming the ledger records the record dispatch handles (admission, surprise, reframe, decision) as families with no lifecycle folder, or reword 08-abcd.md:85 from 'the issue ledger's two folderless families' to 'ledger records'."
finding_ref: cfk-804e7482d38f6bc5
---

The board chapter calls admissions and surprises 'the issue ledger's two folderless families'; the glossary's one map of record families lists no such families.

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/08-abcd.md:85` — "An admission and a surprise are the issue ledger's two folderless families"
- End B: `.abcd/development/brief/glossary/core/record-families.md:4` — "The one page that maps abcd's record families (intent, spec, step, bundle, issue, release, status) and how they relate"

glossary/core/record-families.md presents itself as 'The one page that maps abcd's record families (intent, spec, step, bundle, issue, release, status)' and says a name with no row there is a name nobody has justified. 08-abcd names admissions and surprises as two further families of the issue ledger. Either the map is incomplete or the board chapter uses 'family' outside the glossary's closed set.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 447.
