---
schema_version: 1
id: "iss-305"
slug: "abcd-lint-privacy-hygiene-hard-fails-on-ordinary-committed-c"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "internal/core/repolint/rule_privacy.go"
resolution: "hasAbsHomePath gains the leading-boundary predicate; abcd lint no longer hard-fails on relative/URL home segments. Fixed in this round's scanner/privacy commit."
impact: fix
---

abcd lint privacy-hygiene hard-fails on ordinary committed content because hasAbsHomePath lacks the leading-boundary predicate its scanner twin has
## Evidence

`internal/core/repolint/rule_privacy.go:178` — `hasAbsHomePath` runs
`absPathRe.FindAllStringIndex(line, -1)` with **no leading-boundary predicate**,
unlike its scanner twin `genericHomeRe` which is gated by `leadingBoundaryOK`
(`internal/adapter/scanner/identity.go:238`). Any relative path segment
`home/<x>` or `Users/<x>` — `src/pages/home/index.tsx`, `const Route =
"/home/dashboard"`, a docs URL `https://docs.example.com/home/getting-started` — <!-- abcd-lint:allow -->
matches and returns `SeverityError`, exit 2, on the shipped `abcd lint`
conformance gate.

`rule_privacy.go:32-35` asserts this rule and the scanner "can never disagree
about what a leak is"; measured, they disagree on both detection and severity
for this class. This repo's own `main` already trips a fifth privacy-hygiene
error beyond the four iss-274 documents:
`.abcd/development/research/notes/2026-08-19-issue-ledger-forge-sync-sota.md:85`
(a plain `https://fossil-scm.org/home/doc/...` citation URL).

## Adversarial verdict: CONFIRMED (major)

A shipped conformance gate hard-fails on ordinary committed content. Fix:
mirror the scanner's leading-boundary gate using repolint's own
`isPathSegmentChar` (which excludes `/`, keeping `file:///home/...` flagged). <!-- abcd-lint:allow -->
Measured: FP corpus 6→2, the fossil-scm URL FP cleared, iss-274's four true
findings untouched, 12/12 genuine leak shapes still caught, package tests pass.

Not prior art: iss-274 (abcd lint red on main) describes different lines and
calls them illustrative; iss-146/iss-236/iss-153/154/156 are unrelated FP
classes. The code comment at rule_privacy.go:51-57 deliberates only the
*trailing* separator — the leading boundary was never considered.
