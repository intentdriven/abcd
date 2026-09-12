---
schema_version: 1
id: "iss-2609100518522943"
slug: "an-intent-can-be-planned-without-an-impact-and-only-discover"
severity: "major"
category: "ux"
source: "agent-finding"
found_during: "autonomous-run field experiment in a managed repository, 2026-09-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent"
resolution: "Fixed in v0.8.0, and better than the record asked for. spec close now takes --impact, stamps a supplied judgement onto an intent carrying none, validates one already recorded at the same bar, and refuses a flag that disagrees with the record. The refusal arrives at the close either way, but it now names the remedy and can be satisfied in the same command rather than sending the operator back to edit frontmatter by hand."
impact: fix
---

An intent can be planned without an impact and only discovers it at the moment of shipping, in the landing commit. The readiness report has no row for impact, so a draft filed without one passes every readiness gate and plans cleanly, and the refusal arrives at spec close, which by this repository's own convention happens in the same change that lands the work. The operator is therefore told to go back and stamp a field on a record at the exact point they are trying to close a change they have already written and tested. Two sessions hit this independently within a day, one during an autonomous run and one while shipping an intent by hand, and both fixed it the same way, by editing the frontmatter directly rather than through a verb, because no verb stamps impact on a planned intent. Either readiness should report impact as a row alongside the criteria and the grounds, so the gap surfaces while the draft is still being written, or planning should refuse without one, or a verb should exist to stamp it afterwards. What should not persist is a required field whose absence is silent until the least convenient moment.

## Grounds

- pursued: we expect stamping at the close to be sufficient because the close is the first moment the judgement is unavoidable, so a plan-and-ship sequence driven by abcd's own verbs either records it or says it is missing; it is shown wrong if an operator still needs to edit frontmatter to ship
