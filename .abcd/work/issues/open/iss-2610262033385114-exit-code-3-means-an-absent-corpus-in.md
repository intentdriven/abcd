---
schema_version: 1
id: "iss-2610262033385114"
slug: "exit-code-3-means-an-absent-corpus-in"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 426 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/07-library.md:210"
remedy: "none (filed automatically)"
finding_ref: cfk-c720291535aa09f8
---

Exit code 3 means an absent corpus in the library chapter and contention in the land chapter.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/07-library.md:210` — "exits 3, a code distinct from a refusal, so a script can tell"
- End B: `.abcd/development/brief/04-surfaces/38-land.md:201` — "A landing a release cut holds exits 3, contention, as every verb that backs off does"

07-library has every corpus verb exit 3 when no corpus exists, as a code distinct from a refusal. 38-land says exit 3 is contention, used by every verb that backs off, and keeps a failed fetch off 3 because an automation backing off on 3 would wait on no one. An automation that follows the land chapter would wait indefinitely on a library verb whose corpus is simply missing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 426.
