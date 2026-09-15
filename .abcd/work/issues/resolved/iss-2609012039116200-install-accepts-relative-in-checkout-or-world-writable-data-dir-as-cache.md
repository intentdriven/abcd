---
schema_version: 1
id: "iss-2609012039116200"
slug: "install-accepts-relative-in-checkout-or-world-writable-data-dir-as-cache"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous-run-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/data_dir.go"
resolution: "ahoy install now ignores a CLAUDE_PLUGIN_DATA that is relative, lies inside the repository being installed, or names a world-writable directory: dataDirHazard names the shape, ownedCopySourceReady reports no source, and install degrades to the existing loud pinned-symlink path with a note saying which variable was ignored and why. Detection agrees — no symlink.legacy heal is offered from such a cache. TestInstallIgnoresRelativeDataCache, TestInstallIgnoresDataCacheInsideTheCheckout and TestInstallIgnoresWorldWritableDataCache pin all three shapes; each was watched copying the artefact into the PATH target before the change."
impact: fix
---

Sub-finding of GHSA-4q78-ccfv-f374 that does not need the design decision: `ahoy install` uses any CLAUDE_PLUGIN_DATA value as the verified cache, including one that is relative, lies inside the repository being installed, or is world-writable. `internal/core/ahoy/data_dir.go:pluginDataDir` returns the env value unexamined and `owned_copy.go:ownedCopySourceReady` plus `apply.go:installOwnedEntry` copy from it. The harness's persistent per-plugin data directory (spc-35) is always absolute, never inside a project checkout, and never world-writable, so refusing those shapes is a strict hardening of the current contract: a relative value resolves against the checkout the verb runs in, an in-checkout value blesses committed bytes as the owned PATH binary, and a world-writable cache lets any local user swap both artefact and record. The fix must establish that such a data dir is ignored with a note naming why, that install then degrades to the existing loud pinned-symlink path exactly as when no cache exists, and that a harness-shaped data dir still installs the owned copy; the display-only readers of the same variable (`vintage.go:readPinnedTag`, the skew notice) make no trust decision and are unchanged.

## Grounds

- pursued: the harness never produces these shapes, so refusing them is a strict hardening that costs a real install nothing and needs no decision; binding the cache to an attestation the environment cannot supply stays open on the parent record GHSA-4q78-ccfv-f374 (iss-2609012039102770)
