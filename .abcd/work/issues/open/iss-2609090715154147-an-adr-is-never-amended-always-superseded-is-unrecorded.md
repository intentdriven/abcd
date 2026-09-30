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
remedy: "Waits on ruling E (amend or always supersede): if never amended, state in .abcd/development/decisions/adrs/README.md that an accepted ADR's decision text is never edited, a change mints a successor with supersedes, and the old record receives only its status, superseded_by and a forward-pointer line; if amendment is admitted, add a typed amends and amended_by pair beside supersedes (the form adr-2609231048308186 used on adr-19 and adr-20), under the same forward-pointer rule. Prove either with a docs test that the ADR README states the rule and a record-lint check that both halves of the chosen pair are present."
deferred_after: "v0.11.1"
deferral_reason: "ruling owed to the product thinker (renewed by run A 2026-09-29 after the v0.10.0 grant lapsed at the v0.11.0 anchor): Is an ADR never amended in place, always superseded, given E2 had adr-2609231048308186 amend adr-19 and adr-20?"
---

The convention is that a decision record is never amended in place: when a decision changes, a new record is minted and the old one is marked superseded by it, so the decision history stays legible and a reader of the old record is pointed forward rather than misled. Nothing in the tree states it. The typed-link vocabulary is recorded in .abcd/development/principles/decompose-before-filing.md as supersedes / reverses / duplicates / refines, and adr-58 and adr-2609021016286571 demonstrate the shape in practice, but the rule that amendment is not an option is carried only by example, so an agent or a contributor reaching for the smaller edit has nothing to read that refuses it. The gap surfaced when a relocation of the transcript corpus made adr-29's stated location wrong and the question of amend-versus-supersede had to be asked rather than looked up. Fix: state the rule where the conventions live, in the AGENTS.md decisions section or as a principle under .abcd/development/principles/, naming the two typed links and the one edit a superseded record does receive, and say why: an amended record erases the fact that the decision changed, which is the thing the record family exists to preserve. Detector: the conventions text names the rule and a docs test or record-lint rule refuses a decisions section that omits it, or at minimum the rule is present where a contributor reading AGENTS.md would meet it before editing a record.

## Remedy grounds (2026-09-29)

- Both answers keep the old decision text intact and differ only in whether a partial change has its own link word, which is the fork adr-2609231048308186 exposed.
- SOTA check: Nygard's original ADR post (https://www.cognitect.com/blog/2011/11/15/documenting-architecture-decisions, read 2026-09-29) keeps a reversed decision and marks it superseded; adr-tools (https://github.com/npryce/adr-tools, src/adr-new, read 2026-09-29) supports both a supersede flag and an Amends/Amended by link, so both answers have prior art.
- Rejected: editing the accepted decision text in place, which both sources and the record refuse.
