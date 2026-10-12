---
schema_version: 1
id: "iss-2610110941283143"
slug: "a-push-to-an-open-pull-request-started"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "merge-pipeline evidence from abcd-0b, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "have the loop and the landing steps check, after a push to an open pull request, that every required check context started for the new head within a few minutes, and say which did not with the close-and-reopen remedy; record each sighting here until the cause is known"
---

A push to an open pull request started only some of its workflows: after b36b551ac was pushed to #904 (2026-10-10, about 22:50Z) only external-review ran, ci and attribution never triggered, and the pull request sat BLOCKED with eight of its nine required checks missing until it was closed and reopened. Nothing noticed: the checks were not failing, only absent.

**Likely cause (2026-10-11, abcd-0b).** GitHub runs no `pull_request` workflow while it reports a pull request as mergeable CONFLICTING (DIRTY), because it cannot build the test merge commit; `pull_request_target` workflows such as external-review run on the base branch and still fire. #919 and #920 showed exactly this on 2026-10-11: a body edit and a close and reopen started only external-review. A local merge of main into #919 was clean, so the conflict is probably the forge ignoring the `.abcd/work/DECISIONS.md` union driver, as with #902 (iss-2610110908443341; root cause iss-2609100507439414). Whether #904 was conflicting at 22:50Z on 2026-10-10 is not yet checked.
