---
schema_version: 1
id: "iss-2610262033042911"
slug: "itd-35-writes-its-lifeboat-report-in-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 355 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-35-lifeboat-integrity-audit.md:42"
remedy: "none (filed automatically)"
related_intents: [itd-35]
finding_ref: cfk-b01c709d57778adb
---

itd-35 writes its lifeboat report in the repository under .abcd/logbook/audit/; the embark page says lifeboat operations record at the operator level under ~/.abcd.noindex/voyage/, never committed.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-35-lifeboat-integrity-audit.md:42` — "Verification report output**: `.abcd/logbook/audit/lifeboat-<ts>/report.{json,md}`"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:262` — "Lifeboat *operations* write provenance and history to **`~/.abcd.noindex/voyage/<source-root-sha>/`**: the operator level"

itd-35's check of a lifeboat is a lifeboat operation, and it writes its report inside the repository, at .abcd/logbook/audit/lifeboat-< ts>/report.{json,md}. The embark page says lifeboat operations write provenance and history to ~/.abcd.noindex/voyage/< source-root-sha>/, at the operator level and never committed, under adr-35, which superseded the earlier in-repository location of adr-4. The two documents give incompatible homes for the record of a lifeboat operation.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 355.
