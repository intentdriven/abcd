---
schema_version: 1
id: "iss-2609090951291799"
slug: "workflows-pin-the-go-version-a-second-time-beside-go-mod"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial-review"
origin: researcher-authored
production_mode: hand-written
found_at: ".github/workflows/ci.yml"
---

The format gate exists because two gofmt versions judge a tree differently, and it goes to some trouble to derive the toolchain version from the go directive in go.mod so that one declaration governs. The workflows then declare it again: nine setup-go steps across four workflow files pin the version as a literal string, four of them in the CI workflow and the rest in the release, site and screenshot workflows. The action already accepts a reference to the go.mod file in place of a literal, which is the form that removes the second spelling. Verified by grepping the workflow tree. It matters in exactly the direction the format gate was built for: bump the go directive and forget one of the nine, and CI compiles and tests on the old toolchain while the format gate resolves and fetches the new one, so the gate and the lane disagree about which gofmt is correct, reintroducing at the CI boundary the skew the gate refuses to allow locally. Fix direction: replace every literal pin with the go.mod file reference so the go directive is the only place the version is written. Detector: a change to the go directive in go.mod must change which toolchain every workflow job runs, with no workflow edit.
