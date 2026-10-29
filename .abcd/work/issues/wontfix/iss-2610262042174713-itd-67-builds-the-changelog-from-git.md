---
schema_version: 1
id: "iss-2610262042174713"
slug: "itd-67-builds-the-changelog-from-git"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1059 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-d746792ff452a028
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The release entry describes what ships; itd-67's source list and reroute are the original design replaced by the derived changelog."
---

itd-67 builds the changelog from git tags, commit and spec history and the launch report, and lets a gate append to it; the release entry composes it from records that reached a terminal folder.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48` — "generated from the canonical history (git tags + commit/spec history + launch report) — never hand-curated prose"
- End B: `.abcd/development/brief/glossary/distribution/release.md:29` — "composes its changelog from the records that reached a terminal folder"

itd-67 generates the changelog from git tags, commit and spec history and the launch report. It also makes the changelog the place where itd-65's doc-history gate appends the change narration it strips from doc bodies. The release entry says `launch ship` composes the changelog from the records that reached a terminal folder. A changelog built from records has no room for lines taken from commit history or for prose a gate appends.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1059.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The release entry describes what ships; itd-67's source list and reroute are the original design replaced by the derived changelog.
