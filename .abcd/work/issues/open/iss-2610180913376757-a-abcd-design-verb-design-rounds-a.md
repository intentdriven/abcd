---
schema_version: 1
id: "iss-2610180913376757"
slug: "a-abcd-design-verb-design-rounds-a"
severity: "minor"
category: "future-work-seed"
source: "managed-repo"
found_during: "abcd inbox report rpt-2610111258569639 from a managed repository (root commit dd203ff780118159c16a4fbf48899c880e71a88e)"
origin: researcher-authored
production_mode: hand-written
found_at: "new verb (design); neighbours abcd lab, abcd ideate"
remedy: "Earn it in stages: finish the open design lab, run a second lab on a non-UI kind such as wording variants that also tests the cull-first protocol, then propose a design verb with assessment as its judging step, and decide by ADR whether the assessment page is served locally."
related_intents: [itd-2610171228577260]
---

A /abcd:design verb: design rounds, a person-only assessment, and a committed design history

A managed repository (Dessau) needed a visual reference for a planned intent. Agents generated ten HTML mockups in two rounds; the maintainer rejected round 1 as a whole, then judged all ten in one sitting in an HTML assessment page built in a lab, and chose a base design with amendments and borrowings. The record went into the repository as a design history (PR 206 above). The maintainer asked whether abcd should carry this as a verb.

Proposed shape (maintainer's preference: one user-facing /abcd:design, not a separate assess verb, and not under ideate, which judges whether to do something rather than what form it takes):

- People: /abcd:design lists runs and what each waits for; /abcd:design "<question>" starts a run linked to an intent, with criteria from abcd defaults then repo config then the run; /abcd:design assess <run> opens the judging page; /abcd:design decide files the result as a design-history entry, a dated decision line, a reference on the intent, and offers lessons for a taste file.
- Agents only: design brief <run> renders the generation brief including the taste file and lessons from rejected rounds; design variant add records each variant's round, parent, model and brief. The generating agent never judges; any self-review is labelled weak evidence and never counted.

What the lab found (protocol A: criteria first, blind overview tags, 25 pairs plus 5 side-swapped repeats, a 1-7 rubric with a reason on the top three, a Bradley-Terry ranking with bootstrap ranges): 9.5 minutes; 5 of 5 repeats agreed; no loops; top design chosen without override; all round-2 designs ranked above all round-1 designs. But pairs were judged at a glance (median 2.1 s at half-size side by side), followed looks, and did not separate the top four; the criteria step surfaced findability and phone width. The real decision was a composite (base plus amendments plus borrowings), which a single-winner result cannot hold, and the judge pointed at parts with screenshots after the session rather than pins during it.

Judge's own tool debrief: sortable result columns (e.g. by mean score); a way back to each design from the result; remove the finalist swap; serve the assessment locally so the result is saved without an export. Serving locally opens a listening port, so it needs a decision record (bind address, who may write the result) before it ships.

SOTA grounding gathered in the lab: Tohidi et al. CHI 2006 (designs judged alone are rated higher); Kinnear et al. 2025 (about 10 comparisons per item for comparative judgement reliability); Vitello and Bramley 2017 (adaptive reliability is inflated); LMSYS (Bradley-Terry over Elo); UK DfE design histories (keep what did not work); QOC and MADR for the record shape; Anthropic's harness article on generator self-grading.

Remedy the reporter proposes: Earn it in stages: finish lab-261011111706-9fed999, run a second lab on a non-UI kind (e.g. wording variants) that also tests the cull-first protocol, then propose /abcd:design with assessment as its judging step, and decide by ADR whether the assessment page is served locally.

Reported by a managed repository (root commit dd203ff780118159c16a4fbf48899c880e71a88e) through the abcd inbox as rpt-2610111258569639, a enhancement against abcd v0.13.3, surface new verb (design); neighbours abcd lab, abcd ideate.

Evidence:

- rpt-2610111258569639 (the report, kept in the inbox)
- lab-261011111706-9fed999
- https://github.com/intentdriven/Dessau/pull/206
- https://github.com/intentdriven/Dessau/pull/205
