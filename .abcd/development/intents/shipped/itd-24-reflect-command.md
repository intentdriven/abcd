---
id: itd-24
slug: reflect-command
spec_id: spc-2609211751376504
kind: bundle-member
bundle: spc-83-operator-surfaces
suggested_kind: null
reclassification_history: []
glossary_terms_used: [core/phase, core/intent, core/voyage, core/persona, core/brief, core/lifeboat, core/oracle, core/spec, interview/embark, distribution/release]
grill_session_id: e6a24d86-e133-495f-8dec-94dec21449ea
grilled_at: 2026-05-16T15:37:17Z
grilled_intent_hash: 8412a59b575df882fc4a370ab01404796cad4dd9e120d0519e9918d3ea891c61
prd_path: null
prd_grandfathered: true
severity: minor
builds_on: []
impact: additive
---

# Completed Releases Get A Retrospective

## Press Release

> **abcd ships `/abcd:reflect` for release retrospectives.** Run `/abcd:reflect v0.11.0` and abcd walks an interview-driven retrospective: what went well, what could improve, lessons learned, decisions made, metrics. The interview is *seeded* by what the release shipped — the intents the tag shipped, read the way the release cut reads them, each with the audit notes the intent auditor wrote on it, and the changelog section the cut composed — so the conversation opens from what actually passed and failed. Output is a structured `.abcd/development/retrospectives/v0.11.0/README.md`, committed as part of the permanent record. Future lifeboats carry the retrospective forward; future intents reference past lessons. Reflection becomes a first-class abcd primitive, not an afterthought.
>
> "abcd's brief and intents captured *what* I'd done," said Henry, a junior-developer persona. "Reflect captures *what I learned* — and because it starts from what the release shipped and how each audit went, it doesn't ask me to re-remember the work, it asks me what the verdicts *mean*. When I started a new voyage six months later, embark surfaced past retrospectives in the lifeboat unpack — the lessons came with the work. I didn't re-make the mistakes."

## Why This Matters

abcd ships strong post-implementation transparency: shipped intents have audit notes (per `itd-1` acceptance criteria); the native spec store's completion records capture what was built; and each release cut composes a changelog from the records that shipped in it. What's missing is **post-release reflection** — the structured "what did we learn" doc that's bigger than per-intent audit notes, bigger than a pass/fail audit verdict, and smaller than a brief rewrite.

The legacy `~/.claude/templates/retrospective.md.template` had the right prompt structure (what went well, what could improve, lessons learned, decisions made, metrics) but lived as a manual template that rarely got used. It was deferred (see [`research/legacy-harvest.md`](../../research/legacy-harvest.md) Pass 4 retrospective decision); `/abcd:reflect` promotes it to a first-class command with structured interview + structured output.

`/abcd:reflect` is the *release-level* reflection surface — broader than per-intent audit notes (which the intent auditor already produces on ship per the itd-1 discipline) and narrower than a brief rewrite. It spans the intents a release shipped and captures what's transferable to future voyages. It was first written at the grain of a roadmap phase (adr-9); phases are retired (adr-2609212115255771), and the release is the unit that replaced them (decisions 4 and 5).

An audit and a retrospective are **distinct activities**. The audit asks *did each intent deliver its criteria* (a per-criterion verdict). The retrospective asks *what did we learn* (transferable insight, interview-driven). `/abcd:reflect` does not replace the audit — it **consumes** it: the release's audit verdicts are the seed material the retrospective interview opens from.

## Mechanism

We expect the value of a retrospective to be in the lifeboat: a new project that starts from an old one's lessons avoids a repeat, because the lessons arrive with the work rather than being re-derived; shown wrong if projects embarked from a lifeboat carrying retrospectives never open a surfaced lesson.

## What's In Scope

