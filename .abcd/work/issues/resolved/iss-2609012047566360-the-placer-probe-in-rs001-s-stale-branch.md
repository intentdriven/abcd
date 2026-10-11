---
schema_version: 1
id: "iss-2609012047566360"
slug: "the-placer-probe-in-rs001-s-stale-branch"
severity: "minor"
category: "observation"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
resolution: "RS001's stale-branch split (and RS005's twin for shipped/) now asks the merge base's tree whether the record was already terminal there, and names as placer the base-side commit that added or renamed the path into place (--diff-filter=AR), never a later edit; fail() deletes control bytes from every refusal, so a placer subject carrying an escape sequence no longer reaches the terminal. Proved by cases in scripts/check-issue-resolution-cases.sh: a base-side body edit of a record terminal at the merge base (and the shipped/ twin) prescribes no rebase, the placer named is the move and not a later edit, and an escape sequence in the placer subject is named without its control bytes; each failed against the previous gate."
impact: internal
resolved_by:
  commit: "7a96f7b0f"
---

The placer probe in RS001's stale-branch diagnosis (scripts/check-issue-resolution.sh) names the last base-side commit that touched the record's terminal path after divergence, as evidence that the base placed the record there after the branch forked. Any touch qualifies, so a body edit of a record that was already terminal at the merge base reports 'placed there on main's side by <sha> … rebase' when the honest verdict is 'terminal before this branch diverged; drop the trailer'. The probe should key on the commit that ADDED or renamed the path into the terminal folder (--diff-filter=AR) or compare the merge base's tree, not on any touch. Separately, the base-side commit subject is interpolated into the stderr message unsanitised; a subject carrying terminal control sequences would reach the terminal through the gate's output. Found by the ruthless review of the hygiene branch; left open as a follow-up.

## Grounds

- pursued: a trailer naming a record terminal before the branch diverged is told to drop the trailer and a truly stale branch is still told to rebase naming the placing commit; the existing stale-branch and terminal-before-divergence cases going red would show it wrong
