---
schema_version: 1
id: "iss-2610262036429463"
slug: "the-decision-log-intent-has-the-site"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 667 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md:28"
remedy: "In 22-site.md (the paragraph at line 199) say two inputs reach into the working tier, the issue ledger and the decision records (record.decisions), each off unless the composition asks for it, as 10-site.md already says."
related_intents: [itd-2609151138388536]
finding_ref: cfk-f236e5bb797ea55c
---

The decision-log intent has the site list every decision from `.abcd/work/decisions/`; the site chapter says its one working-tier input is the issue ledger, off by default.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2609151138388536-append-only-logs-conflict-on-every-merge.md:28` — "A command and the site list every decision in date order."
- End B: `.abcd/development/brief/04-surfaces/22-site.md:199` — "One input reaches past the durable record into the working tier, and it is off"

The intent writes each decision to its own file in `.abcd/work/decisions/` and says the site lists every decision in date order, on a page its 2026-10-14 ruling names the Decision log. The site chapter says one input reaches past the durable record into the working tier: the issue ledger, and only when a repository turns it on. The chapter's fixed page set names no decision log and cannot be added to. A site that lists the decisions would read a second working-tier input, and one that is always on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 667.
