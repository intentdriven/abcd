---
schema_version: 1
id: "iss-2610262044430202"
slug: "itd-30-builds-its-fiction-format-on-an"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1225 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:41"
remedy: "none (filed automatically)"
related_intents: [itd-30, itd-46]
finding_ref: cfk-f09b5837c75840c9
---

itd-30 builds its fiction format on an `/abcd:intent new` sub-verb, but shipped itd-46 removed that sub-verb and made quoted text the create path

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:41` — "/abcd:intent new --format=fiction "< free-text>""
- End B: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18` — "files the new intent directly — no `new` sub-verb required."

itd-30 puts its format flag and its fiction-shaped interview on `/abcd:intent new --format=fiction "<free-text>"`. itd-46 shipped quoted text as the create path, with no `new` sub-verb required. Built as written, itd-30 has no `new` sub-verb to carry its flag. Either itd-30 moves the flag to the quoted-text form or itd-46's removal does not hold.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1225.
