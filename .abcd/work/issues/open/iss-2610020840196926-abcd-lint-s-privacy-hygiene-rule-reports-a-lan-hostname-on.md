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
---

abcd lint's privacy-hygiene rule reports a LAN hostname on internal/core/capture/eligible.go:137, where the match is a Go selector, the field `local` of the value `anchor` joined by a dot, which closes a call's argument list and no hostname at all. The net_lan_hostname pattern's selectorExpression skip (internal/adapter/scanner/network.go) treats a match as a selector only when '(' or '[' follows it, or '=' or '{' after spaces, so a selector passed as a call's last or middle argument, followed by ')' or ',', is reported. abcd lint exits 1 on the warning in a clean tree at 7fb52a6b5.
