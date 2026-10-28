---
schema_version: 1
id: "iss-2610262038329713"
slug: "persona-means-a-placeholder-stakeholder"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 801 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/persona.md:4"
remedy: "none (filed automatically)"
finding_ref: cfk-a97494003fde2ed1
---

"persona" means a placeholder stakeholder from the personas registry, and also the name a session takes from a machine register.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/persona.md:4` — "A placeholder stakeholder character drawn from the abcd personas registry"
- End B: `.abcd/development/brief/04-surfaces/README.md:276` — "sets the session's persona name as its title (itd-2610111707085098)"

persona.md defines a persona as a placeholder stakeholder from the personas registry, used in press releases and design documents. 04-surfaces/README has the prompt router set the session's persona name as its title, and session-start reserve that name in the machine register. Agent sessions are not stakeholder archetypes, so one word names two concepts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 801.
