---
id: dcn-2608310000000010
slug: the-cold-reading-evals-ci-job-is-not
date: 2026-08-31
---
2026-08-31 — The cold-reading evals' CI job is NOT added to the branch
ruleset's required-check list yet, and the sequencing is the decision rather
than the delay. The job and its make target exist only on
`experiment/cold-reading`; neither is on `main`, which was checked rather than
assumed. A required status check blocks a pull request until that context
reports, and a context no workflow on the base branch produces never reports —
so arming `cold-reading-evals` on the active `main protection` ruleset before
the workflow reaches `main` would wedge every merge to `main`, including the
very pull request that would deliver the workflow. The order is therefore:
merge the workstream to `main` first, confirm the job runs and reports its
context on a real pull request, and only then add it to the live ruleset and
to the committed mirror at `.abcd/work/rulesets/main-protection.json` in one
change. Rejected: updating the mirror now to record the intent, because a
mirror asserting a required check that is not required is a false record of
exactly the kind this phase spent the day removing, and it would read as
done. Until the arming lands, itd-186's always-run lane runs on every pull
request and gates none of them, which is a green for work that did happen but
binds nothing — the weaker half of the failure spc-64 was written against.
Tracked at iss-2608311051046981.
