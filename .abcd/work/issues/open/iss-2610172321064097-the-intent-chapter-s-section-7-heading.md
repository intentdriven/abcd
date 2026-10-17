---
schema_version: 1
id: "iss-2610172321064097"
slug: "the-intent-chapter-s-section-7-heading"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "brief-drift fixes on fix/deferred-brief-followups (peer report)"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md"
remedy: "Add one clause to the section 7 opening saying the first two roles ship and the third is a design target, agreeing with 05-internals/01-agents.md."
---

The intent chapter's section 7 heading and opening paragraph (.abcd/development/brief/04-surfaces/05-intent.md, 'The intent-auditor agent (three roles, three verbs)') say the auditor owns three roles, each with its own verb, without saying which ship. Two do: the audit (role 1) and the consistency pass (role 2) are registered sub-verbs of the intent verb, while the shape role (role 3) has no registered verb and its own subsection calls its surface a later phase. A reader of the opening alone takes all three verbs as shipped.
