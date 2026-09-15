---
schema_version: 1
id: "iss-2608311230204063"
slug: "the-ingest-verb-hands-item-body-text-to-capture-ingestreadin"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "manual-capture"
origin: researcher-authored
production_mode: hand-written
resolution: "Item text passes through termsafe.EncodeHiddenRunes on the way to the record writer, after the checks so the encoding cannot defeat them."
impact: fix
resolved_by:
  intent: "itd-185"
  spec: "spc-63"
---

The ingest verb hands item body text to capture.IngestReading after redaction only, and capture's yamlScalar refuses runes below 0x20 and nothing above, so a model-authored bidi override, C1 control or zero-width rune lands verbatim in a committed markdown reading record. This diff is what makes that writer reachable, and it makes it reachable exclusively from model output, so a Trojan-Source-style reversed span can reach a record a reviewer reads in a terminal. internal/termsafe carries a lossless encoder for exactly this boundary.

## Grounds

- pursued: the finding is closed by a test that fails without the change; a later review or mutation run finding the same shape again would show this wrong
