---
schema_version: 1
id: "iss-2610180542030806"
slug: "the-macos-leg-of-the-check-job-runs-one"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
remedy: "Run the ubuntu leg's plain-only guard step on the macOS leg too, wherever that leg runs its suite, and hold it with the same derived list"
resolution: "The macOS leg runs the plain-only guards by name after its race pass, in a step the drift test holds to the same derived list as the ubuntu leg's"
impact: internal
---

The macOS leg of the check job runs one race-enabled pass, so the tests that assert only without the race detector (cost guards that skip under -race or drop their CPU bound there, 43 of them on 2026-10-18) run there and assert nothing: their bounds are checked on the ubuntu leg only, never on macOS. The workflow comment said dropping the plain pass dropped no test, which those tests had made untrue.

## Grounds

- pursued: the macOS leg grows by about a minute and stays well inside its 60-minute ceiling; a merge-group macOS run above 50 minutes, where the warning fires, would show it wrong
