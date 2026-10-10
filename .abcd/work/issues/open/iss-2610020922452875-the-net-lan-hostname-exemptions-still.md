---
schema_version: 1
id: "iss-2610020922452875"
slug: "the-net-lan-hostname-exemptions-still"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/network.go"
remedy: "Waits on a ruling on how far the net_lan_hostname trust path may widen: each position is a separate exemption with its own prose collision (':' ends a YAML key and a label, ']' closes a markdown link text, end of line ends a sentence), so the fix adds the positions one at a time to the Go-only exemption (scanner.GoSourceSkip, which only the privacy lint rule consults, for a tracked .go file, and never a path-less redactor) behind the same guards callArgumentSelector uses (a lower-case Go selector chain, no Go comment marker before it, a bounded prefix) and a still-flagged prose and config twin for each; shown wrong if any added position exempts a host in the twin"
---

The net_lan_hostname exemptions still report a lower-case Go selector whose LAN-suffix field sits in any code position other than a call target or index (selectorExpression), an assignment target or block head, or a call argument closed by ')' or ',' (callArgumentSelector): a case label closed by ':', an index closed by ']', the right-hand side of ':=' or '=' at end of line, a 'return' operand, an operand of '&&' or '||', a composite-literal value or element, and a unary '!' operand are each reported as a LAN hostname (probed at 6d7ae00d2 on fix/scanner-selector-closing-call), so abcd lint warns and Stage-1 redaction rewrites such a line in a stored transcript. No tracked file carries one today; the class is latent.
