---
schema_version: 1
id: "iss-2608292034215745"
slug: "merged-anchor-and-byte-policy-let-a-home-path-ship-in-a-blob"
severity: "major"
category: "security"
source: "impl-review"
found_during: "v0.6.9-security-pass"
found_at: "internal/adapter/scanner/scanner.go"
resolution: "scanBytes scans with both halves of the home anchor waived (identityMatchers.bytes) — a raw blob carries no path syntax on either side of the literal — so the operator's home is reported in a raw blob wherever the literal sits, and bytes are judged at least as strictly as text; a DryRun test drives identical bytes as .md and .png with the home preceded by an alphanumeric byte, for /root and a two-segment home, and asserts both hard-fail"
impact: fix
---

v0.6.9 combined-diff security review: a regression created by the MERGE of two branches, present on neither alone. fix/scanner's 95b98abc made the byte scan's rule table explicit: byteScanPolicy(local_username) is bytePolicyDrop, safe there because home_path_self was unanchored and always caught the operator's home in raw bytes. fix/ultra-findings' 27081ac1 anchored home_path_self so a home preceded by a path-segment byte is declined as a longer name. On the merged tree (3e1acc36 onto 44329c40) a raw blob — a PNG or PDF, where the home is typically preceded by an alnum byte with no separator — has home_path_self DECLINE the home at the leading anchor, local_username then dropped by policy, and launch ship publishes the operator's absolute home path. Demonstrated end-to-end: identical bytes as .md hard-fail (1 finding) and as .png pass (0 findings, no kinds); a mid-blob home in a .png likewise. Fix: on bytes, the leading anchor cannot hold (a raw blob has no path syntax to anchor on, and the long literal makes chance collision negligible), so scanBytes must keep home_path_self wherever the literal occurs, or stop dropping local_username inside a /Users|/home segment.
