---
schema_version: 1
id: "iss-2610262031019028"
slug: "the-naming-rule-forbids-verb-show-at"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 74 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:100"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): Does the ban on `<verb> show` apply by name, or only to a show that takes no operand? Answers: (a) by name, so `inbox show` and `history show` are gaps to rename; (b) only operand-less renders, so `show <id>` conforms and the sentence is narrowed. Recommend (b): the register's own earn-your-place test admits a sub-verb that takes a positional, and `inbox show` takes a report id while bare `abcd inbox` already lists."
finding_ref: cfk-77c1a868d2dff291
---

The naming rule forbids `<verb> show` at design time, but the inbox chapter ships a show sub-verb and does not mark it as a gap

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:100` — "`<verb> show`, `<verb> stats`, `<verb> view` and a plain unfiltered `<verb> list` are **forbidden** at design time"
- End B: `.abcd/development/brief/04-surfaces/30-inbox.md:26` — "Showing one report by its id renders it whole, waiting or promoted."

The naming convention forbids `<verb> show` at design time instead of leaving it to review. The inbox chapter, designed after the rule, ships `show` as a sub-verb that renders one report by its id. Unlike the memory chapter, which marks its `list` as a gap against the rule, it presents `show` as conforming. The prohibition and the shipped name cannot both hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 74.
