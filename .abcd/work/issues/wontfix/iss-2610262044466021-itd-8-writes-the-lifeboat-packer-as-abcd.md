---
schema_version: 1
id: "iss-2610262044466021"
slug: "itd-8-writes-the-lifeboat-packer-as-abcd"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1229 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-8-with-code-bundling.md:15"
remedy: "none (filed automatically)"
related_intents: [itd-8, itd-88]
finding_ref: cfk-ba187efd4d42f126
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent says which spelling holds and the command page agrees with itd-8, so nothing needs an edit."
---

itd-8 writes the lifeboat packer as `/abcd:disembark pack <repo> <path>`; shipped itd-88 writes it as `disembark <repo> to <dest>`

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-8-with-code-bundling.md:15` — "Running `/abcd:disembark pack <repo> <path> --with-code` packs not only the synthesised brief"
- End B: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:41` — "The packer (`disembark <repo> to <dest>`), embark, and the round-trip"

Both documents name the command that packs a repository into a lifeboat. itd-8 hangs `--with-code` on `/abcd:disembark pack <repo> <path>`. itd-88, which places the packer in the disembark sequence, writes it as `disembark <repo> to <dest>`. One operation has two command forms, so the flag itd-8 adds has no agreed command to attach to.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1229.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The shipped intent says which spelling holds and the command page agrees with itd-8, so nothing needs an edit.
