---
schema_version: 1
id: "iss-2609290743362554"
slug: "the-scanner-s-glued-token-sweep-glued-go"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/percent.go"
resolution: "Both halves fixed on fix/drain-echo-4: aa5ea012d runs the glued sweep over every percent-decoded and JSON-unescaped view through the same viewTokenFindings path as the bounded patterns, so an escaped glued token is found at its raw bytes and sealed by Redact and RedactRefusal; 0d3156d28 folds a sweep New cannot build whole into Unavailable with a reason naming the pattern, so every write-time redactor, the launch scan and RedactRefusal fail closed on it."
impact: fix
resolved_by:
  commit: "0d3156d28"
---

The scanner's glued-token sweep (glued.go, iss-2609290541525428) runs on the raw line only: decodedLineFindings and viewFindings (percent.go) scan the percent-decoded and JSON-unescaped views with the bounded patterns alone, whose leading word boundary cannot hold behind a word byte. A glued token whose own bytes are escaped (a percent-encoded letter of its prefix, or a JSON unicode escape of its first byte, behind an underscore or a letter) therefore survives ScanText, Redact and RedactRefusal raw. Reach is narrow, since no encoder escapes an ASCII letter of a token, but the decoded layers exist to close spelling variants. A second half: ScanText gives no signal when the sweep cannot be built in full (a configured pattern whose leading boundary carries a quantifier); only RedactRefusal reads the sweep's completeness, so every other consumer runs a silently narrower sweep, and Unavailable does not say so.

## Grounds

- pursued: ScanText reports, and Redact and RedactRefusal seal, a glued token whose own bytes are percent- or JSON-escaped (TestScanTextFindsAnEscapedGluedToken, TestRedactRefusalSealsAnEscapedGluedToken, eight subtests RED before aa5ea012d), and a configured pattern with a quantified leading boundary makes Unavailable true with its name (TestUnavailableNamesAnIncompleteGluedSweep, RED before 0d3156d28); an escaped glued spelling that still comes back raw, a decoded-view charge growing past the 6.0x bar in TestGluedSweepWorkIsLinear, or a degraded sweep that leaves Unavailable false would show it wrong.
