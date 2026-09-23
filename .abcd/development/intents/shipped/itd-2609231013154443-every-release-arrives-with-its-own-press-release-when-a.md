---
id: itd-2609231013154443
slug: every-release-arrives-with-its-own-press-release-when-a
spec_id: spc-2609231435545473
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-67, itd-73]
severity: minor
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-24, itd-2609212103572513]
related_adrs: [adr-2609212115255771]
impact: additive
---

# Every release arrives with a press release composed from what shipped

## Press Release

> Every feature release arrives with its own press release. When a release is cut, abcd writes a short announcement to `RELEASE.md` at the top of the repository. The headline features are told as the moment a person notices them, the rest are listed by name, and each is in the words its own press release already uses, quotes included. The previous page moves to the release archive, so the top of the repository always shows the latest feature release and the history is one folder away. It looks back only: nothing still planned, no dates, no "coming next". A release of fixes alone leaves the page as it is and says why. The changelog stays the line-by-line record; `RELEASE.md` is the page a person reads first to learn what a release was for.
>
> "I used to work out what a release was for by reading forty changelog lines," said Iris, a product thinker. "Now the first page I open tells me, in the words I used when I asked for each piece."

## Why This Matters

A release is the checkpoint of this record (adr-2609212115255771, decision
3): the version is derived from what shipped and the changelog is composed
from the records that reached a terminal folder. What the checkpoint lacks is
the one paragraph a person reads to learn what the release was for. The phase
carried that paragraph as its `## Expectation`, written before the work;
retiring the phase removed it, and the ADR's last consequence names a release
press release composed from the shipped intents' press releases as its nearest
home and leaves the step unruled. The product thinker ruled it on 2026-09-23
(DECISIONS.md, the 2026-09-23 rulings entry; the interview's fuller wording is
recorded in this intent's Decisions below): each release gets a press release
composed at the cut from its shipped intents' press releases, backward-looking,
and this release is the first to carry one.

The material already exists. Every shipped intent opens with a press release
written as the user moment, and the changelog composer (itd-67) already reads
those records at the cut to word its lines. The changelog answers "what changed,
line by line"; nothing answers "what is this release for" in the words the
product thinker used when each piece was asked for.

An intent's press release is written before its work, so a page composed from
it is only as true as the deliveries behind it. This intent does not claim
otherwise: its Mechanism states the expectation and what would show it wrong.

## Decomposition (itd-84 hand-run, 2026-09-23, confirmed by the product thinker at the planning interview)

Verdict: **SPLIT**. The website half is a second capability and is filed as its
own draft; the rest is this intent, already-recorded stance, or plumbing.

| Part | Type | Home |
|------|------|------|
| `RELEASE.md` at the repository root, composed at the cut from the shipped intents' press releases, with the outgoing page moved to an archive | capability | this intent |
| Release pages on the project website, rendered from `RELEASE.md` and the archive | capability | a separate draft intent, next cycle (planning ruling P7) |
| The release record reports only what shipped: no forecast, no promised date, no still-planned intent | standing stance | **already recorded**: adr-2609212115255771 (decision 3; "a roadmap promises, this view reports"). This intent follows it and declares nothing new |
| Every cited record is data the binary checks, and a mismatch is refused before anything is written | trust rule | **already recorded** for the changelog as spc-11's completeness bijection (itd-67); applied here to a second document, so no new ADR (confirmed by the product thinker) |
| The composer prompt, the ingest step in `launch ship`, the archive move, and their rollback with the changelog | plumbing | the spec `intent plan` mints; the launch surface is `brief/04-surfaces/04-launch.md` |

Relations, written as prose because the schema carries no typed field for them
([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)):

- refines adr-2609212115255771: this intent is the step the ADR's last
  consequence names and leaves unruled.
- refines itd-67: it adds a second composed document to the cut that itd-67's
  changelog slice built, read from the same record set.
- builds on itd-73: the cut's record set and the `impact` derivation are
  itd-73's, and the press release reads that set rather than deriving its own.
- itd-24 (release retrospectives) and itd-2609212103572513 (`target_release`)
  are neighbours: the first may read this page as a seed, and the second is the
  forward-looking line this page must never carry.

## What's In Scope

- `RELEASE.md` at the repository root, holding only the latest feature
  release's press release, with a heading that names that release's version.
- At each cut that writes a new page, the outgoing `RELEASE.md` moves to
  `.abcd/development/releases/<its version>.md`, the permanent-record tier.
- The page is composed at the cut by a composer agent from the press releases
  of the user-facing intents that entered `shipped/` since the previous
  release tag: the same record set the changelog is composed from, never a
  second derivation of it.
- Headline intents told as prose, every other intent in the set listed by name;
  the persona quotes of the intents it tells, carried word for word with their
  attribution.
- The binary validates the composer's output before anything is written, and a
  refused output is sent back to the composer with the reasons until it is
  valid.
- The page, the archive move and the dated changelog section land in one cut,
  or none of them does.
- The read-only preview (`abcd changelog`, or the `launch` dry run) listing the
  intents the page will be composed from.
- `RELEASE` added to the `stray_root_docs` allowlist and to CI's inert root list.
- This release: the next cut carries the first one.

## What's Out of Scope

- Any forecast: no `target_release` intents, no still-planned work, no date,
  no "coming next". The `target_release` report (itd-2609212103572513) stays
  in the cut's report and never enters the page.
- Issues as a source: an issue carries no press release; fixes stay in the
  changelog's **Fixed** lines. A release that ships fixes alone writes no page.
- Intents with `impact: internal`, and intents that left the record as Removed
  (superseded): neither is announced.
- Release pages on the project website: a separate draft intent (see
  Decomposition).
- The published GitHub release notes, which stay as they are.
- Pages for releases cut before this one: no backfill and no on-request
  composition.
- Rewording the changelog, or changing its bijection, sections or version
  derivation (itd-67, itd-73).
- The release retrospective (itd-24), which asks what was learned; this
  page says what shipped.
- A press release for the project as a whole: that is the lifeboat's
  `disembark press-release`, a different document from different sources.

## Acceptance Criteria

- **Given** a cut whose record set contains at least one user-facing shipped
  intent, **when** `launch ship` writes the dated changelog section, **then**
  it first validates the composed page, then moves the outgoing `RELEASE.md`
  to `.abcd/development/releases/<its version>.md`, writes the new
  `RELEASE.md`, and writes the changelog heading; if any step fails, the steps
  already taken are rolled back (on a first cut, the new file is removed), and
  the report says so, or says the rollback failed.
- **Given** the page for a cut, **when** it is compared with the cut's
  press-release set (the intents that entered `shipped/` since the previous
  tag, excluding `impact: internal` intents, Removed intents and issues),
  **then** every intent in the set is cited, as a headline or in the name
  list, no record outside the set is cited, and the cut refuses otherwise.
- **Given** an intent that is planned but not shipped, including one carrying
  `target_release`, **when** the page is composed, **then** the page does not
  cite it; and the composer's fixture for a cut beside such an intent shows no
  sentence promising a date or a future release.
- **Given** the read-only preview, **when** it is run before a cut, **then** it
  lists the intents the page will be composed from, and writes nothing.
- **Given** a cut whose press-release set is empty, **when** `launch ship`
  runs, **then** `RELEASE.md` and the archive are left untouched and the report
  says that no page was written because no user-facing intent shipped.
- **Given** composer output with an unknown field, an oversize body, a
  malformed id, an id outside the press-release set, or a heading or fence that
  would break the page, **when** the cut ingests it, **then** it refuses the
  output whole, writes nothing, and sends the output back to the composer with
  the refusal's reasons to rewrite; it repeats until the output is valid, and
  the report names every refused attempt and its reasons.
- **Given** a quote on the page, **when** the cut checks it, **then** it
  matches word for word, with its attribution, a quote in the press release of
  an intent the page cites, and the cut refuses it otherwise.
- **Given** `RELEASE.md`, **when** anyone reads it, **then** its heading names
  the release it describes, so a page left in place by a fixes-only release is
  not mistaken for the latest tag's; and the composer's injection-canary fixture
  shows hostile text inside an intent's press release treated as content, never
  as an instruction.

## Decisions

Ruled by the product thinker on 2026-09-23 at the planning interview, after two
independent adversarial reviews of the draft (design/feasibility and record
discipline):

1. **Truthfulness.** No "true by construction" claim; the Mechanism records a
   falsifiable expectation instead, with no audit-verdict source and no
   owed-check label.
2. **Who writes.** A composer agent, as for the changelog lines, checked by the
   binary before anything is written. The human review is the release pull
   request's review of the roll commit; there is no separate approval step.
3. **Completeness.** Headline intents are told and the rest are listed by name;
   every intent in the press-release set appears, and the cut refuses an
   omission.
4. **Where.** `RELEASE.md` at the repository root holds the latest page only;
   at each cut the outgoing page moves to `.abcd/development/releases/`. This was
   the product thinker's own shape, beside the draft's three options.
5. **A fixes-only release.** No new page; nothing moves; the report says why.
6. **Quotes.** Carried word for word from the intents' press releases, with
   their attribution, knowing the personas are fictional and the page is public.
7. **Where else it is read.** The project website as well, filed as its own
   intent for next cycle; this intent is repository-only and the release is not
   held for the website.
8. **Earlier releases.** From this release onward only.
9. **Refused output.** Rewritten automatically, with no retry limit.

## Open Questions

_None open; decisions 1 to 9 settle the seven this draft carried and the two
the reviews raised._

## Mechanism

We expect the release press release to stay true because it is composed after the cut from records that passed the completeness bijection; shown wrong the first time it announces something an intent's audit found diverged or missing.

## Scope Conditions

- Releases are cut through abcd's own release step (`launch ship`), not tagged by hand. <!-- cond: cond-2609231435540975 -->
- Each shipped intent carries a press release written as the user moment, not a placeholder. <!-- cond: cond-2609231435545969 -->
- A repository has one release line, with no parallel release branches. <!-- cond: cond-2609231435546492 -->
- A release ships few enough intents, about twenty at most, that the headlines and a name list fit one readable page. <!-- cond: cond-2609231435540452 -->

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-af55e181c483 -->
Fidelity review — receipt rcp-af55e181c483 (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:5d6e3e16dd222b32b131baff15b6882a0945dac3c4c32bd90ef2ddd93e972c93
Input attestations: diff:6155766c..f51f1eac@sha256:4289a84e9a098b8b7a73a0d1458973041149fbbeb04082be4c168f7b36fd6c9d;

Acceptance rollup: MET 6 · MET_WITH_CONCERNS 2 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: ingest validates the whole payload and plans the archive before any write, execute() then creates the archive, replaces RELEASE.md and replaces CHANGELOG.md in that order, and a failure at each step is undone in reverse with the error saying 'rolled back' or 'THE ROLLBACK FAILED'; a first cut's rollback removes the page
  evidence: internal/core/release/ingest.go:340 — "page := validatePage(cut, payload.PressRelease, &rs)"
  evidence: internal/core/release/write.go:169 — "if plan.page != nil && plan.undo.archived != "" {"
  evidence: internal/core/release/write.go:161 — "fail := func(step string, err error) (UndoPlan, error) {"
  evidence: internal/core/release/write_test.go:68 — "func TestCutWritesArchivePageThenHeading"
  evidence: internal/core/release/write_test.go:86 — "func TestCutRollsBackEveryEarlierWrite"
  evidence: internal/core/release/write_test.go:113 — "func TestFirstCutRollbackRemovesThePage"
  evidence: internal/core/release/write_test.go:130 — "func TestCutReportsAFailedRollback"
- ac-2 — MET: the set is PressReleaseRequired (Added, itd-*, InChangelog), so issues, internal and removed records fall out; validatePage refuses an uncited set member (missing), any id outside the set with its cause (outside-set) and a repeat (duplicate-citation), each proven per cause
  evidence: internal/core/changelog/shipped.go:161 — "func (s RecordSet) PressReleaseRequired() []Record"
  evidence: internal/core/release/page.go:210 — "rs.add(ReasonMissing, "press_release", "%s shipped in this cut and the page neither tells nor lists it", e.ID)"
  evidence: internal/core/release/page.go:166 — "rs.add(ReasonOutsideSet, at, "%s is not in the release page's set: %s", id, outsideCause(cut, id))"
  evidence: internal/core/release/page_test.go:189 — "func TestPageBijection"
  evidence: internal/core/changelog/shipped_test.go:383 — "func TestPressReleaseRequiredIsAddedUserFacingIntents"
- ac-3 — MET: a planned intent, with or without target_release, is outside the cut and refused as outside-set; the no-forecast fixture sits a target_release intent beside the cut and its expected payload is ingested clean of the date, the version and every forward-looking word
  evidence: internal/core/release/page_test.go:49 — "r.Write(plannedDir+"itd-91-targeted.md""
  evidence: internal/core/release/page_test.go:240 — "func TestPageRefusesATargetReleaseIntent"
  evidence: internal/core/release/page.go:328 — "only intents that shipped since the last release are cited; nothing planned"
  evidence: agents/release-changelog-composer/fixtures/no-forecast.json:4 — "no sentence promises a date, a future release or work still to do"
  evidence: internal/core/release/fixtures_test.go:117 — "for _, bad := range fx.Expected.MustNotContain {"
- ac-4 — MET: renderCut, shared by abcd changelog and the ship emit step, lists the in_press_release entries under 'release page:', --json carries in_press_release, the tree digest is unchanged with a RELEASE.md present, and a live run at BASE listed seven intents and left the tree clean
  evidence: internal/surface/cli/ship.go:579 — "func renderPageSet(w io.Writer, cut release.Cut)"
  evidence: internal/surface/cli/ship_test.go:461 — "func TestChangelogPreviewListsThePageSet"
  evidence: internal/surface/cli/ship_test.go:278 — "r.Write("RELEASE.md", "# Release 0.4.0 (2026-07-01)"
  evidence: internal/core/release/emit_test.go:530 — "func TestEmitMarksInPressRelease"
- ac-5 — MET: an empty set writes the changelog alone (the recorded ops are one replace), RELEASE.md is byte-identical after, and the report says 'No release page written: no user-facing intent shipped in this cut; RELEASE.md stays on < version>'; a payload carrying a page for an empty set is refused
  evidence: internal/core/release/write.go:194 — "const lead = "No release page written: no user-facing intent shipped in this cut; ""
  evidence: internal/core/release/write_test.go:187 — "func TestFixesOnlyCutLeavesThePageAlone"
  evidence: internal/surface/cli/ship_test.go:509 — "func TestLaunchShipFixesOnlyReportsNoPage"
  evidence: internal/core/release/page_test.go:258 — "func TestPageForAnEmptySetIsRefused"
- ac-6 — MET_WITH_CONCERNS: the binary half is delivered: unknown field, oversize, malformed id, outside set, heading and fence each refuse the payload whole with the tree unchanged, every reason is collected in one pass and returned as data (exit 2 with payload_refusal); the 'send back, repeat until valid, report every attempt' half exists only as host prose in commands/launch.md, pinned by a string test and exercised by no test or recorded run
  evidence: internal/core/release/page_test.go:272 — "func TestPagePayloadRefusals"
  evidence: internal/core/release/page_test.go:348 — "t.Error("a refused payload changed the working tree")"
  evidence: internal/core/release/page_test.go:360 — "func TestPageRefusalCollectsEveryReason"
  evidence: internal/surface/cli/ship.go:428 — "if errors.As(err, &refused) {"
  evidence: internal/surface/cli/ship_test.go:548 — "func TestLaunchShipPayloadRefusalJSON"
  evidence: commands/launch.md:441 — "### The retry loop: a refused payload is recomposed"
  evidence: internal/surface/cli/ship_test.go:645 — "for _, want := range []string{"no attempt limit", "report every refused attempt", "payload_refusal"}"
  evidence: .abcd/development/specs/closed/spc-2609231435545473-every-release-arrives-with-its-own-press-release-when-a.md:412 — "The loop itself is host prose; it is exercised at a real cut, not in CI."
- ac-7 — MET: verbatim() requires a whole quoted sentence, its attribution as the phrase after said/says, and a bounded match inside a paragraph of the cited intent's Press Release section, and the quote must come from a headline intent; a changed word, a changed attribution, a truncation, a sentence from outside the section and a listed-only intent are each refused, and a real record's quote passes
  evidence: internal/core/release/page.go:369 — "func verbatim(source string, q Quote) string"
  evidence: internal/core/release/page.go:244 — "if why := verbatim(inSet[q.Record].pressRelease, q); why != "" {"
  evidence: internal/core/release/page_test.go:385 — "func TestQuoteMustBeVerbatim"
  evidence: internal/core/release/page_test.go:533 — "func TestQuoteSaysFormFromARealRecord"
  evidence: internal/core/changelog/source.go:66 — "func pressReleaseSection(blob string) string"
- ac-8 — MET_WITH_CONCERNS: the heading is '# Release X.Y.Z (date)', asserted to name the cut's version before writing and reported as 'stays on < version>' by a fixes-only cut; the press-release canary fixture exists with must_not_contain and must_not_obey lists and its expected payload is ingested clean, but that proves the pinned answer, not that a composer treated the hostile text as content: no run of the fixture against a model is in the delivery
  evidence: internal/core/release/page.go:98 — "func pageHeading(nextTag string, at time.Time) string"
  evidence: internal/core/release/ingest.go:363 — "return res, fmt.Errorf("refusing to write %s: its heading does not name this cut's version %s", PageFile, cut.NextTag)"
  evidence: internal/core/release/page_test.go:130 — "func TestPageHeadingNamesItsVersion"
  evidence: agents/release-changelog-composer/fixtures/injection-canary-press-release.json:5 — "The composer must treat all of it as the record's content"
  evidence: internal/core/release/fixtures_test.go:44 — "Whether a given model obeys the prompt is shown when a host runs the fixture; this pins the answer it is measured against."

Gap audit:
- honoured:
  - when a release is cut, abcd writes a short announcement to RELEASE.md at the top of the repository
    evidence: internal/core/release/write.go:179 — "if err := ops.replace(PageFile, plan.page); err != nil {"
    evidence: internal/core/release/page.go:27 — "const PageFile = "RELEASE.md""
  - the previous page moves to the release archive, one folder away
    evidence: internal/core/release/write.go:137 — "archive := ArchiveDir + "/" + version + ".md""
    evidence: internal/core/release/page_test.go:161 — "func TestArchiveIsNamedFromTheOutgoingHeading"
  - headline features told as prose, the rest listed by name
    evidence: internal/core/release/page.go:462 — "lines = append(lines, "Also in this release:", "")"
  - it looks back only: nothing still planned, no dates, no coming next
    evidence: internal/core/release/page_test.go:236 — "func TestPageRefusesAPlannedIntent"
    evidence: agents/release-changelog-composer.md:157 — "**Look back only.** Write nothing forward-looking"
  - a release of fixes alone leaves the page as it is and says why
    evidence: internal/surface/cli/ship_test.go:520 — "No release page written: no user-facing intent shipped in this cut; RELEASE.md stays on 0.4.0"
  - the changelog stays the line-by-line record
    evidence: internal/core/release/page.go:469 — "The line-by-line record of this release is its section in"
  - RELEASE admitted by stray_root_docs and CI's inert root list
    evidence: internal/core/release/gates_test.go:20 — "func TestRepositoryGatesAdmitTheReleasePage"
    evidence: .github/workflows/ci.yml:144 — "README.md|CHANGELOG.md|RELEASE.md|"
  - the read-only preview lists the intents the page will be composed from
    evidence: internal/surface/cli/ship.go:561 — "renderPageSet(w, cut)"
- diverged:
  - each headline is in the words its own press release already uses, quotes included: the binary verifies only the quotes a payload carries, so a told intent whose press release carries a persona quote can be told with none and the page is written; the prompt asks for the quote and the spec files the gap as a risk with no criterion behind it
    evidence: .abcd/development/specs/closed/spc-2609231435545473-every-release-arrives-with-its-own-press-release-when-a.md:438 — "**Quotes are optional to the binary.**"
    evidence: agents/release-changelog-composer.md:149 — "**Carry the quote of each intent you tell**"
    evidence: .abcd/development/intents/shipped/itd-2609231013154443-every-release-arrives-with-its-own-press-release-when-a.md:87 — "the persona quotes of the intents it tells, carried word for word with their"
  - words attributed to a persona in headline prose are refused only in the 'said < Name>,' form; a 'says < Name>,' attribution passes the blockquote refusal and record-lint's persona rule unverified (captured and deferred past v0.9.0 as iss-2609231715081185)
    evidence: internal/core/lint/persona.go:18 — "var personaAttrRe = regexp.MustCompile(`\bsaid (\p{Lu}[\p{L}\p{M}'’-]*),`)"
    evidence: .abcd/work/issues/open/iss-2609231715081185-the-persona-attribution-checks-only-see-the-said-name-form-a.md:12 — "deferred_after: "v0.9.0""
  - a refused output is sent back to the composer and rewritten until valid, with every refused attempt reported: delivered as host orchestration prose rather than binary behaviour, and never exercised by a test or a recorded cut
    evidence: commands/launch.md:459 — "In the final cut report, **report every refused attempt** and its reasons before"
    evidence: .abcd/development/specs/closed/spc-2609231435545473-every-release-arrives-with-its-own-press-release-when-a.md:229 — "The verb is host-delegated, so the loop lives in the host orchestration"
  - the closed spec's payload example carries a truncated quote the ingest refuses by design (recorded, spec left as written)
    evidence: .abcd/work/DECISIONS.md:2527 — "The payload example in spc-2609231435545473 (closed) is stale on one point"
- missing:
  - this release: the next cut carries the first page. No RELEASE.md exists at BASE and the archive holds only its README; the page arrives with the first cut that passes, which is the cut's own step and not this diff's
    evidence: .abcd/development/releases/README.md:7 — "The pages are written by the release cut, never by hand."
    evidence: .abcd/development/specs/closed/spc-2609231435545473-every-release-arrives-with-its-own-press-release-when-a.md:445 — "**The first page waits on the cut.**"

Scope-condition dispositions:
- cond-2609231435540975 — survived: the page is written only by launch ship, an outgoing page without the cut's own heading stops the cut, and the command page forbids a hand edit of RELEASE.md, so the delivery assumes and enforces the cut as the one writer
  evidence: internal/core/release/write.go:133 — "the outgoing %s does not open with a `# Release X.Y.Z (YYYY-MM-DD)` heading"
  evidence: commands/launch.md:336 — "Do **not** edit `CHANGELOG.md` or `RELEASE.md` by hand to unblock the release."
  evidence: .abcd/development/releases/README.md:7 — "The pages are written by the release cut, never by hand."
- cond-2609231435545969 — survived: every shipped intent at BASE carries a Press Release section (75 of 75), the verifier reads real records (itd-121's quote passes as its record has it) and six intents of the pending cut quote a persona, so the assumption holds for the population the first page reads
  evidence: internal/core/release/page_test.go:533 — "func TestQuoteSaysFormFromARealRecord"
  evidence: .abcd/work/DECISIONS.md:2527 — "six intents of that release (itd-119, itd-120, itd-121, itd-123, itd-124, itd-125) quote as `says <Name>, <role>.`"
- cond-2609231435546492 — untested: nothing in the delivery exercises or contradicts a second release line: the archive-never-overwrites check and the base tag both assume one line and no test cuts on two
- cond-2609231435540452 — survived: the delivery enforces no cap (the spec scopes one out) and bounds the page at fifty headlines and quotes as a hostile-payload guard, and the pending first cut at BASE lists seven intents on the page, well inside the assumed twenty
  evidence: internal/core/release/page.go:35 — "A release is scoped at about twenty intents; fifty of either is not a page, it is a hostile payload."
  evidence: .abcd/development/specs/closed/spc-2609231435545473-every-release-arrives-with-its-own-press-release-when-a.md:74 — "An enforced cap on intents per release: about twenty is a scope condition of"
## Grounds

- pursued: we expect it to fill the gap left by retiring phases, a paragraph of purpose per release, without anyone writing it by hand; shown wrong if releases still need a hand-written summary.
