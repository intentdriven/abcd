---
id: dcn-2610180905291297
slug: a-declared-dependency-constrains-only
date: 2026-10-18
---
2026-10-18 — A declared dependency constrains only the landing (facilitator-on-delegation, itd-2610170634394993 planning prep): a branch that waits on an open pull request is held only from being queued; its merge from the default branch, its fixes, its review and its preflight run now, and only its landing waits for the dependency, so the stop check counts a stacked branch with an open base as owed work (grounds: iss-2610170619247185, second instance; the draft's first acceptance criterion; the decision that a waiting branch is rebuilt by merging)
