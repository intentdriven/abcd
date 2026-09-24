---
schema_version: 1
id: "iss-2608210934566221"
slug: "hook-binary-to-persistent-data-dir"
severity: "major"
category: "architectural-insight"
source: "user-observation"
found_during: "plugin-update post-mortem 2026-08-21"
found_at: "hooks/bootstrap.sh"
related_intents: [itd-132]
resolution: "binary and binary-meta relocated to the persistent data dir owned copy (itd-132/spc-35); plugin cache re-clones no longer discard the artefact"
impact: fix
---

Relocate the hook binary and .binary-meta from the plugin cache dir to the harness persistent data dir (CLAUDE_PLUGIN_DATA, ~/.claude/plugins/data/abcd-abcd-marketplace/). The harness re-clones every update into a fresh commit-stamped cache dir and its docs state extra files there are NOT preserved across updates, so the ~11MB binary is re-fetched on every plugin update by design. The data dir survives updates and is deleted only on full uninstall. Fixes in one move: the per-update re-download, the SessionEnd bootstrap-cancellation window, per-version binary disk copies (orphaned cache dirs are GC'd ~14 days after .orphaned_at), and the PATH-symlink rot captured separately. Reworks the itd-105/spc-21 bootstrap fast path and every pluginBinaryPath consumer.