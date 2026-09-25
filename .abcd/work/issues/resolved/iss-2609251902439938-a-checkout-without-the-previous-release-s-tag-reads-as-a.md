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
resolution: "The parity diff refuses a tagless or shallow checkout whose CHANGELOG.md dates a release, naming that release and the remedies; --fetch-baseline reads the verified archive as the only baseline; a tree with no dated release and no tag stays a first launch."
impact: fix
resolved_by:
  commit: "4a5b0683"
---

A checkout without the previous release's tag reads as a first launch: launchParityInput (internal/surface/cli/launch_deep.go) leaves the baseline empty when no release tag is listed, and PayloadParity (internal/core/launch/parity.go) reports every payload path as added with exit 0. A git clone --no-tags of a tree whose CHANGELOG.md dates v0.10.0 reports parity source none and 130 added; a clone --depth 1 reports the retention refusal for a shallow checkout beside the same first-launch diff. The configured-baseline hard error (itd-66 AC7) is defeated in the commoner shape: a fork, a mirror, or tags never pushed.

## Grounds

- pursued: a git clone --no-tags or --depth 1 of this tree previews parity as refused against v0.10.0 and a full checkout still diffs against v0.10.0; a tagless or shallow checkout reporting a first launch, or a no-release tree refusing, would show it wrong
