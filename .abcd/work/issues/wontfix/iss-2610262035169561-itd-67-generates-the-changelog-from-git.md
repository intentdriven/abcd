---
schema_version: 1
id: "iss-2610262035169561"
slug: "itd-67-generates-the-changelog-from-git"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 559 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48"
remedy: "none (filed automatically)"
related_intents: [itd-67]
finding_ref: cfk-04ba2e2d8a0f86ad
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The cut composes from shipped intents and resolved issues; the closed spec names the change from itd-67's original source list."
---

itd-67 generates the changelog from git tags, commit and spec history and the launch report; the intent chapter says the cut composes only from terminal record folders

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-67-installable-versioned-plugin.md:48` — "generated from the canonical history (git tags + commit/spec history + launch report)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:245` — "only from terminal folders — the cut exits 0 without it"

itd-67 says the changelog is generated from git tags, commit and spec history and the launch report. The intent chapter says the launch cut composes only from terminal folders, so a planned intent whose code is already on main is left out and the cut still exits 0. A changelog built from commit history would include that work. A changelog built from terminal folders leaves it out. The two sources give different changelogs for the same release.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 559.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The cut composes from shipped intents and resolved issues; the closed spec names the change from itd-67's original source list.
