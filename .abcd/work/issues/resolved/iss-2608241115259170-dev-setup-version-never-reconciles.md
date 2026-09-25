---
schema_version: 1
id: "iss-2608241115259170"
slug: "dev-setup-version-never-reconciles"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "sessionstart-hook-error-investigation"
found_at: "internal/core/ahoy/vintage.go"
resolution: "versionTransitionFrom and detectVersion both treat a dev or unrecorded version on either side as no transition, so a dev-stamped repo neither reports a permanent transition nor raises a version.upgrade gap that flaps the tracked config."
impact: fix
resolved_by:
  commit: "ea96cfc0"
---

A dev-recorded setup_version can never reconcile against a release binary. versionTransitionFrom (internal/core/ahoy/vintage.go:222) guards running == dev but not recorded == dev, so a repo whose .abcd/config.json meta.setup_version was stamped by a local dev build reports a version transition against every release binary, permanently. The recommended fix is a no-op or a flap in exactly the repo that ships the code: ahoy install stamps core.Version of whichever binary runs it (internal/core/ahoy/apply.go:1020 via store.go:21), so abcd ahoy install from a dev PATH entry re-stamps dev and leaves the notice standing, while the plugin binary's install stamps the release version into a tracked file that the next dev install rewrites back. detectVersion (internal/core/ahoy/detect.go:599) carries the same asymmetry and raises version.upgrade as a required gap on the same footing.

## Grounds

- pursued: a repo whose setup_version is dev reports no transition and no upgrade gap under a release binary, and vice versa; shown wrong if TestVersionTransitionFrom or TestDetectVersionIgnoresADevSide sees one
