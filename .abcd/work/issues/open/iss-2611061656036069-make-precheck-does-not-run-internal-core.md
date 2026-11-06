---
schema_version: 1
id: "iss-2611061656036069"
slug: "make-precheck-does-not-run-internal-core"
severity: "minor"
category: "tech-debt"
source: "impl-review"
found_during: "follow-up release batch"
origin: researcher-authored
production_mode: hand-written
found_at: "Makefile"
remedy: "Add the test by name to the precheck recipe and to the records-only rule in the Makefile, and read this beside iss-2610230048490215, whose mark on audit tests would stop the list being kept by hand."
---

make precheck does not run internal/core/launch's command-surface test, TestCommandSurfaceResolvesBinaryFromPluginRoot: it runs internal/core/launch/scaffold but not that package, so a fenced bare binary name in a command page passed precheck and failed 42 minutes into a preflight.
