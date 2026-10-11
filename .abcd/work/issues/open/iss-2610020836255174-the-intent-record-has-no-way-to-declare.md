---
schema_version: 1
id: "iss-2610020836255174"
slug: "the-intent-record-has-no-way-to-declare"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/edges.go"
remedy: "A declared-pair edge the linter honours: a frontmatter field (or an edge note) on each intent of a pair naming the other intent and the piece that direction needs, which edge_cycle reads to pass a two-intent cycle only when both directions declare their piece, still reporting any undeclared cycle; the person asked for an intent to 'check the pair first' (CY1, 2026-10-02) whose routing waits on their decomposition confirm, so this fix may be carried by that intent; shown wrong if a declared pair still blocks the ordering the loop's pick needs."
---

The intent record has no way to declare a mutual builds_on pair that record-lint's edge_cycle rule honours. The technical facilitator ruled on 2026-10-02 that three two-intent cycles stay as declared mutual pairs, each direction naming the piece it needs (CY1: itd-2609201916056194 and itd-2609201916151817; CY2: itd-14 and itd-15, itd-2609081951381895 and itd-2609170822093401), but the frontmatter offers only builds_on and blocked_by, and edge_cycle reports every strongly connected set whatever the records say, so the three pairs are declared only in each intent's Audit Notes and still read as cycles to the linter, and to anything that orders work by these edges. Until a declared pair is something the linter can read, edge_cycle cannot be promoted from warn to blocker without refusing the ruled pairs.
