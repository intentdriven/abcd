---
schema_version: 1
id: "iss-2610262031161961"
slug: "05-personas-says-the-canonical-create"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 108 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/05-personas.md:7"
remedy: "none (filed automatically)"
finding_ref: cfk-a9d982cd469320ad
---

05-personas says the canonical create picks a persona for each press release; the verification matrix stages persona selection and says nothing selects.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/05-personas.md:7` — "picks a persona for the customer quote in each press release"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:59` — "picking a persona from the roster and biasing that pick on a role hint: nothing selects"

05-personas says the canonical create 'picks a persona for the customer quote in each press release', selection being by role. The matrix's Persona registry row marks as staged 'picking a persona from the roster and biasing that pick on a role hint: nothing selects', the shipped part being only the lint that refuses an unregistered name. Either the create picks a persona or nothing does.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 108.
