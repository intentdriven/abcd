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
resolution: "All nine literal pins are gone: every setup-go step in the four workflows, and all three in the scaffold template, now carry go-version-file: go.mod. The scaffold substitution that fed the template's literal is removed with them — a scaffold-time snapshot of the adopter's go directive went stale the moment they bumped it, and it was a value that had to be validated against an injection-safe allowlist before it could be written into YAML, where a fixed go.mod needs neither. Report.GoVersion survives as what the workflows will RESOLVE, which is the fact an adopter wants before their first tag. TestWorkflowGoVersionsMatchSubstitutions, which policed the nine literals against the substitution, is replaced by TestEverySetupGoResolvesTheToolchainFromGoMod, which refuses the literal outright."
impact: fix
---

The format gate exists because two gofmt versions judge a tree differently, and it goes to some trouble to derive the toolchain version from the go directive in go.mod so that one declaration governs. The workflows then declare it again: nine setup-go steps across four workflow files pin the version as a literal string, four of them in the CI workflow and the rest in the release, site and screenshot workflows. The action already accepts a reference to the go.mod file in place of a literal, which is the form that removes the second spelling. Verified by grepping the workflow tree. It matters in exactly the direction the format gate was built for: bump the go directive and forget one of the nine, and CI compiles and tests on the old toolchain while the format gate resolves and fetches the new one, so the gate and the lane disagree about which gofmt is correct, reintroducing at the CI boundary the skew the gate refuses to allow locally. Fix direction: replace every literal pin with the go.mod file reference so the go directive is the only place the version is written. Detector: a change to the go directive in go.mod must change which toolchain every workflow job runs, with no workflow edit.

## Grounds

- pursued: the go directive is now the only place the toolchain is written, so bumping it moves every lane and the format gate together with no workflow edit; it would be shown wrong if setup-go's go-version-file resolution ever diverged from the toolchain go build selects — the two read the same directive today, and the release lane's patch-precision, which iss-289 was about, survives because go.mod declares a patch
