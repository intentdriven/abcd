---
schema_version: 1
id: "iss-2611101319460145"
slug: "the-narrow-gate-runs-only-go-test-so-the"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "autonomous run of 7 Nov 2026"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/load_narrow.go"
remedy: "Add a narrow form that runs the text and docs gates over the changed files only, in seconds; a test runs it on a change holding a curly apostrophe and finds it refused."
---

The narrow gate runs only go test, so the text and docs gates first see a lane's change at precheck. On 9 Nov a lane's narrow gate passed after 33 minutes, and make precheck then failed on two text rules in the change, a spelled-out home folder in a fixture and a curly apostrophe in a regular expression, costing a second precheck.
