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
deferred_after: "v0.7.1"
deferral_reason: "Two instances are confirmed and the population is sixty-one, so the question is not whether to close two specs but whether each planned intent's work is already merged, and that is a judgement per record rather than a sweep. Closing spc-44 and spc-14 would ship two more intents into this release and add lines to a changelog composed from the records that reached a terminal folder, which makes it a decision about what this release contains rather than a defect to repair inside it. The maintainer ruled on exactly that: fix the two verbs that write records where nobody will find them, defer this. Deferred rather than left silent because the finding is real, its population is large, and nothing detects it: the record lint reads what records say about each other and the release derivation reads only terminal folders, so the one question that would catch it is asked by no gate. What is owed next cycle is the detector, not another audit by hand."
related_intents: [itd-2609111003026787]
resolution: "RS005 in scripts/check-issue-resolution.sh is the detector this record said was owed: a change declaring Delivers: itd-N is refused at merge unless the intent enters shipped/ in the same change, naming every spec still open that names it. It fires on a declaration, not on inferred liveness, per the settled remedy of itd-2609111003026787 (promoted from this record). The two confirmed instances (spc-44, spc-14) are closed by the batch clearance filed against iss-2608290808193471, which keeps the standing backlog; this gate does not reach it."
impact: breaking
resolved_by:
  intent: "itd-2609111003026787"
  spec: "spc-2609120450289528"
  commit: "c18eea94"
---

The agent-contract prompt lint ships and is armed as a blocker in the record-lint configuration, enforcing the frontmatter floor on every prompt under the agents directory and gating the diff-armed unbumped-edit check, yet the intent that promised it sits in the planned bucket and its spec sits open. The definition of done says a change that delivers a planned intent closes its spec in the same change, and that the omission is silent by construction: the release cut composes its changelog from terminal folders only, so an intent whose code is on the branch with its spec still open ships with no line and the cut exits zero doing so. That is the state here, and it means the release notes for this cut describe a repository in which this rule does not exist. The condition was found by a cross-check of the design record against the shipped surface rather than by any gate, because no gate asks whether a planned intent's work is already present: the record lint reads what the records say about each other and the release derivation reads only what reached a terminal folder, so the one question that would catch it, whether the code a planned intent describes is already merged, is asked by nothing. Fix: close the spec so the intent ships and the cut sees it, once the acceptance criteria are checked against what actually landed, which is the maintainer's call rather than an automatic move. Detector: a planned intent whose acceptance criteria are satisfied by code already on the branch is reported before a release derives its cut, so the choice to ship it or to keep it planned is made rather than defaulted.

## Second instance, and the size of the population, 2026-09-09

itd-93 and spc-14 are the same shape: the verb ships, the intent sits planned, the spec sits open, and the cut derives no line for it. Two confirmed instances make this a pattern rather than an oversight, and the population it could hide in is large: the planned bucket holds sixty-one intents and every one of their specs is open, so nothing in the record distinguishes an intent whose work is still ahead from one whose work is already merged. Neither instance was found by a gate. Both were found by a human-directed cross-check of the design record against the shipped surface, which is not a thing that runs on a schedule.

## Grounds

- pursued: we expect a mechanical refusal to close the forgotten-close gap where documentation did not, because the measured population of 61 accumulated under a documented rule that every reader had already been told; what would show this wrong is the gate arming and the population continuing to grow, which would mean the cause is not forgetting but an absent signal for when an intent is done
- pursued: we expect a declared delivery to arrive with its spec closed, because the merge now refuses one that does not; what would show this wrong is planned intents with open specs continuing to accumulate while changes that finish them carry no Delivers: trailer, meaning the cause is reluctance or an absent signal rather than forgetting
