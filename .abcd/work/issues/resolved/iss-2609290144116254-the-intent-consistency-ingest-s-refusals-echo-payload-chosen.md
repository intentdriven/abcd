---
schema_version: 1
id: "iss-2609290144116254"
slug: "the-intent-consistency-ingest-s-refusals-echo-payload-chosen"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/intent/consistency.go"
resolution: "Fixed by c79673c94: the consistency ingest describes a refused _type, policy hash, finding class and severity, and an end's path and quote with termsafe.DescribeRefused instead of quoting them through a bare %q or oneLine, and the decoder's undeclared-field message goes through the canonical scanner (redactRefused), failing closed to a description. The finding and end numbers still locate the fault. The request's review_of_commit stays quoted because it comes from the request abcd wrote and is matched to hex."
impact: fix
resolved_by:
  commit: "c79673c94"
---

The intent consistency ingest's refusals echo payload-chosen values unredacted. validateConsistency in internal/core/intent/consistency.go quotes the findings payload's _type with a bare %q (line 804, the same shape iss-2609290033521472 fixed in the audit ingest), and quotes a hash value (873), a finding's class and severity (938, 941), an end's path (993) and quote (1000) through oneLine, which cleans and caps but runs no privacy redaction. Each refusal returns to the surface, so a token or home path pasted into one of those fields reaches the terminal and the transcript. Found on the sweep for the audit.go _type fix (fix2-drainSec). Fix direction: describe a closed-set value (termsafe.DescribeRefused) and route path and quote through the verdict-prose redactor, deciding per site whether the echoed value is needed to find the fault. Detector: a findings payload carrying a sentinel in each quoted field is refused without the sentinel in the error.

## Grounds

- pursued: a findings payload carrying a marker and a home path in any of those fields or in an undeclared key is refused without either in the error; a refusal that still carries the marker would show it wrong
