---
schema_version: 1
id: "iss-222"
slug: "install-dev-silent-noop-over-unowned"
severity: "minor"
category: "ux"
source: "manual-test"
found_during: "manual-capture"
found_at: "internal/core/ahoy/apply.go"
resolution: "stepSymlink refuses loudly on binTargetForeign instead of silently keeping the foreign entry"
impact: fix
---

ahoy install --dev against an existing unowned track-latest wrapper at the target path performs no bin write, reports no symlink/shadow gap, and leaves install_mode empty — a silent no-op where loud staging requires either adopting the equivalent wrapper, refusing with a named gap, or reporting the skip on the result