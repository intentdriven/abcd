---
schema_version: 1
id: "iss-2609251902439938"
slug: "a-checkout-without-the-previous-release-s-tag-reads-as-a"
severity: "major"
category: "bug"
source: "impl-review"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/launch_deep.go"
---

A checkout without the previous release's tag reads as a first launch: launchParityInput (internal/surface/cli/launch_deep.go) leaves the baseline empty when no release tag is listed, and PayloadParity (internal/core/launch/parity.go) reports every payload path as added with exit 0. A git clone --no-tags of a tree whose CHANGELOG.md dates v0.10.0 reports parity source none and 130 added; a clone --depth 1 reports the retention refusal for a shallow checkout beside the same first-launch diff. The configured-baseline hard error (itd-66 AC7) is defeated in the commoner shape: a fork, a mirror, or tags never pushed.
