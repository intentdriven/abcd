# Requirements traceability, prioritisation and record governance: state of the art

Dated 2026-10-23; recency is assessed against that date. A state-of-the-art
pass over eight open intents that share one cluster: how a brief traces to the
records that realise it, how a record is corrected or routed once written, and
how the product thinker's order among ideas is declared, derived and shown.
The intents it serves:
[itd-2610040817105016](../../intents/drafts/itd-2610040817105016-a-product-thinker-reads-their-brief-as.md)
(every brief section shows what realises it),
[itd-61](../../intents/drafts/itd-61-brief-change-derivation.md)
(a brief edit cannot commit until its implied records are reconciled),
[itd-2609150819439571](../../intents/drafts/itd-2609150819439571-no-supported-way-to-correct-a-factual.md)
(correcting a factual error inside a durable record),
[itd-44](../../intents/drafts/itd-44-fourth-intent-kind-decision.md)
(a capture-time `decision` verdict routed into the ADR store),
[itd-78](../../intents/drafts/itd-78-intent-dependency-graph.md)
(the intent dependency graph),
[itd-41](../../intents/drafts/itd-41-phase-negotiator.md)
(what a sequencing choice costs),
[itd-2610070549060530](../../intents/drafts/itd-2610070549060530-a-product-thinker-who-wants-to-come-back.md)
(a word marks an idea to revisit) and
[itd-2610070550516046](../../intents/drafts/itd-2610070550516046-on-abcd-s-dashboard-a-product-thinker.md)
(the brief and the ideas together on the dashboard, in the product thinker's
order).

It builds on the
[brief-to-record traceability note of 2026-10-04](2026-10-04-brief-to-record-traceability-sota.md),
whose findings and corrected figures stand and are not repeated here, and on
the [intent dependency sweep of 2026-07-08](2026-07-08-intent-dependency-sweep.md).
Every outside source linked below was opened in this pass; a source that could
not be retrieved is named in words under "Not verified" and carries no link.
The shared web-search budget ran out partway through the pass, so the later
angles rest on pages fetched directly rather than on fresh searches; the gaps
that left are listed at the end. The proposer here is not the challenger: the
independent fit-challenge each intent is owed runs at plan time, and "Local
fit risks" under each section is its input.

Tiers: **[EVIDENCE]** a primary document, a measurement or a peer-reviewed
study, named; **[CONSENSUS]** several independent tools or practitioners agree;
**[CONTESTED]** credible disagreement, both sides given;
**[ANECDOTE/MARKETING]** one report or a vendor claim with no method behind it.

## itd-2610040817105016: every brief section shows what realises it

The 2026-10-04 note settled the shape (a stable identifier per section, a
`realises` link stamped with a fingerprint, a coverage report, status in the
links and never in the prose). The gap researched here: how identifiers stay
stable, what a suspect link's life cycle looks like in the tools that have one,
and what the evidence says traceability buys.

