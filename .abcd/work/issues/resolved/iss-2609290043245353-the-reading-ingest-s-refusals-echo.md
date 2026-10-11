---
schema_version: 1
id: "iss-2609290043245353"
slug: "the-reading-ingest-s-refusals-echo"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/reading/ingest.go"
resolution: "Fixed by c79673c94: the reading ingest's envelope refusals (_type, run_id, position, manifest_sha256) and WriteRunArtefact's run id describe a refused value with termsafe.DescribeRefused instead of echoing it, and the decoder's undeclared-field message goes through the canonical scanner (redactRefused), failing closed to a description. Every other echo site was classified and kept on the ground that abcd wrote or validated the value: the hash-proven manifest's fields, a manifest_sha256 already matched to 64 hex, a parsed position, the constant artefact name, and item names already through the payload redactor."
impact: fix
resolved_by:
  commit: "c79673c94"
---

The reading ingest's refusals echo payload-chosen values unredacted. echo() in internal/core/reading/ingest.go cleans a value (termsafe.CleanProseLine) and caps it at maxEchoedBytes (120), but runs no privacy redaction, and 17 call sites use it: the output's _type, run_id, manifest_sha256 and position among them, beside values the manifest (abcd-written) supplies. A reading output whose closed-set or id field carries a token or a home path is refused with up to 120 bytes of it in the error, which reaches the terminal and the transcript. Found on the sweep for iss-2609290033521472, which fixed the same shape in lifeboat review and intent audit with termsafe.DescribeRefused and the audit redactor. Fix direction: per site, describe a payload value (DescribeRefused) or route it through the reading's own payloadField redactor (redact.go), leaving manifest-sourced values as they are. Detector: a reading output carrying a sentinel in each quoted field is refused without the sentinel in the error.

## Grounds

- pursued: a reading output carrying a marker and a home path in any envelope field or in an undeclared key is refused without either in the error; a refusal that still carries the marker would show it wrong
