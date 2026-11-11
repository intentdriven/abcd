---
schema_version: 1
id: "iss-2611110905076301"
slug: "the-preflight-s-govulncheck-gate-scripts"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/preflight-scans.sh"
remedy: "Build CI's govulncheck version with go install into a private temporary directory and run that binary, so its exit code arrives unchanged, for the head scan and the fork-point scan alike; make the test stub behave as the real go does (go run maps any non-zero exit to 1) and add a test that a finding is judged, not skipped."
resolution: "scripts/preflight-scans.sh builds CI's govulncheck with go install and runs the binary for the head and fork-point scans, so a finding's exit 3 reaches the judgement; the test stub now behaves as the real go run does, and a new test holds that a finding is judged, not skipped"
impact: internal
---

The preflight's govulncheck gate (scripts/preflight-scans.sh vulncheck) ran the tool through go run, which reports the program's exit 3 (a finding) as its own exit 1 with an 'exit status 3' line. The script reads every exit other than 0 and 3 as a scan that could not run, so a real finding was reported as SKIPPED and the gate passed, while CI's govulncheck job failed on the same tree. The fork-point scan in ids_not_at had the same flaw, so a finding there read as an unscanned fork point. The test stub for go passed the tool's exit straight through, unlike the real go run, which hid it.
