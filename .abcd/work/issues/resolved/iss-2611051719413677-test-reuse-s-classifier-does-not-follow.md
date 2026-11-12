---
schema_version: 1
id: "iss-2611051719413677"
slug: "test-reuse-s-classifier-does-not-follow"
severity: "minor"
category: "tech-debt"
source: "agent-observation"
found_during: "build run, 5 November"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/testreuse"
remedy: "Treat a non-literal program name as evidence that the test must run."
resolution: "Test reuse's classifier now sends a test to the always-rerun line when exec.Command or exec.CommandContext starts a program under a name that is not a string literal (a local variable, a parameter, a constant, an expression), with its own reason in the report."
impact: fix
resolved_by:
  commit: "e235b53861f4b1e74299b81f209139d45b095aeb"
---

Test reuse's classifier does not follow a program name held in a local variable, so a test that starts a program that way can replay locally until the hosted run. The classifier is a best-effort read of test source, and the hosted run reuses nothing.

## Grounds

- pursued: a test that starts a program under a name held in a variable reruns on every local check instead of replaying; a check-tests report that reuses a package holding such a test, or a classifier case with a non-literal program name that returns no reason, would show it wrong
