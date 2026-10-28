---
schema_version: 1
id: "iss-2610262032092604"
slug: "invariant-21-names-abcd-s-home-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 228 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:179"
remedy: "none (filed automatically)"
related_intents: [itd-2609221656361680]
finding_ref: cfk-854d583e37c02632
---

Invariant 21 names abcd's home `~/.abcd.noindex`; itd-2609221656361680 puts the inbox at `~/.abcd/inbox/`.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:179` — "the `.noindex` suffix the home carries, `~/.abcd.noindex`"
- End B: `.abcd/development/intents/shipped/itd-2609221656361680-a-repository-abcd-manages-reports-back.md:42` — "`~/.abcd/inbox/<received-stamp>-<sender-key>.md` by default"

The brief states that the home carries the `.noindex` suffix as `~/.abcd.noindex`. The report intent names `~/.abcd/inbox/` as the default location of filed reports. A reader following the intent would look for reports in a folder the brief says is not abcd's home.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 228.
