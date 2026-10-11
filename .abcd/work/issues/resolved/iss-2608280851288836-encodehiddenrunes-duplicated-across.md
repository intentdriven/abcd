---
schema_version: 1
id: "iss-2608280851288836"
slug: "encodehiddenrunes-duplicated-across"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "adversarial review of the output-sanitisation batch (2026-08-28)"
found_at: "internal/core/memory/ingest.go"
resolution: "the hidden-rune encoder is one exported termsafe.EncodeHiddenRunes; memory and cite route through it, no duplicate copies"
impact: internal
resolved_by:
  commit: "97c6d6d7"
---

encodeHiddenRunes is duplicated byte-for-byte in memory/ingest.go and cite/fetch.go: both packages carry an identical unexported percent-encoder for hidden runes (Trojan-source/JSON-escape class), added independently to close the same redirect-URL leak. termsafe is the canonical home for untrusted-boundary sanitisers; two private copies will drift, and a rune class added to one leaves the other door open. Export one termsafe.EncodeHiddenRunes and route both call sites through it. Follow-up to the batch that fixed iss-357.