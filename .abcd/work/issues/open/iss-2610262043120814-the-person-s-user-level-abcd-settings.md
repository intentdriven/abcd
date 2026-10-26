---
schema_version: 1
id: "iss-2610262043120814"
slug: "the-person-s-user-level-abcd-settings"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1122 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2609301918174237-a-provider-off-this-machine-takes-a-file.md:19"
remedy: "none (filed automatically)"
related_intents: [itd-2609301918174237, itd-200]
finding_ref: cfk-c97d01dd2a6ee5ce
---

The person's user-level abcd settings live under ~/.abcd/ in itd-200 and under ~/.abcd.noindex/ in the provider-bundle draft.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2609301918174237-a-provider-off-this-machine-takes-a-file.md:19` — "the owner can change that list and the cap for one project in `~/.abcd.noindex/config.json` and nowhere else"
- End B: `.abcd/development/intents/shipped/itd-200-abcd-s-presence-should-be-visible-in-the.md:109` — "The user-level defaults under `~/.abcd/`"

itd-200 places abcd's status-line defaults and the person's off switch in 'the user-level defaults under ~/.abcd/'. itd-2609301918174237 places the owner's personal, machine-layer settings in '~/.abcd.noindex/config.json and nowhere else'. Both describe the same home for the person's own abcd settings under two directory names, so a verb reading one will not find what was written to the other.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1122.
