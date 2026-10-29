---
schema_version: 1
id: "iss-2610262035497039"
slug: "the-glossary-says-a-record-an-adr"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 600 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record.md:20"
remedy: "In record.md limit 'lifecycle is directory-as-truth' to the families that live in buckets (intents, specs, issues) and say an ADR carries its status in frontmatter."
finding_ref: cfk-8d65c73b868db98b
---

The glossary says a record, an ADR included, carries no status field; the decide chapter mints ADRs with a `status: proposed` field the author later changes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record.md:20` — "Lifecycle is directory-as-truth — a record moves between buckets rather than carrying a status field."
- End B: `.abcd/development/brief/04-surfaces/24-decide.md:160` — "and the record lands with `status: proposed`"

record.md names an ADR (adr-N) as a record and states that lifecycle is directory-as-truth, with a record moving between buckets rather than carrying a status field. The decide chapter says a freshly minted ADR lands with `status: proposed`, that the author sets `accepted` when the decision is in force, and that the index row's status comes from the record's frontmatter. So ADR lifecycle is held in a status field, not in a folder.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 600.
