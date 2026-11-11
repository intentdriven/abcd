---
schema_version: 1
id: "iss-2611101317395313"
slug: "a-build-lane-s-definition-of-done-runs"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/validate.go"
remedy: "Have the validate stage's definition of done run precheck's audit packages and the secret scan for the packages the lane touches, from the default branch's current recipe, before the reviewers are asked; a test changes an exported function so the reach audit fails and finds validation refusing."
---

A build lane's definition of done runs only the tests of the packages it changed, so the repository-wide audits, the secret scan and other packages' tests that exercise the changed rule first run at precheck or preflight after validation, when the loop has no route back to a fix round. On 8 Nov at 00:36 and 00:45 precheck at land failed the export reach audit, because the spec exported a function only tests reached; on 8 Nov at 11:54 it failed a gitutil audit, because a package ran rev-parse --show-toplevel itself; on 8 Nov at 13:55 CI's secret scan failed a token fixture that no review and no lane gate had scanned; on 7 Nov at 14:40 a lane branch cut before #1444 lost a 40-minute preflight to an audit precheck now runs; on 9 Nov at 02:50 a preflight failed a CLI test the lane never ran. Each miss cost a preflight and pushed the fix outside the loop.
