---
schema_version: 1
id: "iss-2610262037198530"
slug: "the-bump-re-authoring-workflow-stamps-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 714 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-2609301020001595-a-dependabot-bump-of-a-pinned-action-in.md:73"
remedy: "Ruled 2026-10-27, the recommended answer accepted (dcn-2610271112330536): What trailer does a commit whose message a non-abcd workflow computed (no model involved, the owner as author) carry? Answer 1: `Assisted-by: None`, as adr-2609292116133348 item 1 rules and scripts/dependency-reauthor.sh writes, with 34-build's 'human-only None' clause narrowed to the commits abcd composes under its own label. Answer 2: a new fourth form the attribution gate accepts for workflow-composed text, reversing the ADR's 'gate unchanged'. Recommend answer 1: it is shipped and ruled, the gate has no fourth form, and the owner is author of record under the rule they signed."
related_intents: [itd-2609301020001595]
finding_ref: cfk-74124b83a7988f46
---

The bump re-authoring workflow stamps a commit message a machine wrote with Assisted-by: None; 34-build treats None as meaning human-only, which is false for machine-written text.

A terminology drift (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-2609301020001595-a-dependabot-bump-of-a-pinned-action-in.md:73` — "a message naming the bot, its commit and the workflow, and `Assisted-by: None`"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:196` — "so neither a model's trailer nor the human-only `None` would be true of it"

In itd-2609301020001595, a workflow writes the landed commit's message (naming the bot, its commit and the workflow) and declares Assisted-by: None. 34-build says a commit whose message abcd computed takes abcd's own label, because 'the human-only None' would not be true of it. The two documents give the None trailer incompatible meanings: no AI involved, versus written by a person.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 714.
