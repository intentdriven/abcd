---
schema_version: 1
id: "iss-2609290218032954"
slug: "four-more-host-payload-ingests-echo-payload-chosen-values"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/scribe/ingest.go"
resolution: "Fixed by 0a7d57626. Every site the record named was classified and each payload-chosen one fixed: scribe's handles, states and digests are quoted only in their closed shape and described otherwise, its free text is described, and its undeclared keys and strict-decoder messages (output, parked context and manifest) are redacted; release describes an out-of-set section, a malformed record id and an attributed headline name, quotes a stale next_tag only in a tag's shape, and redacts its decoder message and persona finding; ideate and the four lifeboat decoders redact their messages. One redaction definition serves them all: scanner.RedactRefusal, which fails closed to a description on an unavailable or degraded scanner. The sweep also fixed the reading parked-manifest decoder message and captured the memory page ingest (iss-2609290300464268) and the implement lane receipt (iss-2609290300462829)."
impact: fix
resolved_by:
  commit: "0a7d57626"
---

Four more host-payload ingests echo payload-chosen values into their refusals, cleaned but never redacted: the class iss-2609290043245353 and iss-2609290144116254 fixed in reading and intent consistency. (1) scribe ingest (internal/core/scribe/ingest.go) passes 21 values through its own echo(), which sanitises and caps but does not redact: the output's _type and run (177, 180), an undeclared key (343), a disposition's item id, state and free text (481-516), among them. (2) release changelog ingest (internal/core/release/ingest.go) quotes next_tag (328), an out-of-set section (532) and a malformed record id (560) through termsafe.Sanitize alone, and the section is uncapped. (3) Five strict decoders return encoding/json's message raw, which names an undeclared field by the payload's own key: ideate record.go:248, lifeboat graveyard_lessons.go:83, synthesis_pressrelease.go:157, synthesis_principles.go:217, synthesis_review.go:275 (release ingest.go:478 sanitises the same message). A token or a home path in any of them reaches the terminal and the transcript. Fix direction: per site, describe a closed-shape value (termsafe.DescribeRefused), redact a name the reader needs through the canonical scanner, keep a value abcd wrote. Detector: each payload carrying a marker and a home path in each field is refused without either in the error.

A sixth decoder site joins the list from the review of lane drainEcho: scribe.go:198, decodeStrict, reached from ingest.go:328, returns encoding/json's undeclared-field message raw for the scribe output, the parked context and the parked manifest, and its duplicate-key refusal names the repeated key the same way.

## Grounds

- pursued: a scribe, release, ideate or lifeboat payload carrying a marker and a home path in any refused field or undeclared key is refused without either in the error, still naming the field; a refusal that still carries the marker or the path would show it wrong
