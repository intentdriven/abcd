---
schema_version: 1
id: "iss-2608271014245874"
slug: "the-abcd-version-binary-missing-recovery"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "external-issue-494-review"
resolution: "commands/version.md gains a 'When no binary resolves' section that leads recovery with the no-Go path (bootstrap re-provision on a networked restart, plugin reinstall, or the CLI one-liner) and frames go run/go build as source-checkout or unsupported-platform only, so a missing binary no longer reads as Go-required (#494)."
impact: fix
---

The /abcd:version binary-missing recovery guidance over-promotes the build-from-source rung, so an agent facing a failed provisioning tells the user to install Go — which is a dependency of neither supported install route. Both the plugin route (hooks/bootstrap.sh fetches a checksum-verified prebuilt release binary) and the CLI one-liner download a prebuilt binary; Go is only for building from source in a contributor checkout or an unsupported platform. commands/version.md's Binary resolution section lists 'go run ./cmd/abcd' as a peer rung without telling the reader that a genuinely missing binary is recovered no-Go: bootstrap re-provisions on the next networked session, reinstalling the plugin, or the CLI one-liner. External GitHub issue #494: a contributor concluded Go was required.