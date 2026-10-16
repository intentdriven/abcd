---
schema_version: 1
id: "iss-2610160858232720"
slug: "launch-s-hook-compliance-gate-treats-a"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "v0.15.0 deferred findings, PR1, 2026-10-16"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/installsurface.go"
remedy: "Treat a file a hook sources with . or source, or reads (cat, <), as not executed, in hookExecutedRefs; test both forms against the hook-row judgement."
---

launch's hook-compliance gate treats a plugin-root file that a hook sources (. "$CLAUDE_PLUGIN_ROOT/x") or only reads as executed directly, so it judges that file's mode and can warn that a sourced file is not executable, the same false-positive shape iss-2610160608456302 fixed for files handed to sh or bash. No abcd hook sources or reads a plugin-root file today. Found while fixing iss-2610160608456302, 2026-10-16.
