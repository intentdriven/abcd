---
schema_version: 1
id: "iss-2609090715154147"
slug: "an-adr-is-never-amended-always-superseded-is-unrecorded"
severity: "minor"
category: "documentation"
source: "user-observation"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
deferred_after: "v0.9.0"
deferral_reason: "Routed to the product thinker by the 2026-09-23 run (ruling owed: adopt 'an ADR is never amended, always superseded' as a stated convention; no record decides it and iss-2608220750024303 asks for an amendment). The 2026-09-23 interview gave routed minor and nitpick captures the default: deferred past v0.9.0, returning at the next anchor."
---

The convention is that a decision record is never amended in place: when a decision changes, a new record is minted and the old one is marked superseded by it, so the decision history stays legible and a reader of the old record is pointed forward rather than misled. Nothing in the tree states it. The typed-link vocabulary is recorded in .abcd/development/principles/decompose-before-filing.md as supersedes / reverses / duplicates / refines, and adr-58 and adr-2609021016286571 demonstrate the shape in practice, but the rule that amendment is not an option is carried only by example, so an agent or a contributor reaching for the smaller edit has nothing to read that refuses it. The gap surfaced when a relocation of the transcript corpus made adr-29's stated location wrong and the question of amend-versus-supersede had to be asked rather than looked up. Fix: state the rule where the conventions live, in the AGENTS.md decisions section or as a principle under .abcd/development/principles/, naming the two typed links and the one edit a superseded record does receive, and say why: an amended record erases the fact that the decision changed, which is the thing the record family exists to preserve. Detector: the conventions text names the rule and a docs test or record-lint rule refuses a decisions section that omits it, or at minimum the rule is present where a contributor reading AGENTS.md would meet it before editing a record.
