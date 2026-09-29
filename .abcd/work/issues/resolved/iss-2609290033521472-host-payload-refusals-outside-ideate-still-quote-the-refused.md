---
schema_version: 1
id: "iss-2609290033521472"
slug: "host-payload-refusals-outside-ideate-still-quote-the-refused"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lifeboat/synthesis_review.go"
resolution: "Fixed: lifeboat review describes a refused mode, prompt_version and verdict through termsafe.DescribeRefused, and intent audit's dead-letter result returns its reason through the same redactor as the record's copy. The reading ingest's unredacted echo was split out as iss-2609290043245353 and deferred with its reason. The pre-receipt _type refusal, which quoted the value outside the dead-letter path, describes it through termsafe.DescribeRefused since commit 4ad272505."
impact: fix
resolved_by:
  commit: "950b69ca2"
---

Host-payload refusals outside ideate still quote the refused value into the error text. Found on the sweep for iss-2609090951295881, which fixed the six ideate sites: lifeboat synthesis_review.go quotes an out-of-set mode, prompt_version and verdict with a bare %q (no redaction, no sanitising beyond Go quoting); intent audit.go quotes an out-of-enum criterion verdict, a malformed criterion id, a _type and a receipt id the same way (its dead-letter RECORD is redacted, the returned error is not). Each field arrives in a host-composed payload beside free text the verb does redact, so a token or home path pasted into a closed-set field is printed verbatim to the terminal and transcript on refusal. The reading ingest routes its quoted values through echo(); whether that redacts or only bounds is unverified. Fix direction: the ideate describeRefused shape (length, never the value), shared rather than copied. Detector: for each payload verb, a closed-set field carrying a sentinel must be refused without the sentinel in the error.

## Grounds

- pursued: neither verb returns a refused payload value unredacted; a sentinel in the lifeboat review's closed-set fields, or a planted path in an audit verdict token, that reaches the returned error or reason would show it wrong
