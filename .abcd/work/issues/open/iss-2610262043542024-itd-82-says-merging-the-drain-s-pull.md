---
schema_version: 1
id: "iss-2610262043542024"
slug: "itd-82-says-merging-the-drain-s-pull"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1170 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-82-drain-ledger-triage.md:58"
remedy: "none (filed automatically)"
related_intents: [itd-82, itd-2610150602547271]
finding_ref: cfk-95a5c8e5d359310c
---

itd-82 says merging the drain's pull requests is a human gate by design; itd-2610150602547271 says that in auto mode a ready pull request merges with no human step

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:58` — "Merging pull requests, planning promoted drafts, writing decision records: human gates by design."
- End B: `.abcd/development/intents/planned/itd-2610150602547271-a-repository-declares-whether-its.md:145` — "in `auto`, a ready pull request merges with no human step."

itd-82 lists merging pull requests as a human gate by design, outside what the drain may do. itd-2610150602547271 lets a repository declare `auto`, in which the merge gate that the build loop (and so the drain's issue lanes) asks queues or arms the merge with no human involved. Under `auto`, the drain's pull requests merge without the human gate itd-82 says is built in.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1170.
