---
schema_version: 1
id: "iss-2610301246136766"
slug: "rs006-in-scripts-check-issue-resolution"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "lane A precheck of fix/load-scaled-test-timeout"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/check-issue-resolution.sh"
remedy: "Match a test name only where no identifier character precedes Test (a dot still may, so pkg.TestX counts), and add a case to check-issue-resolution-cases.sh where a resolution names a function containing Test and passes."
resolution: "RS006 now reads a test name only where no identifier character precedes Test, so a function such as GoTestTimeout names no test while a package-qualified test name still counts. Two cases in scripts/check-issue-resolution-cases.sh pin it: a resolution naming such a function passes, and one naming a qualified test no file defines is refused."
impact: internal
---

RS006 in scripts/check-issue-resolution.sh reads a test name out of any word that contains Test followed by a capital, so a resolution naming a function such as GoTestTimeout is refused as naming a test no _test.go file defines. It refused the load-scaled-timeout resolution, which named machineload.GoTestTimeout, on 30 Oct 2026; the note had to be reworded to pass.
