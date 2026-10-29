---
schema_version: 1
id: "iss-2610262044310958"
slug: "the-shipped-inbox-hands-cross-machine"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1212 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:90"
remedy: "none (filed automatically)"
related_intents: [itd-2609151838312703, itd-2609221656361680]
finding_ref: cfk-868ddc82c9c9c640
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Decision 1 of the announce intent states the carrier changed from a shared directory to fixed-form notes; the inbox intent's out-of-scope pointer is stale but nothing is built on it."
---

The shipped inbox hands cross-machine report delivery to the announcements intent, which carries no free text

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md:90` — "Each note carries a kind from a fixed list, the record or pull request it concerns, and an expiry, and no free text"
- End B: `.abcd/development/intents/shipped/itd-2609221656361680-a-repository-abcd-manages-reports-back.md:50` — "Cross-account and cross-machine delivery (the mailbox draft itd-2609151838312703 and the broker draft itd-2609151838327688)."

itd-2609221656361680 puts cross-account and cross-machine delivery of reports (a template with prose) out of scope and routes it to itd-2609151838312703, calling it a mailbox. That intent now carries only fixed-form notes with a kind, an item and an expiry, and no free text. As written, it cannot deliver a report.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1212.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. Decision 1 of the announce intent states the carrier changed from a shared directory to fixed-form notes; the inbox intent's out-of-scope pointer is stale but nothing is built on it.
