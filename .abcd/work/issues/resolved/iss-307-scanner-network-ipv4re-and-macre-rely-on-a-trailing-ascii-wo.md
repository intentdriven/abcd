---
schema_version: 1
id: "iss-307"
slug: "scanner-network-ipv4re-and-macre-rely-on-a-trailing-ascii-wo"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/adapter/scanner/network.go"
resolution: "ipv4Re/macRe drop the trailing \\b with a compensating truncated-number guard; a word-suffixed hard_fail address is caught. Zero added FPs measured."
impact: fix
---

scanner network ipv4Re and macRe rely on a trailing ASCII word boundary, silently missing a hard_fail address whose only sin is an adjacent word char
## Evidence
`internal/adapter/scanner/network.go:75` (`ipv4Re`, hard_fail) and `:91` (`macRe`, hard_fail) both end in `\b`. `macRe` is fixed-length pure-word-char + trailing `\b` — the exact dead-corner class burst-4/5 removed from `patterns.go` — so `a4:83:e7:11:22:33_eth0` and `10.1.2.3_gw` / `"192.168.1.44_gw": 1` are dropped whole. Reproduced through `ScanText`; the adjacency machinery does not rescue it (`10.1.2.3_10.1.2.4` → 0 findings). `history.go:139-145` acknowledges a trailing-boundary-dropped span slips both redaction stages; its literal backstop covers `$HOME` only. <!-- abcd-lint:allow -->

## Adversarial verdict: CONFIRMED for ipv4Re + macRe (minor); lanHostRe/deviceHostRe REFUTED
Dropping the trailing `\b` on the two host-shape warn patterns adds hundreds of FPs (`types.Locals`, `www.lan` from `www.langchain.com`) — their trailing `\b` is precision-load-bearing. For ipv4/mac the fix is measured zero-FP: drop trailing `\b`; `macRe` needs no compensating guard (`insideLongerColonRun` already suppresses on a hex-digit neighbour); `ipv4Re` needs an `isASCIIDigit(line[end])` guard added to `insideLongerDottedRun` to keep `1.2.3.1234`/`[::ffff:192.168.0.1000]` silent. Repo tree 74→74, GOROOT src 1201→1201 added FPs = 0. Not prior art: the burst-5 "no token pattern relies on trailing \b" invariant was scoped to patterns.go and predates the network set (iss-157). <!-- abcd-lint:allow -->
