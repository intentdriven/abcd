---
schema_version: 1
id: "iss-2608210934566230"
slug: "spc35-owned-copy-breaks-terminal-root"
severity: "major"
category: "bug"
source: "user-observation"
found_during: "spc-35 adversarial review 2026-08-21"
found_at: "internal/core/ahoy/data_dir.go"
resolution: "writePathEntry records a home-scoped plugin_root= and resolvePluginRoot reads it from ~/.abcd/path-entry, so terminal verbs can route home without CLAUDE_PLUGIN_DATA"
impact: fix
resolved_by:
  commit: "bdc5f2ab986ba3014131e1f226004215539d58e2"
---

spc-35 FIX-FIRST (ruthless review): replacing the PATH symlink with an owned regular-file copy severs the executable-ancestor route back to the plugin root, so every plugin-root-dependent verb no-ops when abcd is invoked BY NAME from a plain terminal — which is where bootstrap.sh itself tells the user to run 'ahoy install', and where 'abcd update' runs. Root cause: CLAUDE_PLUGIN_DATA is exported only to hook processes, not terminals (itd-132 records this). Two failures: (1) resolvePluginRoot() walks the copy's ancestors (~/.local/bin -> ~ -> /Users), none has hooks/, returns ('',false) -> Detect emits plugin.root_missing, Install/version/staleness no-op (store.go:96-108; store_symlink_test.go pins the OLD symlink mechanism and stays green only because it builds the symlink by hand, not via ahoy install). (2) path-entry lives in the data dir resolved solely from CLAUDE_PLUGIN_DATA (data_dir.go:12), so from a terminal isOwnedCopyFile is false -> uninstall leaves the binary ('not a symlink; left untouched'), abcd reports its OWN binary as symlink.foreign, and abcd update's RefreshPathEntryDigest no-ops so path-entry desyncs permanently. Contradicts AC 4 and the CHANGELOG. Fix (ruthless): record plugin_root= in the provenance and add it as a resolvePluginRoot candidate; write the provenance to a home-scoped abcd-owned location derivable from the installed copy, consulted when the var is absent; extend store_symlink_test.go with the copy layout; update commands/ahoy.md + docs/reference/cli/commands.md (still say 'symlink'). Note the recorded plugin_root is itself a GC'd cache dir, so the hook-side bootstrap must rewrite it on each provision.