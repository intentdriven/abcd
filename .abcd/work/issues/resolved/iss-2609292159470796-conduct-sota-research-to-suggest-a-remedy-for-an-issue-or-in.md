---
schema_version: 1
id: "iss-2609292159470796"
slug: "conduct-sota-research-to-suggest-a-remedy-for-an-issue-or-in"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "autonomous run A, 2026-09-29 (user to abcd-cb [e13a96])"
origin: researcher-authored
production_mode: hand-written
remedy: "Add a short section to the prefer-sota principle: a remedy proposed for an issue, or chosen in an autonomous run, cites its grounds, a primary-source state-of-the-art check wherever the fix depends on outside practice, so the record shows why it beat the alternatives; and say so in a clause of the OPINIONS rule that points at prefer-sota."
resolution: "prefer-sota gains a section: a remedy proposed for an issue, or chosen in an autonomous run, cites its grounds, including a primary-source state-of-the-art check where the fix depends on outside practice; the bundled OPINIONS rule pointing at the principle says so in a clause, pinned by TestOpinionsPreferSOTARuleNamesRemedyGrounds."
impact: internal
resolved_by:
  commit: "0ce47ff89fc668343d8d1c12f890fc2082c9bfd0"
---

Conduct SOTA research to suggest a remedy for an issue or, in an autonomous run, to justify selecting a remedy: a remedy chosen without a prior-art check can repeat a known-bad approach, and a run that picks one leaves no evidence of why it beat the alternatives

## Grounds

- pursued: a remedy that records its grounds, with a primary source where outside practice decides it, lets a later reader see why it beat the alternatives; internal record practice, the principle's own reasoning, so no outside source is cited. Shown wrong if remedies picked under the rule still arrive with no grounds a reader can open.
