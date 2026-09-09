---
schema_version: 1
id: "iss-2609091642508005"
slug: "the-agent-contract-rule-ships-while-its-intent-sits-planned"
severity: "major"
category: "process"
source: "agent-finding"
found_during: "release-gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned"
---

The agent-contract prompt lint ships and is armed as a blocker in the record-lint configuration, enforcing the frontmatter floor on every prompt under the agents directory and gating the diff-armed unbumped-edit check, yet the intent that promised it sits in the planned bucket and its spec sits open. The definition of done says a change that delivers a planned intent closes its spec in the same change, and that the omission is silent by construction: the release cut composes its changelog from terminal folders only, so an intent whose code is on the branch with its spec still open ships with no line and the cut exits zero doing so. That is the state here, and it means the release notes for this cut describe a repository in which this rule does not exist. The condition was found by a cross-check of the design record against the shipped surface rather than by any gate, because no gate asks whether a planned intent's work is already present: the record lint reads what the records say about each other and the release derivation reads only what reached a terminal folder, so the one question that would catch it, whether the code a planned intent describes is already merged, is asked by nothing. Fix: close the spec so the intent ships and the cut sees it, once the acceptance criteria are checked against what actually landed, which is the maintainer's call rather than an automatic move. Detector: a planned intent whose acceptance criteria are satisfied by code already on the branch is reported before a release derives its cut, so the choice to ship it or to keep it planned is made rather than defaulted.

## Second instance, and the size of the population, 2026-09-09

itd-93 and spc-14 are the same shape: the verb ships, the intent sits planned, the spec sits open, and the cut derives no line for it. Two confirmed instances make this a pattern rather than an oversight, and the population it could hide in is large: the planned bucket holds sixty-one intents and every one of their specs is open, so nothing in the record distinguishes an intent whose work is still ahead from one whose work is already merged. Neither instance was found by a gate. Both were found by a human-directed cross-check of the design record against the shipped surface, which is not a thing that runs on a schedule.
