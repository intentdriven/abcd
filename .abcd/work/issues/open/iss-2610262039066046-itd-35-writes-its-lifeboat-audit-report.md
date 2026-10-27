---
schema_version: 1
id: "iss-2610262039066046"
slug: "itd-35-writes-its-lifeboat-audit-report"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 840 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-35-lifeboat-integrity-audit.md:42"
remedy: "none (filed automatically)"
related_intents: [itd-35]
finding_ref: cfk-4adc71f4ea678b8d
---

itd-35 writes its lifeboat audit report under `.abcd/logbook/`, a root the universal-patterns page says was retired and must not be re-minted

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-35-lifeboat-integrity-audit.md:42` — "`.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:94` — "Its root was once `.abcd/logbook/`; a 2026-07-12 adjudication moved run output to the local ephemeral tier and retired that name, which must not be re-minted"

itd-35 puts the verification report at `.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}` and cites 04-universal-patterns § 6 as its convention. That section says the log root was once `.abcd/logbook/`, that run output moved to the local ephemeral tier, and that the name is retired and must not be re-minted. Built as written, itd-35 re-mints the retired name that the page it cites forbids.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 840.
