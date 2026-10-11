---
schema_version: 1
id: "iss-2608270720336165"
slug: "the-percent-decode-pre-pass-added-for"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security-cut-consolidated-review-2026-08-27"
found_at: "internal/adapter/scanner/percent.go"
resolution: "the identity matchers and $HOME backstop now run over the percent-decoded copy too, so a percent-encoded home path or email is redacted before it can enter a committed record"
impact: fix
resolved_by:
  commit: "e383293e"
---

The percent-decode pre-pass added for #370 (internal/adapter/scanner/percent.go / scanner.go ScanText) re-runs only the token-pattern matchers on the decoded copy, not the IDENTITY matchers (matchers.findings runs on the raw line alone), so a percent-encoded home path or email (e.g. %2Fhome%2Falice%2F.ssh, %61lice%40host) can survive redaction into a committed artifact (memory --keep-original, intent body, capture ledger). The literal $HOME backstop also sees only the encoded bytes. This is a scope boundary of the #370 fix as designed (its comment scopes to the leading-\b token patterns), not an integration regression — surfaced by the consolidated adversarial review of the security cut. Fix: run the identity matchers (and the $HOME backstop) over the decoded copy too, mapping hits back to raw spans as the token path already does.