---
schema_version: 1
id: "iss-2609012020479351"
slug: "testbootstrapfreshinstallselfcheck-in-internal-surface-cli-a"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/bootstrap_freshinstall_test.go"
resolution: "TestBootstrapFreshInstallSelfCheck no longer times the spawn. Its fourth assertion now checks that the provisioned binary's answer is the version report itself: the identity line 'abcd <version>' first, then the vintage and staleness fields, alongside the existing check that the answer names abcd. Proved on a scratch copy against a mutant version verb that prints 'abcd: version report unavailable': the old test passed it, the new one refuses it at all three new assertions; the live tree passes with no duration anywhere in the test."
impact: internal
---

TestBootstrapFreshInstallSelfCheck in internal/surface/cli asserts wall-clock time: it wraps one exec of the provisioned binary answering version in a timer and fails when elapsed exceeds five seconds. It failed on the macOS CI leg after v0.7.0 merged (run 33496438371: 5.77s, 'want about a second') on a tree the merge-queue leg had just passed; locally the same test passes in about one second. The timer measures a single process spawn of an ~18MB freshly written executable, which on macOS carries first-run code-signature validation against a cold page cache, so the number is a property of the runner, not of the code. The test's own comment names the budget as loosened 'so a loaded CI box cannot fail a check about provisioning on a timing wobble', and it then did exactly that. Fourth site of the class recorded by iss-2608292246210181, iss-2608301301041887 and iss-2608290810037763; carried from the session handover into autonomous-run-2026-09-01. The fix must assert the behaviour the gate is about -- the provisioned binary answers version correctly, as the version report, with no Go on PATH -- and carry no duration at all.

## Grounds

- pursued: the gate is about provisioning, so it should assert what the provisioned binary does and not how long the host took to spawn it; if a later flake on this test appears with a behavioural failure message, the assertion was wrong, and if it appears with a timing message, a duration crept back in
