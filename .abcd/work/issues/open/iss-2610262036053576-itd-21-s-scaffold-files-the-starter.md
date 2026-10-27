---
schema_version: 1
id: "iss-2610262036053576"
slug: "itd-21-s-scaffold-files-the-starter"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 622 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-21-no-lifeboat-scaffolding.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-21]
finding_ref: cfk-82a7c5cc300232a8
---

itd-21's scaffold files the starter intent as `itd-1`; 24-decide says every minting family draws timestamp ids through one mint that reads no maximum.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-21-no-lifeboat-scaffolding.md:32` — "capturing the project's mission as `itd-1`"
- End B: `.abcd/development/brief/04-surfaces/24-decide.md:125` — "four-digit uniform suffix every other minting family draws through"

itd-21 has the scaffold's starter intent capture the project's mission as itd-1. 24-decide describes ids as the twelve-digit UTC stamp plus the four-digit uniform suffix every other minting family draws through recordid.Minter, a mint that reads no maximum. Under that mint no newly created intent can carry the ordinal id itd-1, so the scaffold as written cannot produce what it promises.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 622.
