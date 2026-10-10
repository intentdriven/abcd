---
schema_version: 1
id: "iss-2609261658553101"
slug: "the-repolint-privacy-hygiene-rule-and"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/repolint/rule_privacy.go"
resolution: "Both rules now read each committed line as written and through scanner.DecodedViews, the exported seam onto the scanner's one list of decoded views (the percent pre-pass and every JSON-escape layer), so a home after a newline escape, the solidus and unicode escapes, doubled Windows separators, a percent-encoded home, a session URL straight after a newline escape and a footer on its own line inside a quoted string are all refused as their plain spellings are. The footer's line-start SkipAt reads a break inside the text as a line start, which reaches the scanner's own views too. The Go-literal cost was measured on the live tree with abcd lint: 101 findings before, 103 after; twelve deliberately illustrative test-fixture lines that read newly carry the rule's waiver, no production line reads newly, and the remaining three are resolved records describing the escaped spellings with placeholder names, the same class as the record findings already on the baseline. docs-lint and record-lint report nothing new. Tests: TestAC_PrivacyReadsEscapedSpellings, TestAC_PrivacyEscapedSpellingsKeepTheExemptions, TestHarnessLeakReadsEscapedSpellings, TestHarnessLeakEscapedSpellingsSpareProse, TestFooterAfterADecodedLineBreakOwnsItsLine."
impact: fix
resolved_by:
  commit: "facc64db"
---

The repolint privacy-hygiene rule and the harness_leak lint rule read each committed line raw, so the JSON-escape spellings the scanner reads through its decoded views (jsonescape.go) pass both: a third-party home after a \n escape, a home written with the solidus escape, a Windows home with doubled separators, and a token or a session URL written straight after a \n escape (the patterns anchor on a leading word boundary, and the escape letter is a word byte). A committed JSON fixture, export or transcript carrying any of them is not refused. Reading the views in the lint is not contained: Go interpreted string literals use the same escapes, so the views would surface every test string that puts a \n escape before a /home root and a name in committed Go source, and the waivers on those lines do not all exist. The CI gitleaks history scan covers the token half; the home-path and session-URL halves have no other gate. Detector: privacyLeak and harnessLeakOnLine report each of the four spellings on a committed line.

## Grounds

- pursued: a committed line whose escapes hide a home path, an address or a harness-leak shape is refused by both rules at no production-code false-positive cost; a production source line flagged only through a decoded view, or an escaped leak either rule still passes, would show it wrong
