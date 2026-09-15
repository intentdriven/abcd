---
schema_version: 1
id: "iss-2609012111168716"
slug: "ahoy-install-from-a-terminal-finds-no-data-dir-and-degrades-to-the-legacy-symlink"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "ship-audit-itd-130-itd-132-2026-09-01"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy/data_dir.go"
resolution: "Fixed in 6e05c449: the bootstrap now stamps the data dir it provisioned the root from into <plugin-root>/.data-dir, and pluginDataDir falls back to that stamp (validated: absolute, existing directory) when CLAUDE_PLUGIN_DATA is unset, so the documented terminal ahoy install lands the owned copy; the artefact is still re-hashed against the recorded binary_sha256 before any copy, and the degradation note names every source tried."
impact: fix
resolved_by:
  commit: "6e05c449"
---

pluginDataDir reads CLAUDE_PLUGIN_DATA from the environment, which the harness exports only to hook processes. The bootstrap's success notice and docs/how-to/install.md direct the user to run '<plugin-root>/abcd ahoy install' from their terminal, where that variable is unset, so ahoy install finds no cache and degrades (loudly) to the spc-21 pinned symlink into the plugin root, the exact shape spc-35 replaced because it dangles after the next plugin update. Every install that follows the documented instruction therefore lands the legacy shape. Surfaced by the itd-132 fidelity audit (receipt rcp-acde3e9ce729) against ac-4. Directions, none adopted: ahoy install resolves the data dir from recorded provisioning metadata (the bootstrap knows the path and could stamp it into .binary-meta or path-entry) rather than the environment; or the notice tells the user to run it via the hook path; or the docs state the degradation.

## Grounds

- pursued: recording the provisioning-time data dir beside the binary is enough for a terminal ahoy install to reach the verified cache; a terminal install that still lands a symlink on a cache-provisioned root, or a stamp that bypasses the binary_sha256 re-verification, would show it wrong.
