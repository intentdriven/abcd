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
deferred_after: "v0.11.1"
deferral_reason: "lane drainEcho (run A) fixed the reading and consistency ingests and stopped here: about thirty sites across four packages (scribe, release, lifeboat, ideate) each need a call on origin, and the lane brief keeps its hunks local while integration branches land; no product ruling is owed"
---

Four more host-payload ingests echo payload-chosen values into their refusals, cleaned but never redacted: the class iss-2609290043245353 and iss-2609290144116254 fixed in reading and intent consistency. (1) scribe ingest (internal/core/scribe/ingest.go) passes 21 values through its own echo(), which sanitises and caps but does not redact: the output's _type and run (177, 180), an undeclared key (343), a disposition's item id, state and free text (481-516), among them. (2) release changelog ingest (internal/core/release/ingest.go) quotes next_tag (328), an out-of-set section (532) and a malformed record id (560) through termsafe.Sanitize alone, and the section is uncapped. (3) Five strict decoders return encoding/json's message raw, which names an undeclared field by the payload's own key: ideate record.go:248, lifeboat graveyard_lessons.go:83, synthesis_pressrelease.go:157, synthesis_principles.go:217, synthesis_review.go:275 (release ingest.go:478 sanitises the same message). A token or a home path in any of them reaches the terminal and the transcript. Fix direction: per site, describe a closed-shape value (termsafe.DescribeRefused), redact a name the reader needs through the canonical scanner, keep a value abcd wrote. Detector: each payload carrying a marker and a home path in each field is refused without either in the error.
