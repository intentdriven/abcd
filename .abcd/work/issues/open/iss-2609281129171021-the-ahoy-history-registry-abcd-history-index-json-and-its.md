---
schema_version: 1
id: "iss-2609281129171021"
slug: "the-ahoy-history-registry-abcd-history-index-json-and-its"
severity: "minor"
category: "security"
source: "user-observation"
found_during: "autonomous run A resumed 2026-09-25: lane drainHome"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/store.go"
---

The ahoy history registry (~/.abcd/history/index.json and its lock) is created and written through a symlinked ~/.abcd: historyRoot joins the home directory and withHistoryLock and bootstrapHistory call os.MkdirAll, which follows the link, so on a machine whose ~/.abcd is symlinked into a dotfiles checkout the registry of every managed repository, remote URLs included, lands in that repository. Its sibling stores refuse a symlinked level through their create-then-prove seam (transcripts, voyage, lab, inbox, runs use fsutil.EnsureRealDir/EnsureRealDirAll/IsRealDir), and every file abcd trusts in ~/.abcd refuses the link through fsutil.HomeScopeLink (iss-2609281017573862, iss-2609260958587561). The registry is a store, not a trust declaration, and refusing it changes ahoy install's registration on such machines, so it was left for its own ruling: route it through fsutil.EnsureRealDirAll on ".abcd/history" below the home directory and say what the install does when the registry cannot be created.
