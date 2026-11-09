---
schema_version: 1
id: "iss-2611081341084748"
slug: "make-preflight-s-secret-scan-passed-a"
severity: "minor"
category: "process"
source: "agent-observation"
found_during: "run-2611080653222827 lane-1 landing, PR gitleaks check"
origin: researcher-authored
production_mode: hand-written
found_at: "scripts/preflight-scans.sh"
remedy: "preflight-scans.sh secrets runs the gitleaks version ci.yml pins: when PATH's differs it fetches that release, checksum-verified, into a cache under the abcd home (as scripts/pinned-toolchain.sh does for Go), and refuses rather than warns when it cannot, naming the skew"
resolution: "make secrets-check now scans with the gitleaks ci.yml pins: a PATH copy at another version, or none, is replaced by that release, sha256-checked against scripts/gitleaks-checksums.txt, and the scan refuses when it cannot be fetched or verified; CI's install checks against the same file"
impact: internal
---

make preflight's secret scan passed a branch that CI's gitleaks then failed: the scan ran the gitleaks on PATH (8.30.1), not the 8.24.3 ci.yml pins, and a version skew only warns, so a clean local verdict that is not CI's still mints the push receipt. Two token-shaped test literals reached a pull request this way and failed its gitleaks check after the push.
