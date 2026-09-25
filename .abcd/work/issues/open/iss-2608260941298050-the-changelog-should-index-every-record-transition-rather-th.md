---
schema_version: 1
id: "iss-2608260941298050"
slug: "the-changelog-should-index-every-record-transition-rather-th"
severity: "major"
category: "architectural-insight"
source: "user-observation"
found_during: "changelog design discussion 2026-08-26"
found_at: "internal/core/changelog/shipped.go"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Do ADR and principle transitions get their own changelog section or a line in the existing groups?"
---

the changelog should index every record transition rather than curate a subset, which deletes the inclusion judgement and the reason shipped_in exists. Design decision of 2026-08-26, recorded in DECISIONS.md and resting on the less-but-better principle. Today the cut reads two families (intents/shipped, issues/resolved) and renders only records whose impact is non-internal, so impact carries TWO jobs: it decides the version bump, which it must, and it silences a changelog line, which is a publication judgement bolted onto a product one. checkIssueImpact's own comment names the cost — a rule refusing internal would force work into a user-facing changelog or push authors into a mislabel. Measured consequence of removing the judgement: v0.6.2 saw 175 records enter terminal folders against 98 rendered lines, of which 57 records were internal, so that release becomes roughly 175 entries. Also in scope: principles (30 files) and ADRs (45) are invisible to the cut today, and a new principle is arguably more consequential than half the issues that render. NOT in scope: bundling stays, because one line citing several records that were one user-visible change is fewer lines carrying the same information, which is the better half rather than curation; and impact keeps its version arithmetic. Residue that is NOT a migration artefact and needs a decision: AGENTS.md makes a stale closure legal forever, so some transitions carry no code change even in a greenfield repo, and under this model they render.