---
schema_version: 1
id: "iss-2609230733497536"
slug: "itd-67-ac-3-promises-that-a-ship-bumps-plugin-json-version"
severity: "major"
category: "drift"
source: "agent-finding"
found_during: "autonomous run 2026-09-23 fidelity audit"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/ship.go"
---

itd-67 ac-3 promises that a ship bumps plugin.json.version, updates marketplace.json and that /plugin update abcd pulls the new version; delivered reality: launch ship stamps the version only into a payload staged outside the repository when --payload-dir is passed (internal/surface/cli/ship.go), release.yml uploads binaries and checksums.txt only, and the marketplace source ./ is the git tree whose committed manifests carry no version — so the version-stamped manifests are produced by nothing in the release path and consumed by nothing; the installed plugin's version arrives through the bootstrap binary download alone. Found by the itd-67 fidelity audit (rcp-7af1556ce4f7).
