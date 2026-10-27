---
schema_version: 1
id: "iss-2610262030354870"
slug: "the-constraints-index-calls-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 19 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/README.md:12"
remedy: "none (filed automatically)"
finding_ref: cfk-b6dc35cfd23f2d0b
---

The constraints index calls the invariants page a placeholder pointer; the page calls itself the canonical register

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/README.md:12` — "Navigational pointer to invariant-shaped rules scattered across the brief (placeholder)"
- End B: `.abcd/development/brief/02-constraints/03-invariants.md:3` — "This file is the canonical home for the project's invariants"

The constraints README describes `03-invariants.md` as a navigational pointer to invariant-shaped rules scattered across the brief, and marks it a placeholder. The page itself is marked LIVE and says it is the canonical home for the project's invariants, with the other locations as cross-references and not the register. A reader following the index is sent away from the authority.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 19.
