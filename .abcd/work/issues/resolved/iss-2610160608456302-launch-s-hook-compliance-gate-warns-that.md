---
schema_version: 1
id: "iss-2610160608456302"
slug: "launch-s-hook-compliance-gate-warns-that"
severity: "minor"
category: "bug"
source: "review-followup"
found_during: "v0.15.0 release review"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
remedy: "Teach the hook-compliance gate to accept a hook command that runs the script through sh or bash"
resolution: "the hook-compliance gate judges a hook file's mode only where hook text executes it; a script run through sh or bash is no longer reported not executable"
impact: fix
---

launch's hook-compliance gate warns that hooks/*.sh are not executable in the payload (mode 100644), but hooks.json invokes each as `sh "<path>"`, so the mode is irrelevant; the warning is a false positive (unchanged since v0.14.2)
