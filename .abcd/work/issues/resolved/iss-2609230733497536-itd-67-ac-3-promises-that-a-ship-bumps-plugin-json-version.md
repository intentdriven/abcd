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
resolution: "Every release now publishes a pinned, version-stamped plugin archive: launch ship renders abcd-plugin-vX.Y.Z.zip reproducibly from its tree and pins its release URL and sha256 in .claude-plugin/marketplace.json; release.yml re-renders it from the tagged commit with the new launch archive --verify, in verify and again in the publish job, refuses a digest mismatch, and checksums, attests and uploads the zip with the binaries; /plugin update then installs exactly the cut release, which the harness verifies against the pinned digest (adr-2609231048308186, itd-67 AC1 amended on rulings E1/E2)."
impact: additive
resolved_by:
  commit: "657db9a0"
---

itd-67 ac-3 promises that a ship bumps plugin.json.version, updates marketplace.json and that /plugin update abcd pulls the new version; delivered reality: launch ship stamps the version only into a payload staged outside the repository when --payload-dir is passed (internal/surface/cli/ship.go), release.yml uploads binaries and checksums.txt only, and the marketplace source ./ is the git tree whose committed manifests carry no version — so the version-stamped manifests are produced by nothing in the release path and consumed by nothing; the installed plugin's version arrives through the bootstrap binary download alone. Found by the itd-67 fidelity audit (rcp-7af1556ce4f7).

## Grounds

- pursued: a ship pins the archive digest a re-render of the committed tree reproduces (TestLaunchShipPinsTheReleaseArchive), and the release refuses a drifted pin (TestLaunchArchiveVerifyRefusesADrift); it would be shown wrong by the first release past v0.9.0 whose verify step cannot reproduce the committed pin, or whose published zip differs from the pinned digest when the harness installs it
