---
schema_version: 1
id: "iss-2608221456469938"
slug: "the-hard-fail-local-username-redaction"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "all-dimensions bug-hunt round 7"
found_at: "internal/adapter/scanner/identity.go"
resolution: "m.localBare folds case; system-dir exemption lower-cased to keep iss-31 closed; encodedMatches folds with EqualFold."
impact: fix
resolved_by:
  commit: "35b23f1"
---

The hard_fail local_username redaction matcher (m.localBare) is built without the (?i) its home-path sibling carries, so on a case-folding filesystem a case variant of the caller login (ALEX/Alex) is neither redacted nor caught by the blocking residual scan, leaking the login into the stored transcript.