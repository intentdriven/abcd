---
schema_version: 1
id: "iss-2610090821506490"
slug: "scanner-dot-skip-fragment-passes-addresses"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "private security advisory GHSA-8fqr-pv94-5jwp, filed 2026-10-05"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/scanner.go"
remedy: "Reject a skip fragment that is only punctuation or matches every ordinary source path, and count a byte-only file toward the zero-coverage sentinel unless its extension is on the reviewed binary list; prove it with a scanner test (watched fail first) that the `.` fragment is refused or the address hard-fails, a token on a dotted path still hard-fails, a blank or slash-only fragment is still dropped and the default log-directory fragments still skip only those directories; sweep siblings (the other skip lists mergeConfig accepts)."
resolution: "A file a skip fragment alone sends to byte-only scanning is Unscanned with its reason, so the launch refuses and names it, unless its extension or name is on the reviewed skip lists; a new exclude_path_fragments config field declares an exclusion with a required reason, reported as excluded by choice and never counted as scanned, and the zero-coverage sentinel still refuses a bundle it leaves unscanned."
impact: fix
---

A committed `.abcd/config/pii.json` with `skip_path_fragments: ["."]` sends every dotted path to the byte-only branch, so a non-reserved IPv4, IPv6 or MAC address in an included file ships through the launch scan.

Private security advisory GHSA-8fqr-pv94-5jwp (draft, severity medium). Full text, evidence and reproduction: the security-drain-2026-10-09 run directory in the main checkout's local tier. This record stays uncommitted until its fix lands; the fix commit adds it directly to resolved/.

Evidence (lines at main 7549ca2d5): `mergeConfig` drops a fragment only when trimming slashes and whitespace leaves it empty, so `.` is stored (internal/adapter/scanner/scanner.go:338). `skipByFragment` is `strings.Contains` (internal/adapter/scanner/scanner.go:1495). `secretPatterns` drops the identity kinds, which include the network kinds (internal/adapter/scanner/scanner.go:1474). The zero-coverage sentinel trips only when FilesScanned is zero, and an undotted `LICENSE` on the include list keeps it above zero. `scanRefusals` does not refuse ContentUnverified files (internal/core/launch/dryrun.go:269). A token, the caller's home path, a real email and a long real name on that path still block.

Reproduction: commit `{"skip_path_fragments":["."]}` as `.abcd/config/pii.json`, put a line of `dns ` followed by a public, non-reserved IPv4 address (the well-known public DNS resolver of four eights; capture redacted the literal) in `commands/a.md`, leave LICENSE clean. `ScanBundle` is available, FilesScanned is at least 1, findings are empty and the markdown file is ContentUnverified. `ScanText` of the same line is net:ipv4 at hard_fail.
