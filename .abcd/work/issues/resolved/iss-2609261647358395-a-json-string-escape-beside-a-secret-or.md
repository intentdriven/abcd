---
schema_version: 1
id: "iss-2609261647358395"
slug: "a-json-string-escape-beside-a-secret-or"
severity: "major"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/percent.go"
resolution: "Fixed by c55ae5ef: the scanner reads each line's JSON-escape layers (at most three, every layer scanned) as decoded views mapped back to raw spans, the percent pre-pass's shape, so a token after a \\n or \\t escape, a home path beside \\n, \\t or \\\" escapes and a \\u-escaped real name are findings and are masked on disk. Pinned by TestJSONEscapeBesideASecretOrHomeIsStillAFinding and, at the store boundary, TestCaptureRedactsValuesBesideJSONEscapes, both watched RED at 211b8853."
impact: fix
resolved_by:
  commit: "c55ae5ef"
---

A JSON string escape beside a secret or a home path hides it from the scanner's raw-line pass, and the transcript store scans raw JSONL. Every bundled token pattern anchors on a leading word boundary, so a token written straight after the \n, \t, \r, \b or \f escape (a GitHub PAT at the start of a line of pasted output, JSON-encoded) has the escape letter as a word byte before it and never matches: ScanText at BASE 211b8853 returns nothing for {"t":"tok\nghp_..."} while the same token after a space is token:github_pat. The identity matchers fail the same way: home_path_other finds no third-party home path that follows a \n or \t escape (the leading anchor reads the escape letter as a path byte) or that precedes a \n or \" escape (the trailing boundary set has no backslash), and a non-ASCII real_name written by an ASCII-only encoder (Python json's default \u00e9) never matches the configured name. The fix reads a line's JSON-decoded layers the way the percent-decode pre-pass reads its percent-decoded copy, mapping each hit back to its raw span. Detector: a token after a \n escape, a home path between \n and \" escapes and a \u-escaped real_name are each a finding on the transcript path.

## Grounds

- pursued: every ScanText consumer (history, capture, memory, lint outbound, launch, lifeboat) masks a value an escape sits beside; a raw JSONL transcript line with a prefixed token or third-party home next to an escape reaching the store verbatim would show it wrong
