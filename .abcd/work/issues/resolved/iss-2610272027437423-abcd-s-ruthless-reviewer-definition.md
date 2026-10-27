---
schema_version: 1
id: "iss-2610272027437423"
slug: "abcd-s-ruthless-reviewer-definition"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "Mac Studio crash-recovery run 2026-10-27, coordinator's load log"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/ruthless-reviewer.md"
remedy: "In agents/ruthless-reviewer.md and agents/docs-currency-reviewer.md, replace the instruction to run tests with a read-only rule: the reviewer runs no tests, builds or lint, and asks the author for the test evidence (gated preflight output, or a named test watched fail then pass); a finding that needs a run to confirm is reported as such, with the command the author should run through the load gate."
resolution: "Both reviewer definitions now run no tests and ask the author for gated test evidence; the docs-currency reviewer keeps its deterministic docs lint pass."
impact: fix
resolved_by:
  commit: "519e9de09894e79b2b33cf5ad9e52de870cbaaaa"
---

abcd's ruthless reviewer definition tells the reviewer to run the project's test and lint commands, and the docs-currency reviewer to run the tests that pin documented behaviour. On a machine several sessions share, every review then runs a go test that no load gate admits, beside the preflight the gate did admit. On 2026-10-27 the reviewers of three lanes did so: lane D's at 19:22Z, when the one-minute load reached 71.96 on 16 cores and then 97.61 at 19:24Z; lane C's at 20:26Z, at a load of 60.32; and lane A's step-1 review of the merge gate. Each of those times, the admitted preflight's test stage was running at go test -p 16. A reviewer reads a copy and judges it; the test run is the preflight's job.

## Grounds

- pursued: reviewer test runs outside the load gate overloaded the shared machine three times on 2026-10-27
