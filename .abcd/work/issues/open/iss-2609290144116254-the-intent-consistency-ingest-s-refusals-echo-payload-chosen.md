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
---

The intent consistency ingest's refusals echo payload-chosen values unredacted. validateConsistency in internal/core/intent/consistency.go quotes the findings payload's _type with a bare %q (line 804, the same shape iss-2609290033521472 fixed in the audit ingest), and quotes a hash value (873), a finding's class and severity (938, 941), an end's path (993) and quote (1000) through oneLine, which cleans and caps but runs no privacy redaction. Each refusal returns to the surface, so a token or home path pasted into one of those fields reaches the terminal and the transcript. Found on the sweep for the audit.go _type fix (fix2-drainSec). Fix direction: describe a closed-set value (termsafe.DescribeRefused) and route path and quote through the verdict-prose redactor, deciding per site whether the echoed value is needed to find the fault. Detector: a findings payload carrying a sentinel in each quoted field is refused without the sentinel in the error.
