---
schema_version: 1
id: "iss-358"
slug: "scanner-lanhostre-and-devicehostre-keep"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "bughunt-round-2"
found_at: "internal/adapter/scanner/network.go"
wontfix_reason: "the fix was built, adversarially reviewed pre-merge, and reverted: dropping the trailing boundary flags snake_case selectors (stream.local_addr, args.local_rank, storage.local_key) and Stage-1 redaction rewrites every finding into the irreversible history store, so the warn-tier miss on an underscore-suffixed hostname is the accepted cost; TestSnakeCaseSelectorsStayQuiet pins the decision"
---

scanner lanHostRe and deviceHostRe keep the trailing ASCII word boundary the iss-307 sweep dropped from ipv4Re and macRe, so a LAN or device hostname abutting an underscore or alnum suffix is silently missed
## Evidence

- `internal/adapter/scanner/network.go:100` (`lanHostRe`) and `:110` (`deviceHostRe`) end in `\b`; the alternation requires a literal suffix (`local|lan`, `-macbook` etc.), so RE2 cannot shorten to a boundary and a `_`/alnum-adjacent hostname drops the whole match. Verified through `ScanText` -> `Redact`: `my_printer.local` and `bob-macbook.local_backup` (LAN half) produce 0 findings; bare forms produce 1.
- Prior art boundary: iss-307 (hunt A) drops the trailing `\b` from `ipv4Re`/`macRe` only; the two hostname patterns are the residual of that sweep.
- Refuter verdict: CONFIRMED substantive for the class. Fix verified against a whole-tree sweep: findings byte-identical before/after apart from the new catches, `go test ./...` green. **That verification was insufficient** — the sweep ran over a tree containing no `.local_`/`.lan_` snake_case selector, so it could not see the false-positive class the pre-merge review then proved (`stream.local_addr` and kin rewritten by irreversible Stage-1 redaction); a clean sweep over one tree is evidence about that tree, not about the input space.
- Re-minted from the round-2 id that collided with a parallel hunt's mint (the collision class this ledger's own iss-344 records).