- Identifiers derived from heading text are not stable. Pandoc's
  `auto_identifiers` lowercases the text, strips punctuation, joins words with
  hyphens and suffixes duplicates `-1`, `-2` in document order, so a renamed or
  moved heading gets a new identifier; an explicit `{#identifier}` attribute is
  the stable form. [EVIDENCE] ([Pandoc manual, auto_identifiers and header_attributes](https://pandoc.org/MANUAL.html))
- Requirements tools make the identifier a field, not the heading. sphinx-needs
  computes "a short hash value based on the title" when `id` is omitted and
  otherwise requires the id to match `needs_id_regex`; StrictDoc "does not
  impose any limitations on the format of a UID", keeps it optional, and adds a
  machine identifier that "can uniquely identify a given node, independent of
  other fields like UID or TITLE" so its Diff/Changelog matches nodes "even if
  they are moved, renamed". [CONSENSUS] ([sphinx-needs need directive](https://sphinx-needs.readthedocs.io/en/latest/directives/need.html); [StrictDoc user guide](https://strictdoc.readthedocs.io/en/stable/stable/docs/strictdoc_01_user_guide.html))
- A suspect link is cleared per parent, by an act, never by time. Doorstop's
  link pairs the parent UID with the parent's fingerprint ("The link
  fingerprint is used by Doorstop to detect when a parent item is changed"); the
  fingerprint covers the item UID, `text`, `ref` and `references`, the UIDs in
  `links` and any extended attributes opted in, and excludes `header`,
  `level`, `active`, `derived`, `normative` and the link fingerprints;
  `doorstop clear LLT005 REQ002 REQ003` clears the named parents only, and the
  docs say "You should not calculate this value manually, use `doorstop
  review`". [EVIDENCE] ([Doorstop item reference](https://github.com/doorstop-dev/doorstop/blob/develop/docs/reference/item.md); [Doorstop validation](https://github.com/doorstop-dev/doorstop/blob/develop/docs/cli/validation.md))
- Jama's suspect links fire only on fields an administrator chose ("Only
  updates to certain fields trigger suspect links"), and are cleared by hand
  when "a suspect link has no impact or if you make the necessary changes",
  one link or "Clear All". The field filter is the tools' answer to suspicion
  fatigue: fingerprint the normative text, not the whole file. [CONSENSUS with
  Doorstop] ([Jama Connect, clear suspect links](https://help2.jamasoftware.com/ah/en/manage-content/coverage-and-traceability/relationships/clear-suspect-links.html))
- The one controlled experiment found traceability pays at maintenance time:
  71 subjects on real tasks were on average 24 per cent faster and produced 50
  per cent more correct solutions with trace links than without. [EVIDENCE]
  ([Mäder and Egyed, Software Engineering 2016](https://dl.gi.de/items/690677a5-bbad-45af-8843-2380ea5ee92f/full))
- The auditable shape for a model's read of coverage is Spec Kit's `analyze`:
  "STRICTLY READ-ONLY", one table of `ID | Category | Severity | Location(s) |
  Summary | Recommendation`, a coverage-gap pass ("requirements with no tasks,
  tasks with no mapped requirement"), capped at 50 rows. [EVIDENCE]
  ([Spec Kit analyze template](https://github.com/github/spec-kit/blob/main/templates/commands/analyze.md))
- The one-way change-folder alternative: OpenSpec keeps each proposed change
  in its own folder (`proposal.md`, spec deltas under `## ADDED Requirements`,
  `tasks.md`) and an archive step moves it into a dated folder and updates the
  specs. [EVIDENCE] ([OpenSpec](https://github.com/Fission-AI/OpenSpec))

**Options weighed.** (a) Adopt a requirements tool (Doorstop, StrictDoc,
sphinx-needs) as the brief's store: each brings its own runtime and format and
would make the brief a tool's document rather than the product thinker's. (b)
Native: an explicit marker per section, a `realises` link carrying the
section's fingerprint, a lint that lists suspect links, a coverage report in
the three states, a clear-by-act verb. (c) Model-recovered links with no
declared link at all: the 2026-10-04 note's figures (trace recovery F1 under
0.5; impact analysis at under 50 per cent precision) rule it out as the
record.

**Recommendation: Path 2.** Native, shaped to Doorstop's link-plus-fingerprint
and OpenFastTrace's coverage states as the 2026-10-04 note already recommends,
with two seams: the fingerprint form is one primitive shared with itd-78's
edge fingerprints, and the section identifier is an explicit marker the
rendering never derives from heading text. Clearing a suspect link is an act
on the named link (Doorstop's `clear <child> <parent>`), recorded, never a
timeout. The fingerprint covers normative text only (headings and body after
whitespace normalisation), per Jama's field filter; whether linked ids enter
it, as Doorstop's link UIDs do, is a choice the one primitive makes on
purpose, since itd-78's edges and itd-61's trigger share it. No Go dependency.

**Local fit risks.** The user's directory is theirs: markers go into the
product thinker's brief prose, across 111 files, and a migration that rewrites
their text needs their consent and a one-off, not a creeping gate. One
canonical primitive: the fingerprint and the suspect state must be the same
thing itd-78 uses for edges and itd-61 uses as its trigger, or three
near-identical hashes appear. Loud staging: a suspect link is a shown state,
not a silent one. Unrecognised input never writes: clearing is explicit, and a
`realises` pointing at an unknown section is refused at write time.

## itd-61: a brief edit cannot commit until its implied records are reconciled

The 2026-10-04 note covers the classification half (a model proposes "new
intent, issue, contradiction owing a decision, no impact"; the product thinker
confirms; the figures say "no impact" cannot be trusted alone). The gap
researched here: where the block lives, how trivial edits are kept out, and
what shape the dedup and the pass take.

- Where a commit-time block lives and what it is worth: git's `pre-commit`
  hook is "invoked before obtaining the proposed commit log message and making
  a commit. Exiting with a non-zero status from this script causes the git
  commit command to abort", and it "can be bypassed with the `--no-verify`
  option"; hooks live in `$GIT_DIR/hooks` or wherever `core.hooksPath` points,
  outside tracked content. A hook is the early, loud copy of a gate; the gate
  that holds is the one CI and the preflight run. [EVIDENCE] ([githooks](https://git-scm.com/docs/githooks))
- The trivial-edit filter is the fingerprint's scope, not a second heuristic,
  and the scope is narrower than it looks: Doorstop's fingerprint covers the
  item UID, `text`, `ref` and `references`, the UIDs in `links` and opted-in
  extended attributes, not `header`, `level` or layout; Jama fires only on
  configured fields. An edit that changes no normative text (a reflow, a
  whitespace change) changes no fingerprint and fires nothing, by
  construction; a typo fix inside normative text does change it, so an
  editorial edit needs a cheap clear-by-act (an "editorial, no impact" row
  the product thinker confirms in one step), or the intent's open question on
  a pre-filter stays open. [CONSENSUS] ([Doorstop item reference](https://github.com/doorstop-dev/doorstop/blob/develop/docs/reference/item.md); [Jama Connect](https://help2.jamasoftware.com/ah/en/manage-content/coverage-and-traceability/relationships/clear-suspect-links.html))
- The errata world draws the same line in words: the IESG tells reviewers
  that "only items that are clearly wrong should be considered", that an
  editorial erratum "does not affect the technical content", and that
  "Significant clarifications should not be handled as errata reports". For a
  brief: editorial edits pass; technical edits fire the pass; an edit that
  changes what the product is owes a record, not a reconciliation note.
  [EVIDENCE] ([IESG statement on processing errata, 2021](https://datatracker.ietf.org/doc/statement-iesg-iesg-processing-of-rfc-errata-for-the-ietf-stream-20210507); [RFC Editor errata definitions](https://www.rfc-editor.org/errata-definitions/))
- Single pass, then confirm, is the shipped shape: Spec Kit's `analyze` runs
  once, read-only, lists findings with a location each, and "Any remediation is
  only offered, and the user must approve it". Its duplication pass
  ("near-duplicate requirements, with the weaker phrasing flagged") is a
  model's judgement with no published recall. [EVIDENCE] ([Spec Kit analyze template](https://github.com/github/spec-kit/blob/main/templates/commands/analyze.md))
- The change-folder alternative keeps the brief edit itself out of the brief
  until reconciled: OpenSpec's proposal folder holds the delta and the archive
  step applies it. [EVIDENCE] ([OpenSpec](https://github.com/Fission-AI/OpenSpec))
- Accuracy ceiling, restated from the 2026-10-04 note: automated change-impact
  analysis reached 86 to 96 per cent recall at under 50 per cent precision on
  one dataset; pairwise contradiction detection about 65 per cent recall for a
  hybrid and 0 per cent for the model alone. A deterministic suspect set with
  a model's proposal on top is the only honest division. [EVIDENCE]
  ([2026-10-04 note, Correction](2026-10-04-brief-to-record-traceability-sota.md))

**Options weighed.** (a) Block in a pre-commit hook only: cheap, bypassable,
and absent in a checkout that never installed hooks. (b) Block at the
record-lint gate (preflight and CI) with the hook as early warning: holds for
every route a change can take to `main`. (c) Edit-as-proposal (OpenSpec): the
brief edit lands in a change folder and is applied on reconciliation; honest,
but it takes the brief out of the product thinker's hands between edit and
apply, which the forward path (itd-60) and the dashboard's note flow do not.
(d) Interactive grill on every edit: truest to the instrument, heaviest per
edit, and it needs a session to be open at the moment of the edit.

**Recommendation: Path 2.** Native: the trigger is the deterministic suspect
set from itd-2610040817105016's fingerprints (so whitespace-only edits fire
nothing and a typo fix is cleared in one step); the pass is single-shot and
read-only in Spec Kit's row shape, with the model's classification proposed
and the product thinker confirming each row, "no impact" included; the block
is the record-lint rule (a brief section whose fingerprint moved and whose
suspect links are uncleared fails `record-lint`), with the pre-commit hook as
the loud early copy. This moves the hold from the commit to the push and
merge gates, which the acceptance criterion "blocked from committing" does not
yet say; its rewording to "blocked from reaching main" is the product
thinker's at planning. The seam is the finding table:
the proposer behind it can change without the gate changing. Dedup against the
forward path is the provenance marker itd-60's writes carry, with content
matching as the fallback a model proposes and never applies. This does not
contradict the intent's "any change fires the pass": it narrows "any" to any
change of normative text, which is what the dedup was meant to absorb anyway.

**Local fit risks.** Facilitator default, thinker optional: a fail-closed
block that needs a session to clear it stalls the product thinker who edits
their brief on the dashboard at night; the gate must hold the commit, not the
edit. Host-delegated LLM work: the classification runs on the host; the gate
must be decidable with no model present (uncleared suspect links fail; a
missing proposal is a missing proposal, not a pass). One canonical primitive:
one assessment verb serves itd-61 and itd-2610040817105016, not two. Widen,
never recommend: the confirm question offers new intent, issue, decision, no
impact and "not now", and the model's pick is one option among them.

## itd-2609150819439571: correcting a factual error inside a durable record

The product thinker's ruling M25 orders a convention before any verb, through
the planning interview, which opens on one question: per record family, where
does a correction go, what does it carry, and why does the original stay? The
research below is the material for that interview.

- The RFC model is the closest fit: "Because RFCs do not change after
  publication, there is an errata system to note these errors." An erratum is
  technical ("an error in the technical content") or editorial, and carries
  one of four statuses: Reported, Verified ("edited as necessary and verified
  by the appropriate parties"), Rejected ("redundant or incorrect") and Held
  for Document Update ("not a necessary update to the RFC. However, it should
  be considered in future revisions"). [EVIDENCE] ([RFC Editor errata definitions](https://www.rfc-editor.org/errata-definitions/))
- The IESG's triage rule is the one the interview needs: Verified when the
  correction "should be available to implementers or people deploying the
  RFC"; Held when "not a necessary update"; Rejected when "invalid or proposes
  a significant change", because "Significant clarifications should not be
  handled as errata reports". An erratum is reported as Original Text and
  Corrected Text with a comment. [EVIDENCE] ([IESG statement, 2021](https://datatracker.ietf.org/doc/statement-iesg-iesg-processing-of-rfc-errata-for-the-ietf-stream-20210507))
- Scholarly publishing keeps the original and links the notice: an article
  should "remain extant, exact and unaltered as far as is possible", and a
  correction is "a notice that will be permanently linked to the article"; a
  corrigendum corrects an author's error, an erratum the publisher's; a
  retraction notice is titled "Retraction: [article title]", signed, listed
  and shown before the article; an expression of concern "remains part of the
  permanent record". [CONSENSUS across publishers; COPE's own guideline could
  not be retrieved, see below] ([Elsevier article-withdrawal policy](https://www.elsevier.com/about/policies-and-standards/article-withdrawal))
- The reader-facing half is a status on the original: Crossmark "gives
  readers quick and easy access to the current status of an item of content"
  and "lets readers know when a substantial change affecting the citation or
  interpretation has occurred", on the page and inside an already-downloaded
  PDF. A correction nobody can see from the record it corrects is the failure
  the issue describes. [EVIDENCE] ([Crossmark](https://www.crossref.org/services/crossmark/))
- Standards split correction from change by consequence, and admit the line
  is soft: a technical corrigendum corrects "a technical error or ambiguity
  in an IS inadvertently introduced either in drafting or in printing" that
  "could lead to incorrect or unsafe application"; an editorial defect has "no
  consequences in the application"; an amendment publishes "a technical
  addition or change"; a committee expert calls the Directives "deliberately
  vague" on the boundary. [EVIDENCE for the definitions; CONTESTED on the line]
  ([SC 34 WG 4 list, August 2010](https://mailman.vse.cz/pipermail/sc34wg4/2010-August/001720.html))
- ADR tooling has no correction primitive at all: Nygard's rule is "If a
  decision is reversed, we will keep the old one around, but mark it as
  superseded"; MADR's status set is "proposed | rejected | accepted |
  deprecated | … | superseded by ADR-0123"; log4brains states "an ADR is <!-- record-lint: illustrative -->
  immutable. Only its status can change"; adr-tools offers `adr new -s N` to
  supersede. A wrong sentence in an ADR is therefore fixed today by a whole new
  ADR, which is heavier than the case here needs. [CONSENSUS] ([Nygard, 2011](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions); [MADR](https://adr.github.io/madr/); [log4brains](https://github.com/thomvaill/log4brains); [adr-tools](https://github.com/npryce/adr-tools))
- Git notes attach text to an object "without touching the objects
  themselves", under `refs/notes/commits`, appended with `git notes append`
  and merged with their own strategies; they are a separate ref a reader must
  know to fetch and display. [EVIDENCE] ([git-notes](https://git-scm.com/docs/git-notes))

**Options weighed.** (1) An appended, dated erratum inside the record, in the
RFC shape (quoted original, corrected text, kind, why the original stands,
who verified), as the issue's remedy already proposes. (2) A separate errata
ledger in the RFC Editor's shape, with the record carrying only a rendered
badge (Crossmark): cleaner status handling, a second store to keep in step.
(3) Git notes: the mechanism exists and is invisible from the file a reader
opens; nothing in abcd reads notes. (4) Supersede the whole record (the ADR
convention): right for a changed decision, wrong for a misquoted sentence.
(5) Edit in place and point at git history: erases the evidence an audit
reasoned from, which is why the issue rejects it.

**Recommendation: Path 2.** Native, shaped to the RFC erratum: one appended
`## Errata` section (or, for a shipped intent, an erratum-marked entry in the
already append-only Audit Notes, if the interview rules per family) whose
entry carries five fields: the quoted original, the corrected text, the kind
(technical or editorial), why the original stands, and who verified it on
which date, by role. The IESG's triage becomes the convention's boundary: a
correction that changes what was decided is a supersession, a correction of
outdated information is not an erratum (the convention's own boundary; the
current ISO/IEC Directives were not opened, so no ISO wording is quoted), and
a significant clarification is a new record. The rendered half follows Crossmark: the board, the site and
`abcd <record-id>` show "has errata" on the original. A record-lint rule
proves the five fields; a verb follows only if the interview asks for one, as
M25 says. The seam is the entry's shape: a later errata ledger can be derived
from the entries, never the other way round.

**Local fit risks.** The record lands with the act: the erratum is written in
the same diff that found the error, which argues for a shape writable by hand
from the start. One canonical primitive: shipped intents already have an
append-only, dated Audit Notes section, and the interview must decide whether
an erratum is a kind of note there or a section everywhere, not both. Guards
prove themselves: a convention with no lint is a remembered discipline. The
Reported status is deliberately dropped at first (nobody queues errata in a
repository with one product thinker); if the lab shows disputed corrections,
it returns.

## itd-44: a capture-time `decision` verdict routed into the ADR store

- The field's definition of the thing the verdict detects: an architectural
  decision is "a justified design choice that addresses a functional or
  non-functional requirement that is architecturally significant", and an
  architecturally significant requirement is "a requirement that has a
  measurable effect on the architecture and quality of a software and/or
  hardware system"; MADR adds that "architecture" should not be read too
  strictly. A standing choice that answers a requirement without being a user
  moment is exactly the verdict's signature. [CONSENSUS] ([adr.github.io](https://adr.github.io/); [MADR](https://adr.github.io/madr/))
- Nygard's bar for "architecturally significant" is structure, quality
  attributes, external dependencies, interfaces and construction techniques;
  this repository's ADR README sets a stricter three-part test (hard to
  reverse, surprising without context, a real trade-off). The verdict should
  carry the stricter local test, or the ADR store fills with "we use library
  X" entries. [EVIDENCE] ([Nygard, 2011](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions))
- Store semantics the intent already matches: status-field supersession
  (Nygard; MADR's "superseded by ADR-0123"; `adr new -s`), one record per <!-- record-lint: illustrative -->
  decision, immutability after acceptance (log4brains). The thin adoption's
  choice of the existing store over a second lifecycle is the field's
  convention. [CONSENSUS] ([adr-tools](https://github.com/npryce/adr-tools); [log4brains](https://github.com/thomvaill/log4brains))
- MADR 4's optional "Confirmation" section ("how compliance with the decision
  will be checked, such as a design review or an ArchUnit test") is the hook
  the intent's out-of-scope "decision-aware review" would later attach to.
  [EVIDENCE] ([MADR](https://adr.github.io/madr/))
- What models can do here is measured only for generation, not for the
  verdict: GPT-4 zero-shot "generate relevant and accurate Design Decisions"
  from an ADR's Context but "fall short of human-level performance" (ICSA
  2024); a 2026 benchmark of ten small models finds robust zero-shot ADR
  generation above about 3B parameters and that "high semantic diversity in
  off-the-shelf small models often correlates with hallucination". No
  published figure covers classifying a one-line idea as decision versus
  capability, so the intent's fixture set is the right instrument and the
  confirm step is load-bearing. [EVIDENCE for generation; no evidence for the
  verdict] ([Dhar, Vaidhyanathan and Varma, arXiv 2403.01709](https://arxiv.org/abs/2403.01709); [Vo et al., arXiv 2603.07091](https://arxiv.org/abs/2603.07091))
- A YAML concrete syntax for MADR (YADR, June 2026) exists for records meant
  to be validated against a schema; it changes the syntax, not the sections.
  [EVIDENCE] ([Zimmermann, YADR introduction](https://ozimmer.ch/practices/2026/06/19/YADRTemplateIntroduction.html))

**Options weighed.** (a) A fourth persisted kind and its own store: rejected
by the operator deviation, and the field agrees (one store, status-field
supersession). (b) A verdict that routes into the existing store through the
same mint `abcd decide --adr` uses: the intent's thin adoption, with one
correction, that the store no longer numbers by "next-free" but mints
timestamp ids through adr-45's allocator. (c) No classifier, only a prompt in
the capture flow asking "is this a standing choice?": cheapest, and it loses
the regression fixtures that make the verdict testable.

**Recommendation: Path 1 for the artefact and Path 2 for the verdict.** The
artefact is the existing ADR store with Nygard/MADR semantics, as ruled. The
verdict is native: a host-run classification proposed against the repository's
three-part ADR test, fixture-tested, with the confirm question widened to the
four homes an idea can have (ADR, intent draft, issue, drop). Two stale
details in the draft body are noted here rather than edited: the store mints
`adr-<yymmddHHMMSS><rrrr>` through adr-45's allocator, not a next-free ordinal,
and `abcd decide --adr` already writes the README-exact skeleton, so the
verdict's writer is that call, not a second template.

**Local fit risks.** One canonical primitive: two routes into the ADR store
(`decide --adr` and the capture verdict) must share one mint and one skeleton
writer. Host-delegated LLM work: "a CLI primitive that returns a stable JSON
verdict" reads as a native classifier; in abcd the model is host-run, so the
primitive validates and records the host's verdict against fixtures rather
than computing it. Widen, never recommend: the confirm question widens. The
`blocked_by: [itd-34]` edge should be re-read at plan time against where the
three-kinds taxonomy stands today.

## itd-78: the intent dependency graph

- Priority inheritance, the intent's named model, is computed and never
  stored: the Linux rt-mutex design keeps a tree of top waiters per owner and
  "simply need[s] to compare the top pi waiter to its own normal/deadline
  priority and take the higher one", recomputing when a waiter leaves so the
  boost drops again; chains "may converge, but a chain would never diverge",
  because "a process can't be blocked on more than one mutex at a time". An
  intent may be blocked by several intents, so the precedent carries over for
  compute-on-read and the drop when a waiter leaves, not for the chain shape:
  inheritance here is a max over a DAG rather than a single chain. The
  protocol traces to Sha, Rajkumar and Lehoczky (IEEE Transactions on
  Computers, 1990). [EVIDENCE] ([rt-mutex design](https://docs.kernel.org/locking/rt-mutex-design.html); [priority inheritance](https://en.wikipedia.org/wiki/Priority_inheritance))
- Forges store the edge once and render both directions, and derive no order
  from it: GitHub issue dependencies went generally available on 2025-08-21
  ("blocked by" or "blocking", "up to 50 issues for each relationship type",
  filters `is:blocked`, `is:blocking`, `blocked-by:`, `blocking:`, "fully
  supported in the API and webhooks"); the docs say nothing about ordering,
  priority or cycles, and `gh issue edit --add-blocked-by` writes the edge.
  Linear's `m b` / `m x` shortcuts mark blocked and blocking. [EVIDENCE]
  ([GitHub changelog](https://github.blog/changelog/2025-08-21-dependencies-on-issues/); [GitHub docs](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/creating-issue-dependencies); [Linear changelog, 2020](https://linear.app/changelog/2020-03-04))
- Every scoring scheme punts dependencies to a separate step. Reinertsen's
  cost of delay divided by duration reorders three features from a $69,000 to
  a $27,000 delay cost in the worked example; SAFe's relative variant is
  attacked by the method's own popularisers ("You simply can't make trade-off
  decisions with relative value estimates"; adding time criticality to value
  "doesn't make economic sense"); RICE's authors concede "One project may be a
  dependency for another project, so it needs to happen first". A derived order
  from edges is what none of them computes. [CONTESTED on scoring; CONSENSUS
  that edges override scores] ([Black Swan Farming, WSJF](https://blackswanfarming.com/wsjf-weighted-shortest-job-first/); [Black Swan Farming on SAFe's WSJF](https://blackswanfarming.com/safe-and-weighted-shortest-job-first-wsjf/); [SAFe WSJF, preview](https://framework.scaledagile.com/wsjf/); [Intercom, RICE](https://www.intercom.com/blog/rice-simple-prioritization-for-product-managers/))
- Text-based traceability tools store one direction and derive the other:
  Doorstop links point at parents and children are computed; sphinx-needs
  declares one link type with an incoming and an outgoing label ("tests" / "is
  tested by") from a single declaration. The intent's refusal of hand-authored
  reverse fields matches. [CONSENSUS] ([Doorstop item reference](https://github.com/doorstop-dev/doorstop/blob/develop/docs/reference/item.md); [sphinx-needs need directive](https://sphinx-needs.readthedocs.io/en/latest/directives/need.html))
- Local state since the draft was written: phases and milestones are retired
  (adr-2609212115255771: "Sequencing is dependencies plus the shelves";
  Now/Next/Later "is a rendered status, never stored"), so the two phase
  lints in the draft have no object; the shipped pick (itd-2609211116005482)
  already filters on `blocked_by` and reads a superseded blocker as its
  replacement, but inherits no severity; the revisit mark
  (itd-2610070549060530) and the product thinker's own order
  (itd-2610070550516046) are two further inputs to the same order. [EVIDENCE,
  local record]

**Options weighed.** (a) Store an effective priority: rejected by the
intent's own doctrine and by the scheduler precedent. (b) Derive at read time
over hard edges only, `builds_on` reported but unweighted, as the intent's
conservative start. (c) Replace the graph with a score (WSJF, RICE): the field's
own critics show the score cannot carry dependencies. (d) Adopt the forge's
dependency field as the store: abcd's intents are files, the forge mirror is
one-way, and the forge derives nothing.

**Recommendation: Path 2.** Native: `blocked_by` stored one way, `blocks` and
any stored mirror refused by lint, effective priority derived by inheritance
over hard edges at read time (the rt-mutex recomputation, including the drop
when a blocker ships), ties broken as the draft says, edges optionally
fingerprinted with the same primitive as section links. The seam is a single
ordering function the pick consumes, into which severity inheritance, the
record score, the revisit mark and the product thinker's order enter with a
stated precedence; this intent provides the inheritance term, nothing else.
The two phase lints are dropped at plan time as moot; the superseded-target
lint stays (the sweep found three live edges into a dead draft).

**Local fit risks.** One canonical primitive: four things now claim to decide
"what next"; without one function with one precedence they fight, and each
intent's spec would otherwise write its own. Less, but better: priority
inheritance over hard edges is enough for a corpus of about three hundred
records; weighted soft edges and unblock counts are speculation until a pick
is shown wrong. Reality is filable: the inheritance chain must be shown on the
board ("minor, effective major ← blocks itd-N"), or the derived order reads as
an opinion.

## itd-41: what a sequencing choice costs

- The record this intent rests on is gone: adr-9 (phases) is superseded by
  adr-2609212115255771, which retires the phase, the milestone and the word
  roadmap, renders Now/Next/Later from dependencies and the shelves, and makes
  the derived release the checkpoint. The negotiator's object, the phase plan,
  no longer exists in the record; its re-sequencing analysis over the
  dependency graph does. [EVIDENCE, local record]
- The field's sequencing conversations are about what a move costs, not about
  phases. Shape Up bets "only one cycle ahead", drops unchosen pitches, and
  revives an idea only when "brought back with a context, by a person, with a
  purpose"; a project that does not ship "doesn't get an extension" (the
  circuit breaker), and "There is nothing special about bugs that makes them
  automatically more important than everything else". [CONSENSUS among
  practitioner methods; ANECDOTE on outcomes] ([Shape Up, chapter 7](https://basecamp.com/shapeup/2.1-chapter-07); [Shape Up, chapter 8](https://basecamp.com/shapeup/2.2-chapter-08))
- Now-Next-Later exists to stop dates becoming promises: "Putting features in
  dated columns turns every date into a commitment, and when priorities shift,
  those commitments break"; "roadmaps communicate direction, while OKRs
  communicate commitment". abcd's retired-phase decision is the same move.
  [CONSENSUS] ([ProdPad glossary](https://www.prodpad.com/glossary/now-next-later-roadmap/))
- The one quantitative answer to "what does deferring cost" is cost of delay,
  which needs a duration estimate per item; the intent's no-estimates rule
  excludes that term, and the SAFe variant that replaces duration with relative
  job size is the part its critics reject. Without durations, cost of delay
  degenerates to severity weighted by what waits behind an item, which is
  itd-78's derived order. [EVIDENCE for the method; CONTESTED on the proxy]
  ([Black Swan Farming, WSJF](https://blackswanfarming.com/wsjf-weighted-shortest-job-first/); [Black Swan Farming on SAFe's WSJF](https://blackswanfarming.com/safe-and-weighted-shortest-job-first-wsjf/))
- Story mapping gives the "work backwards from what each slice should make
  true" instrument the intent asks for, without phases: a backbone that "just
  is", a walking skeleton ("the smallest possible system you could build that
  would give you end to end functionality") built first, and release slices
  drawn as horizontal lines across the map so "we never release a car without
  brakes". [CONSENSUS; the article is from 2008, updated 2023] ([Patton, The New Backlog](https://www.jpattonassociates.com/the-new-backlog/))
- Grounded-or-asked is the shipped shape elsewhere: Spec Kit's `analyze`
  gives every finding a `Location(s)` cell and a severity, and offers
  remediation the user approves. A trade-off with no location is the defect
  the intent names. [EVIDENCE] ([Spec Kit analyze template](https://github.com/github/spec-kit/blob/main/templates/commands/analyze.md))

**Options weighed.** (a) Keep the phase negotiator as drafted: it proposes a
unit the record retired. (b) Re-scope at plan time to a re-sequencing
analysis: given a proposed move (pull forward, hold, defer, mark to revisit),
report from itd-78's graph what else waits, from acceptance criteria what is
put at risk, and ask a Socratic question wherever neither grounds the concern.
(c) Fold the whole intent into itd-78's report and the pick's falsifier text:
loses the Socratic half adr-10 decided.

**Recommendation: Path 2, a proposed re-scope** for the plan-time interview to confirm (the product thinker has not ruled it; the superseded-record fact comes first in that interview). Native, shaped to cost of delay's
question ("what does waiting cost, and who pays") without its numbers: the
asserted costs come from itd-78's transitive closure and the affected intents'
acceptance criteria, each with a location; everything else is a tagged
question, as adr-10 holds. The seam is itd-78's ordering function: the
negotiator reads it and never writes it. The product thinker's own order
(itd-2610070550516046) and the revisit mark (itd-2610070549060530) are the
"phasing choice" this intent now interrogates. Nothing ruled is contradicted;
the ADR the draft rests on is, and the plan-time interview owes the product
thinker that fact before anything else.

**Local fit risks.** Retire the name: "phase", "milestone" and "roadmap" are
retired words and the draft is written in them. No estimates: cost of delay's
duration term stays out. One canonical primitive: the grill, the negotiator
and the pick's falsifier are three Socratic surfaces; the draft's own open
question about a shared move taxonomy is the fit question. Host-delegated LLM
work: the questions are host-run; the grounded half is the binary's.

## itd-2610070549060530: a word marks an idea to revisit

- A marker at the start of the text is an established convention, read only
  in that position: org-mode's priority cookie sits in the headline "right
  after the TODO keyword" (`*** TODO [#A] ...`); Todoist reads `p1`, `p2` or
  `p3` typed into the task name and sets the priority when the task is added.
  [EVIDENCE] ([org-mode manual, Priorities](https://orgmode.org/manual/Priorities.html); [Todoist, set a priority](https://www.todoist.com/help/articles/set-a-priority-in-todoist-Wy82Jp))
- One level is a shipped design, and it is rendered as a short list, not a
  rank: Jira's flag "has only one value: Impediment", shows the card in yellow
  and takes a comment on adding or removing it; GitHub pins "up to three
  important issues" "above the issues list in your repository". The cap of
  three is the one guard the field has against a revisit list becoming a second
  backlog. [CONSENSUS] ([Jira, flag an issue](https://support.atlassian.com/jira-software-cloud/docs/flag-an-issue/); [GitHub, pinning an issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/administering-issues/pinning-an-issue-to-your-repository))
- Most trackers grow levels (org-mode A/B/C with B as the default; Todoist
  four with p4 the default), and in each the level "affects only sorting"
  (org-mode: "Priorities affect only sorting in the agenda"). The product
  thinker ruled one level; the field shows that is where levels start, not
  where they stay. [CONTESTED only in that sense] ([org-mode manual](https://orgmode.org/manual/Priorities.html))
- Where the mark sits in an order is a precedence rule, not a weight, in the
  tools that have one: Todoist puts high-priority tasks "near the top of the
  list, and below tasks with a date and time". The analogue: a revisit mark
  sits above every unmarked idea and below whatever has a harder claim (a
  lane already running, a blocker that must ship first). [EVIDENCE] ([Todoist](https://www.todoist.com/help/articles/set-a-priority-in-todoist-Wy82Jp))
- When a mark ends: Linear's snooze returns an issue "at a time of your
  choosing, or when there's new activity on that issue: whichever comes
  first"; Shape Up
  argues against any automatic return ("Anything brought back is brought back
  with a context, by a person, with a purpose"). The product thinker's ruling
  (ends when built or removed) is Shape Up's side. [CONTESTED] ([Linear, Triage](https://linear.app/docs/triage); [Shape Up, chapter 7](https://basecamp.com/shapeup/2.1-chapter-07))

**Options weighed.** (a) A one-bit mark in frontmatter set by a leading
marker word or a flag, rendered as a list on the board, a hard tier in the
pick: matches the six rulings. (b) A weight in the pick's score: no tool does
this, and the pick's "computed, never composed" reason could not name a
weight the product thinker would recognise. (c) Levels: ruled out; noted as
the direction every tool later took.

**Recommendation: Path 2.** Native: a one-bit `revisit` mark, set by a small
fixed set of leading words (`urgent:`, `important:`, `revisit:`) or the flag,
settable and removable later, with an optional one-line reason on add and
remove, as Jira allows ("You may want to do this to indicate your reason");
rendered as a short list at the top of the board; in the pick, a hard tier
ahead of every unmarked candidate, named as the placing rule in the pick's
reason, and where the product thinker's own order (itd-2610070550516046) is
present it governs, the mark placing an idea within or ahead of it as the one
ordering function states (the precedence is itd-78's open question). A soft
cap in GitHub's spirit (the board says "more than three ideas marked to
revisit" and does nothing else). The seam is the one ordering function shared
with itd-78 and itd-2610070550516046.

**Local fit risks.** One canonical primitive: the mark is one input to one
ordering function, not a second sort. Unrecognised input never writes: a
leading word outside the fixed set marks nothing and the filing says so; a
word elsewhere in the text marks nothing, as ruled. The pick's reason must
name the mark as the rule that placed the winner, or the computed reason
becomes false. Less, but better: no levels, no expiry, no weights until a pick
is shown wrong.

## itd-2610070550516046: the brief and the ideas together, in the product thinker's order

- Coverage views in the requirements tools are a filtered table with link
  columns and an on-demand graph with a cap: sphinx-needs' `needtable`
  "supports the full filtering possibilities", offers a plain table or an
  interactive one with browser-side sort and search; `needflow` draws needs
  and their links with `max_items`, and a truncated view "says so in the page
  and emits a `needs.max_items` warning". The network graph is a filtered view of a
  neighbourhood, never of everything. [EVIDENCE] ([needtable](https://sphinx-needs.readthedocs.io/en/latest/directives/needtable.html); [needflow](https://sphinx-needs.readthedocs.io/en/latest/directives/needflow.html))
- A coverage report's first row is the uncovered set: Spec Kit's `analyze`
  reports "requirements with no tasks" and a coverage percentage; a section
  nothing realises is the thing the product thinker asked to see "at a
  glance". [EVIDENCE] ([Spec Kit analyze template](https://github.com/github/spec-kit/blob/main/templates/commands/analyze.md))
- A position per item is how the tools built for concurrent editing store
  manual order, and only Figma states the reason: its fractional indexing
  ("Every object has a real number as an index"; insert at "the average index
  of the two objects on either side"; "Reordering an object only involves
  editing a single value"; concurrent inserts can interleave) is for realtime
  multi-user editing; Jira's LexoRank, a lexicographically ordered string
  field with a background balancer and failure modes when the database
  collation disagrees, and GitLab's manual sort, where "Each issue is assigned
  a relative order value" and "everyone who visits the same list sees the
  updated issue order", with list and board order one, store the same shape
  without stating a condition. With one writer, one ordered list of ids is
  enough; that it suffices is this pass's reading, not a rule the tools state. [EVIDENCE] ([Figma, 2017](https://www.figma.com/blog/realtime-editing-of-ordered-sequences/); [Atlassian LexoRank KB](https://support.atlassian.com/jira/kb/lexorank-balancing-or-manual-ranking-of-issues-fails-in-jira/); [GitLab, sorting issue lists](https://docs.gitlab.com/user/project/issues/sorting_issue_lists/))
- GitHub Projects lets a row be dragged when no sort is applied and says
  nothing about what a later sort does to that order. The honest design states
  it: a manual order and a computed sort are two modes, and the view shows
  which it is in. [EVIDENCE] ([GitHub Projects, table layout](https://docs.github.com/en/issues/planning-and-tracking-with-projects/customizing-views-in-your-project/customizing-the-table-layout))
- Now/Next/Later as the surrounding frame is already abcd's rendered status
  (adr-2609212115255771), and its inventors' rule is that the columns carry
  confidence, not dates. [CONSENSUS] ([ProdPad glossary](https://www.prodpad.com/glossary/now-next-later-roadmap/))

**Options weighed.** For the picture: (a) a coverage strip (one tile per
section, coloured by its realising intents' shelves) above the section list
with the ordered ideas beside it; (b) a grid of sections against intents,
readable to a few dozen of each; (c) a tree (brief chapters down to intents),
which hides cross-cutting intents; (d) a network graph, which the tools cap and
filter because an unfiltered one of three hundred nodes is unreadable. The
facilitator's own pass of 2026-10-07, recorded in the intent, reached (a) with
(d) ruled out for everything at once; this pass agrees and adds needflow's
rule: a graph of one section's neighbourhood on demand. For the order: (e) a
rank per record (fractional index or LexoRank) versus (f) one ordered list of
ids in one file. With one writer, a few hundred items and no concurrent
editing, (f) is enough and "reset to the scored pick" is deleting the file;
(e) is what concurrency needs and abcd does not have.

**Recommendation: Path 2.** Native: the coverage strip and section list from
itd-2610040817105016's trace index, the ordered list beside it, a filtered
neighbourhood graph on demand; the product thinker's order as one ordered
list of ids, kept where the dashboard's notes already live, read by the one
ordering function as the highest-precedence input when present (ruling 1) and
absent after a reset, with the revisit mark (itd-2610070549060530) placing an
idea within or ahead of it; the precedence between the two is stated once, in
the answer to itd-78's open question, not in each spec. The seam is that
function; the storage can move to a rank per
record if a second editor ever appears. The four mockups the product thinker
asked for remain owed to them; the recommendation is the proposer's pick, not
theirs.

**Local fit risks.** One writer per file: the dashboard writes the order while
a session reads it; the dashboard's existing note flow (collected for the
host, brought to a session) is the precedent. Durable state lives where the
platform says it survives: an order in a linked worktree's local tier dies with
the worktree; it belongs with the main checkout's dashboard state. The user's
directory is theirs: the order is the product thinker's and no gate reads it
as a rule. One canonical primitive: the product thinker's order, the revisit
mark, severity inheritance and the record score are one function with one
precedence, and ruling 1 of this intent (their order decides, reversible)
fixes the top of it.

## Not worth adopting

- A requirements-management tool as a dependency or a store (Doorstop,
  StrictDoc, sphinx-needs, OpenFastTrace): each brings a Python runtime and a
  document format of its own; the fingerprint, the coverage states and the
  clear-by-act verb are borrowed as shapes.
- Git notes for errata: invisible from the record a reader opens, a separate
  ref to fetch and display, and nothing in abcd reads notes.
- Supersession as the errata mechanism (the ADR convention): right for a
  changed decision, far too heavy for a misquoted sentence, and it would double
  the ADR store's entries for typos.
- Relative WSJF, RICE or any single score as the order: the methods' own
  authors concede dependencies override the score, and the relative variant
  cannot carry a trade-off; the derived graph order plus the product thinker's
  explicit order does what the score pretends to.
- A rank per record for the product thinker's order (fractional indexing,
  LexoRank): built for many concurrent editors over many items; one ordered
  list is enough for one product thinker and a few hundred ideas.
- Priority levels, weights or timed expiry on the revisit mark: the field's
  direction of travel, and ruled out; revisited only if a pick is shown wrong.
- An unfiltered network graph of brief sections and intents: the tools that
  draw one cap and truncate it with a warning.
- Auto-generated heading identifiers as section ids: they change with the
  text and the document order (Pandoc's `-1`, `-2` suffixes).
- A capture-time classifier that computes the `decision` verdict natively from
  keywords: no evidence either way, and the host-run model with fixtures and a
  confirm step is both the local preference and the testable form.

## Not verified

- COPE's retraction guidelines and the Cambridge University Press corrections
  policy: both pages returned HTTP 403 in this pass; the publisher consensus
  above rests on Elsevier's policy alone, and the "version 3, August 2025"
  dating of COPE's guideline seen in a search snippet is unconfirmed.
- Ruiz, Hu and Dalpiaz, "Why don't we trace?" (Requirements Engineering,
  2023): the publisher page redirected to a sign-in and the institutional copy
  refused the connection; the search snippet's finding that practitioners
  judge traceability's benefits to exceed its costs while tracing mostly by
  hand is unverified against the paper.
- Carlshamre et al.'s 2001 industrial survey of requirements interdependencies
  (six dependency types; a fifth of requirements carrying three quarters of
  the dependencies): the abstract page returned no content; cited here from a
  search snippet only.
- The Next Release Problem literature (dependency-constrained release
  selection as search-based optimisation): titles seen in search results, no
  paper opened; it would bear on itd-41 only if estimates returned.
- The SC 34 WG 4 post quotes the JTC 1 Directives of its day (2010); the
  current ISO/IEC Directives Part 1 were not opened, and the ANSI explainer
  returned 403.
- Linear's issue-relations and priority documentation pages returned 404; the
  claim that resolved blockers turn green and move under Related, and that a
  blocked-by relation creates its inverse automatically, come from search
  snippets and a third-party guide, and are not relied on above.
- OpenFastTrace's coverage-state definitions were not re-opened (the reference
  pages returned 404); they are taken from the 2026-10-04 note.
- The Disciplined Agile critique of SAFe's WSJF (the point that prerequisite
  relationships need separate handling in the final order) could not be
  fetched; the same point is made by the Intercom RICE article, which was.
- Duplicate-detection research for itd-61's dedup question (embedding or
  model-based near-duplicate detection among issues and requirements) was not
  surveyed: the shared web-search budget ran out before that angle, and no
  figure is claimed.
- The Crossmark schema's list of update types (correction, erratum,
  addendum, retraction and the rest) was not retrievable (404); the vocabulary
  the errata recommendation uses comes from the RFC Editor and Elsevier pages
  that were.
- A compact Y-statement form for low-stakes decisions beside long-form MADR
  for significant ones appeared in a search snippet describing a third-party
  skill; the page was not opened and the distinction is not relied on.
