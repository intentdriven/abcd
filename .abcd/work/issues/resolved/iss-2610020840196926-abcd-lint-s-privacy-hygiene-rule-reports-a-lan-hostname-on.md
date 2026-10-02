---
schema_version: 1
id: "iss-2610020840196926"
slug: "abcd-lint-s-privacy-hygiene-rule-reports-a-lan-hostname-on"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/network.go"
remedy: "In selectorExpression (internal/adapter/scanner/network.go), also treat a match as a selector when ')' or ',' follows it and the file is Go source (or the left operand is an identifier the line uses as a Go value), with a test pinning such a selector followed by ')' and by ',' as skipped, and a hostname inside prose parentheses still reported; shown wrong if a real LAN hostname in a .go comment followed by ')' is then missed, in which case limit the skip to code outside comments."
resolution: "callArgumentSelector (internal/adapter/scanner/network.go) exempts a lower-case Go selector chain passed as a call argument and followed by ')' or ',', when the innermost unclosed '(' opens a call and no comment marker precedes it; abcd lint conforms on the tree"
impact: fix
resolved_by:
  commit: "d303e2960"
---

abcd lint's privacy-hygiene rule reports a LAN hostname on internal/core/capture/eligible.go:137, where the match is a Go selector, the field `local` of the value `anchor` joined by a dot, which closes a call's argument list and no hostname at all. The net_lan_hostname pattern's selectorExpression skip (internal/adapter/scanner/network.go) treats a match as a selector only when '(' or '[' follows it, or '=' or '{' after spaces, so a selector passed as a call's last or middle argument, followed by ')' or ',', is reported. abcd lint exits 1 on the warning in a clean tree at 7fb52a6b5.

## Grounds

- pursued: abcd lint reports no LAN hostname on internal/core/capture/eligible.go and TestCallArgumentSelectorIsNotAHost passes, while TestHostClosingAParenthesisStillFlags keeps a host in prose parentheses, a list, a URL, a config value, a quoted string and a comment reported; shown wrong if a real hyphen-free lower-case host written as a call argument outside a comment is then missed, the named residual
