---
schema_version: 1
id: "iss-2610262045033573"
slug: "itd-111-names-abcd-version-as-where"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1248 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-111, itd-2609212130136102]
finding_ref: cfk-6600d7c6f8c4aa78
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later shipped record names the replacement; nothing in itd-111 needs an edit."
---

itd-111 names abcd version as where install mode and vintage always show; itd-2609212130136102 removes the abcd version verb

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-111-a-stale-abcd-never-answers-silently.md:23` — "`abcd version` and `abcd ahoy` always show install"
- End B: `.abcd/development/intents/shipped/itd-2609212130136102-abcd-s-verbs-consolidate-ahoy-s-three.md:40` — "replaces `abcd version`; `version --check` becomes `update --check`"

itd-111 promises that abcd version and abcd ahoy always show install mode and vintage. itd-2609212130136102 replaces abcd version with --version. One of the two surfaces itd-111 names as always showing the vintage is a verb the later record retires.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1248.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The later shipped record names the replacement; nothing in itd-111 needs an edit.
