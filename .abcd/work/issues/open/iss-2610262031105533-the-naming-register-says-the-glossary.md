---
schema_version: 1
id: "iss-2610262031105533"
slug: "the-naming-register-says-the-glossary"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 96 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:144"
remedy: "In 04-universal-patterns.md (local logs layout, lines 117-123) drop the grill/ session directory and its 'written inline to terminology/' note; it is the log layout of the grill sub-verb from superseded itd-27, which does not ship."
finding_ref: cfk-e0f22a7c0af23931
---

The naming register says the glossary lives only under `glossary/`, but the designed grill log writes glossary terms to `terminology/`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:144` — "is the one canonical glossary and the only place a glossary lives"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:117` — "glossary terms are written inline to terminology/, not batched here"

The naming register says the glossary at `../glossary/` is the one canonical glossary and the only place a glossary lives. The designed `.abcd/.work.local/logs/` layout in universal patterns says a grill session writes its glossary terms inline to `terminology/`. One concept, the canonical glossary, gets two homes under two names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 96.
