---
schema_version: 1
id: "iss-2608260941298050"
slug: "the-changelog-should-index-every-record"
severity: "major"
category: "architectural-insight"
source: "user-observation"
found_during: "changelog design discussion 2026-08-26"
found_at: "internal/core/changelog/shipped.go"
deferred_after: v0.11.1
deferral_reason: "an intent and a lane owed (re-deferred at v0.11.1 by run A's major-triage lane): ruled 2026-09-23 (DECISIONS) that every terminal transition gets a changelog line, stale closures included, planned as its own intent; no intent is filed yet. Its planning owes one ruling: do ADR and principle transitions get their own changelog section, or a line in the existing groups?"
remedy: "Waits on the M15 planning interview (own section, or a line in the existing groups, for ADR and principle transitions): draft the intent that makes internal/core/changelog read every terminal folder (resolved and wontfix issues, shipped and superseded intents, ADRs, principles) and stops impact silencing a line while it keeps the version arithmetic; if an own section, render ADRs and principles under a Decisions heading; if existing groups, map them onto Added or Changed; either proven by a cut test on a fixture tree that every terminal transition renders exactly one line or one bundled line."
---

the changelog should index every record transition rather than curate a subset, which deletes the inclusion judgement and the reason shipped_in exists. Design decision of 2026-08-26, recorded in DECISIONS.md and resting on the less-but-better principle. Today the cut reads two families (intents/shipped, issues/resolved) and renders only records whose impact is non-internal, so impact carries TWO jobs: it decides the version bump, which it must, and it silences a changelog line, which is a publication judgement bolted onto a product one. checkIssueImpact's own comment names the cost — a rule refusing internal would force work into a user-facing changelog or push authors into a mislabel. Measured consequence of removing the judgement: v0.6.2 saw 175 records enter terminal folders against 98 rendered lines, of which 57 records were internal, so that release becomes roughly 175 entries. Also in scope: principles (30 files) and ADRs (45) are invisible to the cut today, and a new principle is arguably more consequential than half the issues that render. NOT in scope: bundling stays, because one line citing several records that were one user-visible change is fewer lines carrying the same information, which is the better half rather than curation; and impact keeps its version arithmetic. Residue that is NOT a migration artefact and needs a decision: AGENTS.md makes a stale closure legal forever, so some transitions carry no code change even in a greenfield repo, and under this model they render.

## Deferral 2026-09-29

Deferred past v0.11.1: an intent and a lane owed (re-deferred at v0.11.1 by run A's major-triage lane): ruled 2026-09-23 (DECISIONS) that every terminal transition gets a changelog line, stale closures included, planned as its own intent; no intent is filed yet. Its planning owes one ruling: do ADR and principle transitions get their own changelog section, or a line in the existing groups?

## Remedy grounds (2026-09-29)

Why: ruling M15 (2026-09-23) settles that every terminal transition gets a line, so only the placement fork is open. SOTA check: Keep a Changelog warns that a changelog mentioning only some changes can be as dangerous as none, while a commit-log dump is noise, and its six groups carry no decisions group (https://keepachangelog.com/en/1.1.0/, read 2026-09-29); changesets keep one file per change carrying its bump type and summary (https://github.com/changesets/changesets/blob/main/docs/intro-to-using-changesets.md, read 2026-09-29), which the record already is. Rejected: keeping internal as a silencer, which contradicts M15.
