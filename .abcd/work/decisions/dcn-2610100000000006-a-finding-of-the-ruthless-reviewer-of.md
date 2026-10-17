---
id: dcn-2610100000000006
slug: a-finding-of-the-ruthless-reviewer-of
date: 2026-10-10
---
2026-10-10 — A finding of the ruthless reviewer of lane-1 of run-2610101926030564 (iss-2610090642392144) is deferred, not fixed in the change that puts a verifier's copy outside every working tree: the bundled `OPINIONS` rule in `internal/core/rules/defaults/rules.json` ("The user's directory is theirs") still allows a verifier's copy in the checkout's `.abcd/.work.local/` tier, against the repository's own `CONCURRENCY` rule, AGENTS.md, itd-193 and the users-directory principle, which now forbid it. Reason: the bundled rule is injected into every managed repository, so changing it is a user-visible change the issue's remedy did not name, and it waits on a person's ruling, recorded as iss-2610102021042657.