- **`/abcd:reflect <release-tag>`** — retrospective for a cut release, and the command's *only* argument form. Examples: `/abcd:reflect v0.10.0`, `/abcd:reflect v0.11.1`. Per-intent reflection is out of scope (see below) — `/abcd:reflect` operates at the release grain only.
- **Seeded from the release** (decision 5). The seed is the intents a tag shipped, read the way the release cut reads them (ruling AD): those that reached `shipped/` between the previous release tag and this one, less any whose `shipped_in` names another release, plus any whose `shipped_in` names this one. For each, its `## Audit Notes` (per-criterion verdicts, honoured / diverged / missing) and its `impact`; and the changelog section the cut composed. The interview opens from them rather than from a blank prompt.
- **Audit-missing handling.** A shipped intent in the release with no audit notes is named, and the command offers `abcd intent audit <itd-N>` for it first; the interview continues either way.
- **Empty-release refusal.** A tag whose release shipped no intent refuses — there is no delivered work to reflect on.
- **Interview-driven structure**:
  - What went well (successes and strengths, with specific examples)
  - What could improve (issues and gaps, ranked)
  - Lessons learned (transferable insights, framed for future-you)
  - Decisions made (architectural / design choices crystallised during the release)
  - Metrics (intents shipped, audit-note severity distribution, time-to-ship if measurable)
- **Output**: `.abcd/development/retrospectives/<release-tag>/README.md` — a peer of `.abcd/development/intents/`, committed as part of the permanent record.
- **Lifeboat integration**: `/abcd:disembark pack <repo> <path>` packs *all* of the voyage's retrospectives into the lifeboat — the full reflection arc travels. `/abcd:embark from <path>` surfaces predecessor retrospectives during the press-release interview ("here's what the previous voyage learned about X — does that apply here?").
- **Reference back to intents and the audit**: the retrospective links to the release's changelog section, to the intents it shipped, and to their audit notes; the audit notes are referenced, not duplicated.
- **`reflection-composer` agent** — runs the interview, drafts the structured output, asks clarifying questions when answers feel thin.

## What's Out of Scope

- **The audit itself** — judging delivered reality against each intent's criteria is the intent auditor's job. `/abcd:reflect` consumes those verdicts; it does not produce them.
- **Per-intent reflection** — the intent auditor already produces per-criterion verdicts and a three-bucket prose audit on every shipped intent (per the itd-1 discipline). That *is* per-intent reflection; `/abcd:reflect` does not duplicate it. There is no `/abcd:reflect <itd-N>` form — the command takes a release tag only.
- **A date or tag range** as the seed, and **phase documents** as the seed: the alternatives decision 5 did not take.
- **Quantitative retrospective metrics** (DORA, velocity, etc.) — abcd doesn't gather the underlying telemetry. Metrics section is qualitative + simple counts only.
- **Team retrospectives** — abcd is single-developer-shaped (or pair-shaped); team retrospective patterns belong elsewhere.
- **Automatic triggering** — reflection requires the persona's deliberate engagement; beyond the one-line nudge of decision 1, nothing prompts for it.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** an abcd repo with a cut release (`v0.11.0` is tagged and at least one intent reached `shipped/` in it, as the release cut reads membership under ruling AD), **when** the persona runs `/abcd:reflect v0.11.0`, **then** the reflection-composer agent runs an interview *seeded by that release's shipped intents and their audit notes* and writes `.abcd/development/retrospectives/v0.11.0/README.md` with all five required sections populated.
- **Given** a release one of whose shipped intents carries no audit notes, **when** the persona runs `/abcd:reflect <release-tag>`, **then** the command names that intent and offers `abcd intent audit <itd-N>` before continuing into the retrospective.
- **Given** a release tag that shipped no intent, **when** the persona runs `/abcd:reflect <release-tag>`, **then** the command refuses with "no intent shipped in `<release-tag>` — nothing shipped to reflect on" and writes no output.
- **Given** a draft retrospective with thin answers (e.g. "what went well: it worked"), **when** the agent drafts the output, **then** the agent surfaces the thinness as a clarifying question rather than committing the thin answer.
- **Given** the same repo's lifeboat is then packed via `/abcd:disembark pack <repo> <path>`, **when** the lifeboat is inspected, **then** every `.abcd/development/retrospectives/<release-tag>/README.md` the voyage produced is included in the lifeboat artefact.
- **Given** a target repo embarked from a lifeboat that includes retrospectives, **when** `/abcd:embark from <path>` runs the press-release interview, **then** the persona is shown the few predecessor lessons ranked most like the new voyage's brief, with the rest as a list, and asked which apply.
- **Given** a release with intents still unshipped whose `target_release` names it, **when** `/abcd:reflect <release-tag>` runs, **then** the command warns the persona, lists those intents, and asks for confirmation to proceed anyway.
- **Given** a release cut is written, **when** that cut completes, **then** abcd says once that a retrospective for the release is owed and names the command, and says nothing further about it.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that gave this intent its spec:

