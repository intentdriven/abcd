---
schema_version: 1
id: "iss-2610262033062965"
slug: "itd-30-lets-a-user-facing-intent-carry-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 361 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-30]
finding_ref: cfk-a4d6cb4f156a42f9
---

itd-30 lets a user-facing intent carry a design fiction in place of a press release; the launch page composes the release page from the press releases of the user-facing intents shipped.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "abcd lets intent authors choose between a press release and a design fiction when capturing a user-facing change."
- End B: `.abcd/development/brief/04-surfaces/04-launch.md:103` — "composed from the press releases of the user-facing intents shipped since the last tag"

itd-30 lets the author of a user-facing change choose a design fiction instead of a press release, with one format per intent and no automatic conversion between the two, so a fiction intent has no press release. The launch page composes RELEASE.md from the press releases of the user-facing intents shipped since the last tag. Each intent in that set must be cited once, and persona quotes are carried word for word and checked against their source. A shipped fiction intent leaves the release page nothing to compose from, so the cut's rule cannot be met for it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 361.
