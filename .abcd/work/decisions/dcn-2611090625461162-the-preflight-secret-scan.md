---
id: dcn-2611090625461162
slug: the-preflight-secret-scan
date: 2026-11-09
---
2026-11-09 — The preflight secret scan (iss-2611081341084748) runs a gitleaks on PATH only when it reports exactly the version ci.yml pins; any other copy, or none, is replaced by that release fetched once into ~/.abcd.noindex/gate-tools/gitleaks/<version>/ and checked on every run against the sha256 held in scripts/gitleaks-checksums.txt. A machine that cannot fetch or verify it now refuses the scan (exit 2) instead of skipping it, as scripts/pinned-toolchain.sh refuses for Go, because a skipped or skewed scan still minted a push receipt CI then contradicted. CI's gitleaks job checks its download against the same file rather than a checksum list fetched beside it, so the version stays in ci.yml's GITLEAKS_VERSION and the digests in one repository file, with TestGitleaksDigestsCoverCIsPin failing when the two disagree. The cloud connector setup still checks against the fetched list; it installs CI's version, so it causes no skew, and is left for its own change.
