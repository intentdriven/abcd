---
schema_version: 1
id: "iss-2610262033286211"
slug: "capture-defines-promoted-as-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 409 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/06-capture.md:189"
remedy: "State in 35-drain.md line 93 that the promoted hand-back reads only the issue's related_intents, whichever end names the join, and say how that differs from 06-capture's two-ended promoted pair (or rename the hand-back kind); the audit-check filers are already set aside by the setAsidePromoted rule."
finding_ref: cfk-0528cc238f5d5ee1
---

Capture defines 'promoted' as the two-ended join between an issue and an intent; drain hands back as `promoted` any issue whose related_intents names an intent.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/06-capture.md:189` — "the pair is what "promoted" means: an issue may name an intent it is only related to, and it is promoted into the one that names it back"
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:93` — "An issue whose `related_intents` names an intent is handed back under `promoted`, naming each intent: an intent already owns it"

06-capture says 'promoted' means the pair: the issue's related_intents and the intent's related_issues naming each other. An issue may name an intent it is only related to without being promoted. 35-drain hands back under `promoted` every issue whose related_intents names an intent, even when only the issue names it. That includes the audit-check issues the verdict ingest files as merely related to an intent, and 35-drain says an intent already owns them. The same word means a two-way join on one page and a one-way relation on the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 409.
