---
schema_version: 1
id: "iss-2608290810034854"
slug: "two-adversarial-reviews-over-the-assembled-multi-branch-diff"
severity: "minor"
category: "process"
source: "impl-review"
found_during: "intent-implementation-run"
found_at: ".abcd/work/reviews"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Add a reviews-charter line making the integration-level pass non-redundant and handing reviewers the invariants?"
---

Two adversarial reviews over the assembled multi-branch diff each returned a blocker that the same class of review had already passed over on the individual branch, and both blockers were branch-local rather than merge-only: a bare git-directory existence check that let a worktree read and enforce an unrelated repository's private name store, and a test fixture built from a live session identifier and carried past the repository's own new detector by three separate escapes in one commit. The lesson is not that more review is better. A reviewer reading one branch in isolation judges the diff against itself, while a reviewer reading the assembled tree judges it against the repository's stated invariants and notices a change that contradicts one. Worth recording in the reviews charter as the reason an integration-level pass is not redundant with the per-branch pass, and worth giving the per-branch reviewer the invariants explicitly rather than hoping it infers them.