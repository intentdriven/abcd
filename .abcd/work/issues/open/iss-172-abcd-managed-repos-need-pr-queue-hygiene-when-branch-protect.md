---
schema_version: 1
id: "iss-172"
slug: "abcd-managed-repos-need-pr-queue-hygiene-when-branch-protect"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "explainability-plan merge stall"
found_at: ".github/workflows"
related_intents: [itd-107]
related_issues: [iss-178]
remedy: "Waits on the itd-107 planning interview (ruling H): either way, render rung 1 into the merge-lifecycle steps itd-107 emits (after arming auto-merge, poll mergeStateStatus and run `gh pr update-branch` on BEHIND, strict never relaxed); if rung 2 is scaffolded, emit a merge queue with a `merge_group` trigger on every required workflow where the forge offers one, and otherwise an update-branch workflow on a GitHub App token, proven by a scaffold test that the emitted files carry that trigger or name that secret and keep strict true; if rung 1 only, close rung 2 as wontfix naming the credential cost."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25; rulings-owed H): In itd-107's planning interview: scaffold the update-branch workflow and its credential (rung 2), or keep rung 1 as protocol only?"
---

abcd-managed repos need PR-queue hygiene when branch protection requires up-to-date branches: a queued auto-merge PR whose base moved sits BEHIND forever because GitHub auto-merge never updates branches. Two delivery rungs per basics-built-in/SOTA-delegated: (1) protocol-level now — after arming auto-merge, poll mergeStateStatus and run gh pr update-branch on BEHIND (works with the agent's own token, CI triggers normally); (2) scaffolded CI later — abcd scaffolds an update-branch workflow into managed repos, which requires a PAT/GitHub-App secret because the default GITHUB_TOKEN's pushes do not trigger workflows (the recursion suppression trap), or points at a merge queue where the plan allows. Same scaffolding family as itd-93's release gate; same merged-work-hygiene family as itd-95.

Second occurrence, 2026-08-11, on a single-commit record change (PR #213): auto-merge armed cleanly, an unrelated branch merged and moved the base, and the pull request then held at BEHIND indefinitely with auto-merge still armed and every check green. Updating the branch cleared it, CI re-ran on the updated head, and the merge completed unattended. Rung 1 as written is confirmed against the agent's own token; the 2026-08-11 decision is to ship rung 1 first per script-first-mvp and carry rung 2's credential question into the itd-107 grill.

INVARIANT, recorded because the convenient fix is the wrong one: the strict status-check policy is what makes concurrent additive auto-merge safe, and it is never relaxed to smooth this stall. Record ids are minted under a filesystem lock that serialises two sessions sharing one checkout but does not span checkouts, so two branches can each mint the same id and each pass the record gate in isolation — the duplicate exists only in the merged result, and strict is what forces that result to be gated before the second merge lands. Updating the branch resolves BEHIND while preserving the gate; dropping strict resolves it by removing the gate.

## Remedy grounds (2026-09-29)

- SOTA check: GitHub's merge-queue guide (https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue, read 2026-09-29) gives the up-to-date guarantee without the author updating the branch, but only on organisation-owned public repositories or Enterprise Cloud private ones, and required Actions checks need the `merge_group` trigger; the workflow-trigger guide (https://docs.github.com/en/actions/writing-workflows/choosing-when-your-workflow-runs/triggering-a-workflow, read 2026-09-29) confirms events made with GITHUB_TOKEN start no new workflow run, dispatch events excepted.
- abcd's own repository already runs a queue with strict kept (`.abcd/work/rulesets/main-protection.json`), so the queue is the preferred rung where available and the App-token workflow the fallback; the App shape matches ruling H2 for dependency pushes.
- Rejected: relaxing strict to clear BEHIND, which removes the gate the record's invariant keeps.
