---
schema_version: 1
id: "iss-2610111600051340"
slug: "one-letter-exclude-fragment-drops-secret"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-4f3r-hxcm-crq4, filed 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
remedy: "Match exclude_path_fragments on whole path segments (a fragment equals one or more consecutive segments, or is a directory prefix when it ends in /) and refuse a single-character fragment as mergeConfig refuses a punctuation-only one; prove the one-letter configuration fails the scan or leaves the secret file scanned and hard-failing, while testdata/vectors/ and generated/ still exclude only their directory and the excluded-by-choice count is unchanged"
resolution: "exclude_path_fragments now matches whole path segments instead of substrings: a fragment's segments must appear consecutively in the path, a fragment ending in / names a directory, and a dot-led fragment with no slash also matches the end of the file name; a single-character fragment is a config fault that makes the scanner unavailable. Proved by TestSingleCharacterExclusionIsAConfigFault and TestExclusionMatchesWholePathSegments (internal/adapter/scanner), with the existing exclusion tests unchanged; commands/launch.md states the new matching."
impact: fix
---

A one-letter `exclude_path_fragments` entry (for example `e`) passes `mergeConfig`, matches by substring nearly every path, and drops a file holding a secret from the release scan while one other scanned file keeps the zero-coverage sentinel quiet, so the secret is copied into the release payload and the ship gate does not refuse.

Private security advisory GHSA-4f3r-hxcm-crq4 (draft, severity medium). Follows GHSA-8fqr-pv94-5jwp, which refused a punctuation-only fragment. Full text, evidence and reproduction: .abcd/.work.local/scratch/security-drain-2026-10-11/advisories/ in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (main 3d171421a): `mergeConfig` (internal/adapter/scanner/scanner.go) refuses a blank, slash-only or punctuation-only fragment and accepts any fragment holding a letter or digit; `excludedBy` matches with `strings.Contains` on the slash path. Confirmed by reading the code at that commit; not re-run.
