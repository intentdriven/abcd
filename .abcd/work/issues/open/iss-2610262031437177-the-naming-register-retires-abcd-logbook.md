---
schema_version: 1
id: "iss-2610262031437177"
slug: "the-naming-register-retires-abcd-logbook"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 167 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:83"
remedy: "Change itd-35 lines 42 and 59 to `.abcd/.work.local/logs/audit/lifeboat-<ts>/`, the location 05-intent.md already gives audit/lifeboat."
related_intents: [itd-35]
finding_ref: cfk-f721fe4a1148fd0f
---

The naming register retires `.abcd/logbook/` and fails the build on it, but itd-35 writes its verification report under `.abcd/logbook/audit/`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:83` — "`.abcd/logbook/` was the maritime name for per-run logs, state and reports. It is retired and must not be re-minted"
- End B: `.abcd/development/intents/drafts/itd-35-lifeboat-integrity-audit.md:42` — "`.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}`"

04-naming says `.abcd/logbook/` is retired and must not be re-minted, that run output goes to the local ephemeral tier, and that `TestNoRetiredLogbookLocationInSource` fails the build if any Go source names the retired location. itd-35 puts its lifeboat verification report at `.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}`. Built as written, it re-mints the retired location and trips that test.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 167.
