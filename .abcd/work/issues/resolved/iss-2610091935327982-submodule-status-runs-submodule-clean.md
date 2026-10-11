---
schema_version: 1
id: "iss-2610091935327982"
slug: "submodule-status-runs-submodule-clean"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "security-drain-2026-10-09 lane W sweep"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/launch/gates.go"
remedy: "Pass --ignore-submodules=dirty on those four working-tree reads (the flag, since -c diff.ignoreSubmodules loses to a repo's submodule.<n>.ignore=none), proved by a test with a checked-out submodule whose own config names a clean filter, watched fail first."
resolution: "The four filter-free working-tree reads (gitutil.Status, the launch dirty check, the consistency dirty check, the release-receipts status) and the agent-diff path list pass --ignore-submodules=dirty, so git starts no status inside a checked-out submodule and no filter the submodule's own config names runs; a moved submodule pointer still reads as a change."
impact: fix
---

abcd's filter-free working-tree reads (gitutil.Status, the launch dirty check DirtyTreeFiles, intent consistency dirtyCorpusPaths, the release-receipts status) still run a checked-out submodule's own clean filter: without --ignore-submodules git starts a child status inside each submodule, and FilterOverrides blanks only the filters the superproject defines, so a copied checkout's .git/modules/<sub>/config filter runs on a stale index. Sibling of iss-2610090821548169, found by the security-drain-2026-10-09 lane W sweep; kept uncommitted until its fix lands.
