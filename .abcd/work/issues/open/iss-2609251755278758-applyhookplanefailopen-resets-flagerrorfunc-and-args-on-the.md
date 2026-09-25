---
schema_version: 1
id: "iss-2609251755278758"
slug: "applyhookplanefailopen-resets-flagerrorfunc-and-args-on-the"
severity: "minor"
category: "tech-debt"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
---

applyHookPlaneFailOpen resets FlagErrorFunc and Args on the hook plane but not the PreRunE that markUsageErrorsExitTwo now installs on every command, so a flag group declared on guard hook or hook * in future would refuse at exit 2, the host's BLOCK, bypassing iss-269's fail-open (internal/surface/cli/cli.go:138; hypothetical today: no hook command declares a group; review2-consolidate nit).
