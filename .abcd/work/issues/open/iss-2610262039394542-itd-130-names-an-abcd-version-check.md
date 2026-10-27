---
schema_version: 1
id: "iss-2610262039394542"
slug: "itd-130-names-an-abcd-version-check"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 876 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-130]
finding_ref: cfk-f3e080159a93e212
---

itd-130 names an `abcd version --check` command; the surfaces register says `version` is the root's `--version` flag, not a verb

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-130-abcd-update-completes-a-chosen-update-in.md:32` — "typed `abcd version --check` on the machine where they first installed the"
- End B: `.abcd/development/brief/04-surfaces/README.md:156` — "`version` is the root's `--version` flag (itd-2609212130136102), not a verb"

itd-130's press release has the user type `abcd version --check` to learn an update is available, and it builds `abcd update` as the completion of that check. The surfaces register says `version` is the root's `--version` flag and not a verb, and its two verb tables together name every verb the binary registers. Neither table lists `version`, so the command the intent names does not exist on the surface the brief describes.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 876.
