---
schema_version: 1
id: "iss-2610262041398357"
slug: "the-record-families-page-lists-the-step"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1016 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-2609212103565953]
finding_ref: cfk-34e8287b5bfe1b70
---

The record-families page lists the step intent itd-2609212103565953 as planned; that intent is filed under shipped/.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:24` — "no verb: each lands as one pull request (itd-2609212103565953, planned)"
- End B: `.abcd/development/intents/shipped/itd-2609212103565953-a-spec-lists-its-steps-and-the-build.md:21` — "A spec lists its steps, and `abcd build` lands them one at a time, each its own lane and pull request."

The step row of the record-families page cites the intent that introduced steps as '(itd-2609212103565953, planned)'. That intent's record sits in intents/shipped/, and its press release describes the delivered behaviour: the build lands a spec's steps one lane and pull request at a time. Under directory-as-truth the folder is the status, so the page that maps the record families gives a lifecycle state that the record contradicts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1016.
