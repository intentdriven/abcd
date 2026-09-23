---
schema_version: 1
id: "iss-2608231607594913"
slug: "cloudflare-git-integration-branch-builds-run-for-pull-reques"
severity: "major"
category: "drift"
source: "user-observation"
found_during: "pre-merge-disclosure-remediation"
found_at: "wrangler.jsonc"
deferred_after: "v0.9.0"
deferral_reason: "Ruled by the product thinker at the 2026-09-23 run A interview (M12: the host dashboard is authoritative (wrangler.jsonc now says so), and a check that reads the host's real branch-build setting and flags a mismatch is ruled; it needs a read-only host credential, a secret-handling design, so planning is owed)."
---

Cloudflare Git-integration branch builds run for pull request branches even though wrangler.jsonc records them as disabled, so an unmerged commit reaches a third-party build system before review. wrangler.jsonc carries 'automatic production builds: disabled' and 'automatic branch builds: disabled', kept in-tree because the dashboard setting has no other durable record. On 2026-08-23 a Cloudflare build ran for a pull request branch at 10:00 UTC and the integration bot posted a successful-deployment comment naming the build id. That build was not Actions: .github/workflows/site.yml triggers on workflow_call, workflow_dispatch and push to main only, and it did not run on that commit; the workflow header itself states that the main-push preview replaces the Cloudflare branch builds that ran only the version command. Two consequences. Operationally, every pull request commit is cloned by a third party before merge, and Cloudflare exposes no delete endpoint for a build record or its logs, so whatever reaches a build log cannot be withdrawn without a vendor support request. Durably, the in-tree record is wrong in the confident direction: a reader checking whether branch builds run gets a clear no. Suggested direction: reconcile the dashboard setting with wrangler.jsonc and decide which is authoritative. A maintainer decides; this record reports.

Second source, independently verified: the drift is not confined to one branch. A `Workers Builds: abcd` check run is present on the head commit of every pull request opened in the repository on 2026-08-23 — #447, #451, #455, #460, #465, #470, #471 and #473, eight of eight. Surfaced by a peer session from the PR check lists, then confirmed here against the check-runs API for each head SHA independently of that report. So the Git integration builds every pull request branch, continuously, rather than intermittently or for one branch: the recorded `automatic branch builds: disabled` describes no observed build at all.


Refines `iss-2608220150157502` (cloudflare-branch-builds-run-only-the-version-command, 2026-08-22, same `found_at`). That record captured branch builds running the wrong command, and recorded the adr-48 design as replacing them with an Actions preview on push to main while turning Cloudflare's automatic builds off. The link is `refines` rather than `duplicates`: the earlier record states the intended end state, this one is the evidence it was not reached. Read together they say the disabling was designed, recorded as done in `wrangler.jsonc`, and never took effect for branch builds — which is why the drift survived a record that a reader would reasonably treat as current.


On the category, which is `drift` to match the record this refines: read the label as the subject area, not as the trajectory. This is not drift in the true-then-false sense, and the distinction changes the remedy. Drift is a record that was accurate and decayed, so it is repaired by regenerating it from a source of truth. This line had no source of truth to regenerate from: it narrates an external dashboard setting that nothing in the tree can read, so its accuracy was never a property anything local could establish, and it was wrong from the moment it was written rather than becoming wrong later. `inconsistency` would carry the trajectory-neutrality better; `drift` is kept so the pair with `iss-2608220150157502` reads as one subject, and this paragraph carries what the label cannot. The remedy is correspondingly different: either a check that reads the real Cloudflare state, or an explicit demotion of the line from claim to note. Regeneration is not available.

What makes the survival time worth recording separately from the fact: the line does not read as an unchecked assertion, it reads as a completed decision. The adr-48 intent to turn the builds off and the assertion that they are off were written by the same hand in the same file. A record that looks wrong invites a check; a record that looks finished does not get re-read, because re-reading it would be second-guessing a decision rather than verifying a fact.
