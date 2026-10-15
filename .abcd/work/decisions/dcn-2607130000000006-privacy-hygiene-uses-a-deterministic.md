---
id: dcn-2607130000000006
slug: privacy-hygiene-uses-a-deterministic
date: 2026-07-13
---
2026-07-13 (itd-85 M3): privacy-hygiene uses a deterministic, identity-INDEPENDENT
absolute-path regex, NOT the identity-aware scanner (internal/adapter/scanner).
Rejected the scanner because its home-path detection is identity-PARAMETERISED
(kindHomeSelf=hardfail vs kindHomeOther=warn) — machine-dependent severity —
whereas AC3's contract is "ANY absolute local path is an error", deterministic
across machines. The scanner also scans the release BUNDLE (a curated allowlist
excluding tests); audit scans all tracked files, a scope the scanner was not
built for. Flagged for future consolidation: absolute-home-path detection now
lives in two predicates (scanner identity matchers + audit regex); a later phase
should extract a shared identity-independent path matcher.
