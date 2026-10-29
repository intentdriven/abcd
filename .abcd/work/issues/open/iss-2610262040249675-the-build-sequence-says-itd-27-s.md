---
schema_version: 1
id: "iss-2610262040249675"
slug: "the-build-sequence-says-itd-27-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 929 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:100"
remedy: "Retitle the 'Via /abcd:intent grill' section: itd-27 is superseded by itd-94 and no grill sub-verb ships; keep the glossary-aware write-back only as a note for the planning interview (abcd intent interview) if it is still wanted."
finding_ref: cfk-6a83794c767d978b
---

The build sequence says itd-27's grilling is now the planning interview; the glossary README keeps /abcd:intent grill as a live itd-27 design target.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:100` — "the grilling itd-27 asked for is the planning interview"
- End B: `.abcd/development/brief/glossary/README.md:174` — "`/abcd:intent grill` is a **design target** (itd-27)"

The build sequence says there is no grill sub-verb because the grilling itd-27 asked for is now the planning interview, under itd-94, which supersedes it. The glossary README calls /abcd:intent grill a design target of itd-27 and specifies its glossary-aware write-back. A design target cannot rest on an intent that another intent has superseded with a different surface.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 929.