1. **Nudge once.** When a phase's last piece of work closes, abcd says once that a retrospective is owed; it is not repeated and it is not a gate.
2. **A ranked few on embark.** Predecessor lessons most like the new voyage's brief are shown; the rest are a list opened on request.
3. **Layout.** The retrospective lives under the durable record tier, `.abcd/development/retrospectives/<phase-id>/README.md`; the paths this record was written against predate the three-tier layout and are read as that.
4. **The unit is the release** (ruled 2026-09-21, adr-2609212115255771): phases are retired, so `<phase-id>` reads as the release tag (`v0.10.0`), the seed is the release's shipped intents and their audit notes with the derived changelog, the empty case is a release that shipped no intent, the nudge fires once when the cut is written, and criterion 7's warning names intents targeted at the release (`target_release`) still unshipped. Every criterion below is read with "phase" meaning "release".
5. **Seed from a release** (ruled by the product thinker on 2026-09-29, ruling AD of autonomous run A's owed list): a retrospective starts from the intents a tag shipped. Seeding from a date or tag range the person names, and keeping phase documents as the anchor, were the alternatives not taken. The press release, scope and criteria were rewritten to the release on 2026-09-29, so decision 4's reading rule is now the text itself; the Blocking Dependency and v1 notes below record the phase-grain design this replaces.

## Open Questions

_None open; decisions 1 and 2 settle the two this record carried (the reflection cadence and the lifeboat surfacing on embark)._

## Blocking Dependency

_Superseded by decisions 4 and 5: the seed is the per-intent audit notes, which exist, so no phase-fidelity output is awaited. The text below is the phase-grain history._

`/abcd:reflect` **cannot be planned until the phase-fidelity-reviewer ships** and its output artefact is stable and machine-readable. The reviewer is deferred in adr-9. Because `/abcd:reflect`'s core design is to *consume* the audit's per-bullet verdicts as interview seed material, and the audit-missing AC depends on running the reviewer inline, the command has no buildable contract until the reviewer's output format exists. `/abcd:intent plan itd-24` must not proceed while the phase-fidelity-reviewer remains unbuilt.

**Satisfied:** the stable machine-readable phase-fidelity output is provided by spc-66 (predecessor store) (`phase_review_report.schema.json` + `.abcd/logbook/audit/phase-<ts>/report.{json,md}`), so this dependency is met and `/abcd:reflect` is planned under spc-83. V1 keys empty-phase detection off the spc-66 (predecessor store) receipts (not a `phase:` anchor) and refuses on a missing/empty-audited receipt rather than offering the inline reviewer — see the `### Implementation notes (spc-83.3 — v1 scope)` block below.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-2215987d5eb7 -->
Fidelity review — receipt rcp-2215987d5eb7 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:c908ff4a28ec3e93638b483a19765fd18fca15012986840f45674a7aa7f6e489
Input attestations: repository:4f05055572ada95ccedb28a2c49418f4d5d3b6f5@-; diff:c8ddafcaa508868a0685adf0f0e4fa26bfd82dd5^..63c9969e39aee422aed67be8f6e3d7f9f9356857@-; intent:.abcd/development/intents/shipped/itd-24-reflect-command.md@sha256:43c33152de401e4f8b470678484097aeab694c55c4c789bf208594d8cb6011e8; spec:.abcd/development/specs/closed/spc-2609211751376504-reflect-command.md@sha256:a8834c5bae6bf8e0b54509a2757556e66ba827c144889dc4db1cd918ed575699; request:.abcd/.work.local/reviews/rcp-2215987d5eb7.request.md@sha256:7b36a525f63a5ae7820d4566aba12ba8776448dfb96948a603fc21aa32de0ac0;

Acceptance rollup: MET 4 · MET_WITH_CONCERNS 4 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: BuildSeed reads the intents the tag shipped the way the cut reads membership, with each one's impact, audit counts and the changelog section. The host page dispatches the reflection-composer with that seed, and the writer renders the four asked sections plus computed metrics; the test asserts all five. Concerns: since the placement of 2026-10-09 the /abcd:reflect page is hidden from the person's menu, so the persona reaches it through an agent. The only real retrospective (v0.12.0) was written by the old audit reader, still carries its miscount, and cannot be regenerated because a second run on a tag refuses.
  evidence: internal/core/reflect/seed.go:131 — "func BuildSeed(root, tag string) (Seed, error) {"
  evidence: commands/reflect.md:55 — "Dispatch the `reflection-composer` agent with the seed."
  evidence: internal/core/reflect/sections.go:17 — "var AskedSections = []Section{WentWell, CouldImprove, Lessons, Decisions}"
  evidence: internal/core/reflect/write.go:233 — "sec.Heading(), strings.TrimSpace(ans.Text))"
  evidence: internal/core/reflect/write.go:241 — "- Intents shipped: %d (%d with audit notes, %d without)"
  evidence: internal/core/reflect/write_test.go:29 — "func TestWriteProducesTheRetrospectiveWithFiveSections(t *testing.T) {"
  evidence: commands/reflect.md:6 — "user-invocable: false"
  evidence: .abcd/development/brief/04-surfaces/09-reflect.md:41 — "hidden from a person's `/abcd:` menu (the product thinker's placement of"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:56 — "- Intents shipped: 11 (11 with audit notes, 0 without)"
- ac-2 — MET_WITH_CONCERNS: The prior failure is fixed. The review marker is read first: an OWED review is never audit notes, and a section with no marker counts only if it names criteria with verdicts (HandAudited, the owed listing's own rule). Each unaudited intent is offered 'abcd intent audit < itd-N>', and the host page names each one before the interview. Tests cover an owed review beside an amendment note and a stray note. Concerns: (a) a DEAD_LETTER review's quarantine line is not skipped, so it counts as audit notes, although owed.go calls a dead-lettered review unreviewed; (b) the plain-Terminal 'abcd reflect interview' still names no unaudited intent and offers no audit before the first question; (c) the v0.12.0 record still says 'unaudited: []'.
  evidence: internal/core/reflect/audit.go:44 — "case intent.ReviewOwed:"
  evidence: internal/core/reflect/audit.go:47 — "it.Audited = intent.HandAudited(text)"
  evidence: internal/core/reflect/seed.go:172 — "s.Unaudited = append(s.Unaudited, AuditOffer{IntentID: it.ID, Command: "abcd intent audit " + it.ID})"
  evidence: commands/reflect.md:44 — "- **Intents without audit notes** (`unaudited[]`): name each one to the product thinker"
  evidence: internal/surface/cli/reflect.go:351 — "no audit notes on %d intent(s); offer the audit first, then continue either way:"
  evidence: internal/core/reflect/seed_owed_test.go:37 — "func TestSeedOffersTheAuditToAnOwedReviewBesideAnAmendmentNote(t *testing.T) {"
  evidence: internal/core/reflect/seed_owed_test.go:65 — "func TestSeedJudgesAnUnmarkedSectionByTheOwedListingsRule(t *testing.T) {"
  evidence: internal/core/reflect/audit.go:71 — "strings.HasPrefix(t, "Fidelity review OWED")"
  evidence: internal/core/reflect/audit.go:99 — "if prose {"
  evidence: internal/core/intent/audit.go:1877 — ""Fidelity review DEAD_LETTER (receipt %s): %s. Raw payload retained at %s. "+"
  evidence: internal/core/intent/owed.go:61 — "dead-lettered review is unreviewed but not owed"
  evidence: internal/surface/cli/reflect.go:205 — "w := &interview.Written{"
  evidence: internal/core/interview/tasks.go:12 — "(an intent with a NOT_MET verdict, an intent shipped with no audit, an intent that missed the release)"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:7 — "unaudited: []"
- ac-3 — MET: NothingShippedError's message is the criterion's exact wording. BuildSeed returns it before anything is created, Write rebuilds the seed and so refuses the same way, and the CLI and core tests assert the exact text and that nothing is written.
  evidence: internal/core/reflect/reflect.go:67 — "return fmt.Sprintf("no intent shipped in `%s` — nothing shipped to reflect on", e.Tag)"
  evidence: internal/core/reflect/seed.go:158 — "return Seed{}, &NothingShippedError{Tag: tag}"
  evidence: internal/surface/cli/reflect_cli_test.go:148 — "func TestReflectRefusesAReleaseThatShippedNothing(t *testing.T) {"
  evidence: internal/surface/cli/reflect_cli_test.go:158 — "no intent shipped in `v0.3.0` — nothing shipped to reflect on"
  evidence: internal/core/reflect/write_test.go:140 — "func TestWriteRefusesAReleaseThatShippedNoIntentAndWritesNothing(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: underFloor marks 'it worked' as thin. Write refuses, writing nothing, until every thin section has a follow-up, and returns that section's question for the reflection-composer to ask. Concerns: the thin original is still written verbatim above the follow-up reply (v0.12.0 'What went well' opens 'How we worked.'), and the reply is not held to the floor. So a thin answer can still be committed after one question, while the agent page says a thin answer is not committed.
  evidence: internal/core/reflect/thin.go:33 — "func underFloor(s Section, text string) (bool, string) {"
  evidence: internal/core/reflect/write.go:105 — "if !isThin || strings.TrimSpace(a.FollowUp) != "" {"
  evidence: internal/surface/cli/reflect_cli_test.go:276 — "func TestReflectWriteMeetsAThinAnswerWithAQuestion(t *testing.T) {"
  evidence: internal/core/reflect/write.go:234 — "if f := strings.TrimSpace(ans.FollowUp); f != "" {"
  evidence: internal/core/reflect/write_test.go:109 — "func TestWriteTakesAThinAnswerOnceItsFollowUpIsAnswered(t *testing.T) {"
  evidence: agents/reflection-composer.md:82 — "A thin answer is **not** committed."
  evidence: agents/reflection-composer.md:92 — "if the reply is still thin, say so to the person and let them decide"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:31 — "How we worked."
- ac-5 — MET: The lifeboat plan copies every < release-tag>/README.md under the retrospectives store into the lifeboat as a record family sealed by the manifest. The core and CLI tests pack two retrospectives verbatim and write them back on embark.
  evidence: internal/core/lifeboat/plan.go:288 — "if ctx.IsDir(corereflect.RetrospectivesRelDir) {"
  evidence: internal/core/lifeboat/plan.go:299 — "pb.copyRecord(ctx, src, path.Join(retrospectivesLifeboatDir, tag, retrospectiveLeaf))"
  evidence: internal/core/lifeboat/retrospectives_test.go:50 — "func TestPlanPacksEveryRetrospective(t *testing.T) {"
  evidence: internal/surface/cli/reflect_lifeboat_cli_test.go:45 — "func TestDisembarkPackCarriesEveryRetrospectiveAndEmbarkWritesThemBack(t *testing.T) {"
  evidence: internal/core/lifeboat/retrospectives_test.go:102 — "func TestEmbarkFromWritesRetrospectivesBack(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: Main today: 'embark lessons' verifies the manifest and ranks every lesson by deterministic lexical term overlap (record/match) against the target's framing chapter or a --brief file. It shows the top three, then the rest as a list, with 'which of these apply?', and the embark page has the host ask that one question. This is the term-overlap method the spec names. Concerns: (a) the ranking is a lexical heuristic; the judgement-based ranking through the decision adapter is planned in itd-2609221009495079 and not on main; (b) 'embark from' runs no press-release interview (the matrix marks it staged), so the lessons are a separate step; (c) with no framing chapter and no --brief the lessons come back unranked but are still headed 'most like this voyage'.
  evidence: internal/surface/cli/cli.go:1479 — "lessonsCmd := &cobra.Command{"
  evidence: internal/surface/cli/cli.go:1515 — "const embarkFramingRel = ".abcd/development/brief/01-product/06-framing.md""
  evidence: internal/core/reflect/lessons.go:15 — "const TopLessons = 3"
  evidence: internal/core/reflect/lessons.go:100 — "fwd, _, shared := match.Overlap(terms[i], framingTerms, w)"
  evidence: internal/core/lifeboat/retrospectives.go:54 — "func PredecessorLessons(lifeboatDir, framing string) (LessonsView, error) {"
  evidence: internal/surface/cli/reflect_lifeboat_cli_test.go:77 — "func TestEmbarkLessonsShowsTheFewMostLikeTheBrief(t *testing.T) {"
  evidence: commands/embark.md:83 — "Show the product thinker the `top` few (three), each with the release it came from, and"
  evidence: .abcd/development/specs/closed/spc-2609211751376504-reflect-command.md:60 — "whole; `embark` ranks the packed retrospectives' lessons against the new"
  evidence: .abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:43 — "embark's lesson ranking"
  evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:62 — "| Embark press-release interview | **(staged)**"
  evidence: commands/embark.md:59 — "`from` runs the same planner as `probe`, then writes each `create` file into the"
  evidence: internal/core/lifeboat/retrospectives.go:122 — "strings.TrimSpace(framing) == """
  evidence: internal/core/lifeboat/retrospectives.go:151 — "most like this voyage — which of these apply?"
- ac-7 — MET: The prior failure is fixed. unshippedTargets now lists the planned intents whose target_release names the tag, plus the still-planned intents that the release's own changelog section names in the cut's target-move note. The note's lead is spelled once for its writer and this reader, so targets launch ship rewrote to 'next' before tagging are found again. The CLI prints a WARNING listing them, the host page asks, Write and the plain-Terminal interview refuse without --proceed, and a test cuts a release with moved targets and checks only the still-unshipped one is listed.
  evidence: internal/core/reflect/seed.go:200 — "func unshippedTargets(corpus intent.Corpus, tag string, cl ChangelogSection) []TargetedIntent {"
  evidence: internal/core/reflect/seed.go:216 — "if it.Bucket == intent.BucketPlanned && it.TargetRelease == tag {"
  evidence: internal/core/reflect/seed.go:221 — "if !changelog.IsTargetMoveNote(ln) {"
  evidence: internal/core/changelog/targetmove.go:22 — "const targetMoveLead = "Targeted and not shipped in this release, so each targets the next release (`next`): ""
  evidence: internal/surface/cli/reflect.go:357 — "WARNING: %d intent(s) targeted at %s have not shipped; ask before proceeding (reflect write --proceed):"
  evidence: commands/reflect.md:49 — "(`unshipped_targets[]`): warn the product thinker, list them, and ask whether to"
  evidence: internal/core/reflect/write.go:98 — "if len(seed.Unshipped) > 0 && !req.ProceedDespiteUnshipped {"
  evidence: internal/surface/cli/reflect.go:192 — "if len(seed.Unshipped) > 0 && !proceed {"
  evidence: internal/core/reflect/seed_owed_test.go:109 — "func TestSeedListsTheTargetsTheCutMovedPastTheRelease(t *testing.T) {"
  evidence: internal/core/reflect/write_test.go:153 — "func TestWriteAsksBeforeProceedingPastUnshippedTargets(t *testing.T) {"
- ac-8 — MET: launch ship sets retrospective_owed to reflect.Nudge(tag) only when the cut is written and prints it as the run's last line. The CLI is the only non-test caller of release.Ingest, and no other non-test surface mentions an owed retrospective. The test asserts one mention, at the end, and none on an emit that writes nothing.
  evidence: internal/core/reflect/nudge.go:10 — "return fmt.Sprintf("A retrospective for %s is owed: run /abcd:reflect %s when you are ready.", tag, tag)"
  evidence: internal/surface/cli/ship.go:677 — "res.RetrospectiveOwed = reflect.Nudge(ingested.Cut.NextTag)"
  evidence: internal/surface/cli/ship.go:48 — "return release.Ingest(repoRoot, current, raw, at)"
  evidence: internal/surface/cli/reflect_cli_test.go:363 — "func TestLaunchShipSaysOnceThatARetrospectiveIsOwed(t *testing.T) {"
  evidence: internal/surface/cli/reflect_cli_test.go:371 — "if n := strings.Count(string(out), "retrospective"); n != 1 {"

Gap audit:
- honoured:
  - The interview opens from what the release shipped: the intents the tag shipped, read the way the cut reads membership, each with impact and audit counts, plus the changelog section the cut composed.
    evidence: internal/core/reflect/seed.go:319 — "func shippedBy(root, tag, prev string, corpus intent.Corpus) ([]member, error) {"
    evidence: internal/core/reflect/seed_test.go:22 — "func TestSeedIsTheIntentsTheTagShipped(t *testing.T) {"
  - A shipped intent whose review is owed is named and offered the audit, by the same rule the list of reviews owed uses.
    evidence: internal/core/reflect/audit.go:44 — "case intent.ReviewOwed:"
    evidence: internal/core/reflect/seed_owed_test.go:65 — "func TestSeedJudgesAnUnmarkedSectionByTheOwedListingsRule(t *testing.T) {"
  - Intents targeted at the release that did not ship are warned about and confirmed past, including those a cut moved to next.
    evidence: internal/core/reflect/seed.go:221 — "if !changelog.IsTargetMoveNote(ln) {"
    evidence: internal/core/reflect/seed_owed_test.go:109 — "func TestSeedListsTheTargetsTheCutMovedPastTheRelease(t *testing.T) {"
  - The output is a five-section README in the durable record tier that links to the changelog section and each intent's audit notes rather than copying them; metrics are computed, never asked.
    evidence: internal/core/reflect/write.go:216 — "b.WriteString(" Each intent's audit notes stay on the intent and are linked here, not repeated.\n\n")"
    evidence: internal/core/reflect/sections.go:17 — "var AskedSections = []Section{WentWell, CouldImprove, Lessons, Decisions}"
  - An empty release is refused in the criterion's words, and the release is the only grain (an intent id is refused).
    evidence: internal/core/reflect/reflect.go:67 — "return fmt.Sprintf("no intent shipped in `%s` — nothing shipped to reflect on", e.Tag)"
    evidence: internal/surface/cli/reflect.go:271 — ""abcd reflect: %s is an intent; reflect takes a release tag (such as v0.11.0). Per-intent reflection is the intent audit: abcd intent audit %s","
  - Future lifeboats carry every retrospective forward, and embark writes them back.
    evidence: internal/core/lifeboat/plan.go:299 — "pb.copyRecord(ctx, src, path.Join(retrospectivesLifeboatDir, tag, retrospectiveLeaf))"
    evidence: internal/core/lifeboat/retrospectives_test.go:102 — "func TestEmbarkFromWritesRetrospectivesBack(t *testing.T) {"
  - Reflection has a single nudge at the written cut, and nothing repeats it or gates on it.
    evidence: internal/surface/cli/ship.go:677 — "res.RetrospectiveOwed = reflect.Nudge(ingested.Cut.NextTag)"
    evidence: internal/surface/cli/reflect_cli_test.go:363 — "func TestLaunchShipSaysOnceThatARetrospectiveIsOwed(t *testing.T) {"
- diverged:
  - Clarifying questions for thin answers add to the thin answer instead of replacing it: the thin original is committed beside the follow-up, and the reply is not held to the floor.
    evidence: internal/core/reflect/write.go:234 — "if f := strings.TrimSpace(ans.FollowUp); f != "" {"
    evidence: .abcd/development/retrospectives/v0.12.0/README.md:31 — "How we worked."
  - Embark surfaces predecessor lessons during the press-release interview: delivered as a separate 'embark lessons' step ranked by lexical term overlap. The interview embark would run is staged, the judgement-based ranking awaits the decision adapter, and unranked lessons are still headed 'most like this voyage'.
    evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:62 — "| Embark press-release interview | **(staged)**"
    evidence: .abcd/development/intents/planned/itd-2609221009495079-abcd-s-typed-judgements-can-run-through.md:43 — "embark's lesson ranking"
    evidence: internal/core/lifeboat/retrospectives.go:151 — "most like this voyage — which of these apply?"
  - Reflection is a first-class command the persona runs as /abcd:reflect: the page is hidden from the person's menu, while the cut's nudge still tells the person to run /abcd:reflect < tag>.
    evidence: commands/reflect.md:6 — "user-invocable: false"
    evidence: internal/core/reflect/nudge.go:10 — "return fmt.Sprintf("A retrospective for %s is owed: run /abcd:reflect %s when you are ready.", tag, tag)"
  - Every shipped intent without audit notes is named and offered the audit: a DEAD_LETTER quarantine line is counted as audit notes, and the plain-Terminal interview makes no offer before its first question.
    evidence: internal/core/reflect/audit.go:71 — "strings.HasPrefix(t, "Fidelity review OWED")"
    evidence: internal/core/intent/owed.go:61 — "dead-lettered review is unreviewed but not owed"
    evidence: internal/core/interview/tasks.go:12 — "(an intent with a NOT_MET verdict, an intent shipped with no audit, an intent that missed the release)"
  - The one retrospective in the record (v0.12.0) was written under the old audit reader and still overstates its audits; a second run on a tag refuses, so it is not regenerated.
    evidence: .abcd/development/retrospectives/v0.12.0/README.md:7 — "unaudited: []"
    evidence: .abcd/development/retrospectives/v0.12.0/README.md:56 — "- Intents shipped: 11 (11 with audit notes, 0 without)"
- missing:
  - Future intents reference past lessons: nothing on the intent side reads the retrospective store; lessons are read only by embark's lifeboat view.
    evidence: internal/core/reflect/lessons.go:28 — "func ReadLessons(readme []byte) []Lesson {"
    evidence: internal/core/lifeboat/retrospectives.go:107 — "for _, l := range corereflect.ReadLessons(data) {"
<!-- abcd-review-end receipt=rcp-2215987d5eb7 -->

Audit owed flag cleared 2026-10-29 (receipt rcp-2215987d5eb7): a re-run of the audit judged no criterion NOT_MET or INCONCLUSIVE, and iss-2610241431221889 is resolved.

- 2026-09-30: the press release, scope item 1 and criterion 1 read release membership the way the release cut reads it, under ruling AD, as spec scope 1 does since its amendment of the same date; `shipped_in` moves a record between releases and does not define membership.

### Implementation notes (v1 scope)

_Phase-grain history of a thin V1 that is not in the tree; superseded by decision 5, which seeds from a release._

`/abcd:reflect` (thin V1) refines two acceptance behaviours from
their originally-drafted form; both are recorded here so the fidelity review
reads them as deliberate v1 scope, not gaps:

- **Missing-audit inline-reviewer offer → refusal (deferred).** The draft AC
  had the command "offer to run the phase-fidelity-reviewer inline" when no
  audit exists. V1 instead **refuses** when no spc-66 (predecessor store) phase-audit receipt exists
  for the named phase (and when the latest matching receipt is empty-audited).
  Empty-phase detection keys off the spc-66 (predecessor store) receipts, not a `phase:` spec anchor
  (that anchor is deferred; phase membership is editorial). Running the reviewer
  inline from reflect is a recorded future extension.
- **Open-spec warn/confirm (deferred).** The draft AC had reflect warn and ask
  for confirmation when a phase's specs are not all closed. V1 does not parse
  editorial Scope membership, so this warn/confirm path is deferred; the refusal
  semantics above are the v1 gate.
- **Phase-only grain, source links, lifeboat.** `/abcd:reflect <itd-N>` is
  refused (phase-only grain). V1 links to the phase doc + audit report + member
  specs only (no intent links — the spc-66 (predecessor store) receipt carries no intent ids;
  recorded future extension). The lifeboat-packs-all-retrospectives requirement
  is a DOCUMENTED forward requirement on the future disembark spec (spc-17 (predecessor store) stubs),
  not a behaviour this surface implements — recorded in the surface doc.

The interview is a single seeded pass (per-bullet verdicts → five questions);
multi-turn depth is a recorded future extension. Full surface record:
[`../../brief/04-surfaces/09-reflect.md`](../../brief/04-surfaces/09-reflect.md).

### Linkage note (2026-09-30)

`builds_on` named itd-27 (the grill sub-verb), which is superseded by itd-94.
The edge is dropped rather than relinked: itd-94 carries the grill forward only
as the planning interview behind the implement-readiness gate, and a
retrospective seeded from a release's shipped intents needs neither that
interview nor that gate. The retrospective interview is its own, so nothing
itd-94 delivers is an input to this record.

### Linkage note (spc-83.5)

Ships as one of FOUR intents sharing spec
`spc-83-operator-surfaces-manifest-lockstep`. abcd represents "N intents, one
spec" as a bundle (`kind: bundle-member` + shared `bundle: spc-83-operator-surfaces`)
— the representation the doc_fidelity intent-resolution + spec-close preflight
require. Bundle member by delivery relationship, not a scope change. This intent
keeps its real grill linkage (`grill_session_id`); GR002 is handled via
`prd_grandfathered`. Full record in the spec's process-exception note.

## Grounds

- pursued: nine phases have closed with their lessons living only in session handovers; we expect the first retrospectives to hold what the handovers do not, and a new voyage to open a carried lesson; shown wrong if the first retrospectives say nothing the handovers did not, or if no embarked project opens one
