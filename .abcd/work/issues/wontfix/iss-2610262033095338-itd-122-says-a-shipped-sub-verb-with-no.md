---
schema_version: 1
id: "iss-2610262033095338"
slug: "itd-122-says-a-shipped-sub-verb-with-no"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 368 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:964"
remedy: "none (filed automatically)"
related_intents: [itd-122]
finding_ref: cfk-642159f335377e71
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The press-release line in itd-122 states the rule without the hidden-subtree exemption the README and the gate carry; the committed chapter passes the gate."
---

itd-122 says a shipped sub-verb with no table row fails the build; the launch chapter ships `smoke-pages` with no row

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:964` — "`abcd launch ship`, `abcd launch smoke-pages`"
- End B: `.abcd/development/intents/shipped/itd-122-the-registry-cannot-wave-its-hands-every.md:21` — "sub-verb that ships without a row fails the build"

itd-122 says `surface_coverage` checks every surface chapter's sub-verb table against the command tree in both directions, so a sub-verb that ships without a row fails the build. The launch chapter's generated appendix lists `abcd launch smoke-pages` among the shipped sub-verbs, and its prose calls it a hidden, operator-internal sub-verb. Its sub-verb table lists only archive, manifests, receipts, scaffold and ship. Either the gate exempts hidden sub-verbs, which itd-122 does not say, or the committed chapter would fail it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 368.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): the two ends do not conflict when read in full. The press-release line in itd-122 states the rule without the hidden-subtree exemption the README and the gate carry; the committed chapter passes the gate.
