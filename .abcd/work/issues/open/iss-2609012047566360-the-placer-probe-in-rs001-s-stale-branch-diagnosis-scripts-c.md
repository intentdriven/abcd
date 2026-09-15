---
schema_version: 1
id: "iss-2609012047566360"
slug: "the-placer-probe-in-rs001-s-stale-branch-diagnosis-scripts-c"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
---

The placer probe in RS001's stale-branch diagnosis (scripts/check-issue-resolution.sh) names the last base-side commit that touched the record's terminal path after divergence, as evidence that the base placed the record there after the branch forked. Any touch qualifies, so a body edit of a record that was already terminal at the merge base reports 'placed there on main's side by <sha> … rebase' when the honest verdict is 'terminal before this branch diverged; drop the trailer'. The probe should key on the commit that ADDED or renamed the path into the terminal folder (--diff-filter=AR) or compare the merge base's tree, not on any touch. Separately, the base-side commit subject is interpolated into the stderr message unsanitised; a subject carrying terminal control sequences would reach the terminal through the gate's output. Found by the ruthless review of the hygiene branch; left open as a follow-up.
