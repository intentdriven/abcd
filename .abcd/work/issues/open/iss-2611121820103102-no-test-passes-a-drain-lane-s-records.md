---
schema_version: 1
id: "iss-2611121820103102"
slug: "no-test-passes-a-drain-lane-s-records"
severity: "minor"
category: "tech-debt"
source: "agent-finding"
found_during: "intent audit of the landings record"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/land_attribution_test.go"
remedy: "Extend the landed-range attribution test with a drain lane's records commit, and add one fixture test that installs the repository's commit-msg hook and makes the records commit through it."
---

No test passes a drain lane's records commit message, with its drain admission trailer, through the attribution gate and the issue-resolution gate's message form: the attribution test runs both gates over the build forms only, and no test makes the commit through the repository's installed commit-msg hook. The intent audit of the landings record marked its first criterion met with concerns for this.
