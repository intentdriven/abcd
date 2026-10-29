---
schema_version: 1
id: "iss-2610262038066422"
slug: "itd-67-has-abcd-launch-ship-tag-the-repo"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 772 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-ad4593a3247f040c
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 37-reserve describes the shipped sequence and the ship code writes no tag; itd-67's tagging wording is the pre-adr-37 design."
---

itd-67 has `/abcd:launch ship` tag the repo; 37-reserve has the release workflow push the release tag after the merge

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:47` — "`/abcd:launch ship` auto-selects the bump tier (patch/minor/major) via the brief §4 phase-completion detection, writes the new version into `plugin.json`, updates `marketplace.json` (version + changelog entry), tags the repo"
- End B: `.abcd/development/brief/04-surfaces/37-reserve.md:311` — "the release tag is pushed by the release workflow after the merge, so a two-hour hold could lapse in the middle of a cut."

itd-67 puts tagging the repository in `/abcd:launch ship` itself, beside writing the new version. 37-reserve states the release tag is pushed by the release workflow after the merge, which is why the release hold lasts a day and ends when the tag appears on origin. The tag cannot be both the cut verb's own act and a workflow's act after a merge the cut waits on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 772.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. 37-reserve describes the shipped sequence and the ship code writes no tag; itd-67's tagging wording is the pre-adr-37 design.
