---
schema_version: 1
id: "iss-2610262032038666"
slug: "the-press-release-says-abcd-intent-shape"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 215 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-9f4f54ef2aa3fe2f
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The quoted scope sentence in itd-34 predates the re-scope that limits the record to the bundle command and reclassify; both ends were true when written and the record says which holds."
---

The press release says `/abcd:intent shape` (Role 3) is designed and not yet built; the shipped itd-34 has that role in scope, running from the pre-commit hook and on demand.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:26` — "Two companions are designed and not yet built: `/abcd:intent grill` (per itd-27), a Socratic interview that stress-tests an intent, or a brief section, before it is planned; and `/abcd:intent shape` (Role 3), which keeps each intent's `kind` honest as the corpus grows."
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:90` — "It runs continuously via the pre-commit hook (writing findings to the latest report) and on-demand via `/abcd:intent shape`."

The brief lists the shape role among companions not yet built. itd-34, on the shipped shelf, includes in scope a shape-classification role that runs continuously through the pre-commit hook and on demand through `/abcd:intent shape`. Either a shipped intent's scope was never delivered or the brief understates what shipped.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 215.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The quoted scope sentence in itd-34 predates the re-scope that limits the record to the bundle command and reclassify; both ends were true when written and the record says which holds.
