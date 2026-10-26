---
schema_version: 1
id: "iss-2610262044330700"
slug: "the-dashboard-ruling-promises-the-record"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1214 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-2610032150577708, itd-2609061543533170]
finding_ref: cfk-c5da748d90d453ed
---

The dashboard ruling promises the record site for abcd's own record only, but the shipped site verb renders the same pages for every managed repository

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2610032150577708-a-product-thinker-opens-abcd-s-dashboard.md:47` — "The record site and the dashboard are promised for abcd's own record only; itd-140 carries the ruling."
- End B: `.abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s.md:43` — "landing, explorer, record pages, graph, timeline, glossary and status render for every managed repository from its own text"

itd-2610032150577708 gives up the promise that the record site works for any project and keeps the record site and dashboard for abcd's own record only. itd-2609061543533170 shipped `abcd site setup`, which renders landing, explorer, record pages, graph, timeline, glossary and status for every managed repository from its own text. The two records disagree on whether a managed repository gets the record site.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1214.
