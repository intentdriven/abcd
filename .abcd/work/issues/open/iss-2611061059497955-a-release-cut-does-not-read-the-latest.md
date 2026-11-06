---
schema_version: 1
id: "iss-2611061059497955"
slug: "a-release-cut-does-not-read-the-latest"
severity: "minor"
category: "bug"
source: "agent-observation"
found_during: "v0.16.2 release gate"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/auto-release.yml"
remedy: "Have the cut's preview and `abcd land` of a release branch name the latest completed nightly's verdict, using the kept scheduled-workflow verdict the status reader already holds, and refuse or warn when it is red."
---

A release cut does not read the latest nightly. The nightly on the commit just before the v0.16.2 cut had failed its macOS suite while the cut's pull request sat in the merge queue, and nothing said so: the failure surfaced about 45 minutes after the merge, when the cut's own gate failed the same way.
