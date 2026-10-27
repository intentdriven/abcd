---
schema_version: 1
id: "iss-2610262030475870"
slug: "the-context-page-says-disciplines-move"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 44 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/02-context.md:11"
remedy: "none (filed automatically)"
finding_ref: cfk-22452cb95401e2ce
---

The context page says disciplines move drafts/ to disciplines/ on plan; the intent chapter says plan's discipline branch does not ship

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/02-context.md:11` — "disciplines move from drafts/ to disciplines/ on plan and stay there"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:224` — "and a discipline branch (no spec — registers the rule in a disciplines gate store, moves drafts/ → disciplines/). None of these ship today."

02-context describes the shipped intent verb as moving disciplines from drafts/ to disciplines/ on plan, while 05-intent says the discipline branch of plan, which would make that move, is a later phase and that none of it ships today. Planning a discipline as 02-context describes is not something the shipped plan does.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 44.
