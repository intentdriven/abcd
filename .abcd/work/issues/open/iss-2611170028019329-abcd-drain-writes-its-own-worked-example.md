---
schema_version: 1
id: "iss-2611170028019329"
slug: "abcd-drain-writes-its-own-worked-example"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "security review of the consent-flag worked-example test"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/drain.go"
remedy: "extend the consent-flag test to every visible command's own Example text, and either drop --yes from drain's example or declare it required in drain --drive's usage line, which the drive's owner decides"
---

abcd drain writes its own worked example on the command (abcd drain --drive --yes --max 3), and the test that holds worked examples off a consent flag reads only the shared example table, so drain's own example is checked by nothing; drain --drive refuses inside an agent's session, which bounds it today
