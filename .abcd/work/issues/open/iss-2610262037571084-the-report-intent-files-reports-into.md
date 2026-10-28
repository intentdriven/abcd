---
schema_version: 1
id: "iss-2610262037571084"
slug: "the-report-intent-files-reports-into"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 761 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/29-report.md:92"
remedy: "none (filed automatically)"
related_intents: [itd-2609221656361680]
finding_ref: cfk-17d586e7c9bd03bd
---

The report intent files reports into ~/.abcd/inbox/; the report chapter files them into ~/.abcd.noindex/inbox/.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/29-report.md:92` — "~/.abcd.noindex/inbox/< received-stamp>-< sender-key>.md"
- End B: `.abcd/development/intents/shipped/itd-2609221656361680-a-repository-abcd-manages-reports-back.md:42` — "`~/.abcd/inbox/<received-stamp>-<sender-key>.md` by default"

itd-2609221656361680 places the inbox at ~/.abcd/inbox/< received-stamp>-< sender-key>.md. 29-report.md lands every report at ~/.abcd.noindex/inbox/< received-stamp>-< sender-key>.md, and its greeting and board row count that folder. The one inbox carries two home names. After itd-2610030720038073's rename, which stops abcd while ~/.abcd exists, only the brief's can be current.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 761.
