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
Input attestations: diff:5346ff97bdf7b0b69d89e9147cf3cc781bb0a7c0..62cb45d843d93c1fd9ae99aea1ff7e71a1753ca2@sha256:2a71ea3be77b21b8963580770b5a04b677435966bf065c442b3048b94f54fcba; merge:0ebf2fe492279aadb875b1ec698c564d3f609078@-; repository:2e03c61e454bf09ecda8cee0ae2f377157b522f7@-; intent:.abcd/development/intents/shipped/itd-24-reflect-command.md@sha256:47e96f2f08b7c13849617e7585a1da1e5229c27cd11df6aeee1042a14ffb0072; spec:.abcd/development/specs/closed/spc-2609211751376504-reflect-command.md@sha256:a8834c5bae6bf8e0b54509a2757556e66ba827c144889dc4db1cd918ed575699; retrospective:.abcd/development/retrospectives/v0.12.0/README.md@sha256:65af7cf68b9a2a0bd704f63d406f1e0c62d5a15e0d25bc12ba9449ab4de614a7; request:.abcd/.work.local/reviews/rcp-2215987d5eb7.request.md@sha256:7b36a525f63a5ae7820d4566aba12ba8776448dfb96948a603fc21aa32de0ac0;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 3 · NOT_MET 2 · INCONCLUSIVE 0

Audit owed (receipt rcp-2215987d5eb7): ac-2 NOT_MET · ac-7 NOT_MET. Remedy: fix, then re-run the audit. Carried by iss-2610241431221889.

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: BuildSeed reads the intents the tag shipped, read the way the cut reads membership, with each one's impact and audit counts and the changelog section; the host page dispatches the reflection-composer with that seed; render writes four answered sections plus computed metrics, and blank answers are refused. The v0.12.0 retrospective in the record shows the whole path working. Concerns: that real seed counted six intents whose fidelity reviews were OWED as audited, so the record's audited list and its '11 with audit notes, 0 without' line are wrong (see ac-2). Also, since the product thinker's placement of 2026-10-09, /abcd:reflect is an agents' page hidden from the person's menu, so the persona reaches it through an agent.
  evidence: internal/core/reflect/seed.go:128 — "func BuildSeed(root, tag string) (Seed, error) {"
  evidence: commands/reflect.md:55 — "Dispatch the `reflection-composer` agent with the seed."
  evidence: internal/core/reflect/write.go:233 — "sec.Heading(), strings.TrimSpace(ans.Text))"
  evidence: internal/core/reflect/write_test.go:29 — "func TestWriteProducesTheRetrospectiveWithFiveSections(t *testing.T) {"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:11 — "# Retrospective for v0.12.0"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:56 — "- Intents shipped: 11 (11 with audit notes, 0 without)"
  evidence: commands/reflect.md:6 — "user-invocable: false"
- ac-2 — NOT_MET: The seed offers 'abcd intent audit < itd-N>' for each intent it reads as unaudited, the host page names each one before the interview, and the fixtures (a placeholder, an OWED marker alone, no notes) pass. But readAuditNotes counts any other prose in the section as an audit, and the brief's own rule says an owed review is not audit notes. Promised: every shipped intent with no notes from the auditor is named and offered the audit before the retrospective continues. Exists: an intent whose review is OWED but whose Audit Notes section carries an amendment note is read as audited, so it is neither named nor offered. In the only real run, v0.12.0, six of its eleven intents (itd-24 among them) were in that state: the record says 'unaudited: []' and lists five receipts, and its own text concedes 3 features shipped before their after-delivery audit. The plain-Terminal interview, added later, shows no offer before the first question either.
  evidence: internal/core/reflect/seed.go:169 — "s.Unaudited = append(s.Unaudited, AuditOffer{IntentID: it.ID"
  evidence: commands/reflect.md:44 — "- **Intents without audit notes** (`unaudited[]`): name each one to the product thinker"
  evidence: internal/core/reflect/seed_test.go:85 — "func TestSeedNamesEachIntentWithoutAuditNotesAndOffersTheAudit(t *testing.T) {"
  evidence: internal/core/reflect/audit.go:55 — "prose = true"
  evidence: internal/core/reflect/audit.go:81 — "it.Audited = true"
  evidence: .abcd/development/brief/04-surfaces/09-reflect.md:55 — "A placeholder, an owed review and an absent section are not audit"
  evidence: .abcd/development/intents/shipped/itd-24-reflect-command.md:108 — "Fidelity review OWED (receipt rcp-2215987d5eb7)."
  evidence: .abcd/development/intents/shipped/itd-24-reflect-command.md:111 — "- 2026-09-30: the press release, scope item 1 and criterion 1 read release membership"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:6 — "audited: [itd-24, itd-42,"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:7 — "unaudited: []"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:8 — "audit_receipts: [rcp-e6bcb27c8050, rcp-7a2c62ab0228, rcp-121c23aff688, rcp-12ccf7627aac, rcp-ebf7d171b544]"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:37 — "Checks running too late: 3 features shipped before their after-delivery audit"
  evidence: internal/core/interview/tasks.go:12 — "an intent shipped with no audit"
- ac-3 — MET: NothingShippedError's text is the criterion's own wording. BuildSeed returns it before anything is created, the CLI exits 1 with it for both the seed render and the write, and the tests assert the exact text and that no retrospectives store appears.
  evidence: internal/core/reflect/reflect.go:67 — "no intent shipped in `%s` — nothing shipped to reflect on"
  evidence: internal/core/reflect/seed.go:155 — "return Seed{}, &NothingShippedError{Tag: tag}"
  evidence: internal/surface/cli/reflect_cli_test.go:148 — "func TestReflectRefusesAReleaseThatShippedNothing(t *testing.T) {"
  evidence: internal/surface/cli/reflect_cli_test.go:158 — "no intent shipped in `v0.3.0` — nothing shipped to reflect on"
  evidence: internal/core/reflect/write_test.go:140 — "func TestWriteRefusesAReleaseThatShippedNoIntentAndWritesNothing(t *testing.T) {"
- ac-4 — MET_WITH_CONCERNS: underFloor marks 'it worked' thin: fewer than two clauses of at least three words. Write then refuses, writing nothing, until every thin section has a follow-up, and returns that section's question, which the reflection-composer asks. Concerns: the thin original is still written verbatim above the follow-up reply (v0.12.0's 'What went well' opens with 'How we worked.'), and the reply is not held to the floor, so a thin answer can still be committed after one question, although the agent page says a thin answer is not committed.
  evidence: internal/core/reflect/thin.go:33 — "func underFloor(s Section, text string) (bool, string) {"
  evidence: internal/core/reflect/write.go:105 — "if !isThin || strings.TrimSpace(a.FollowUp) !="
  evidence: internal/surface/cli/reflect_cli_test.go:276 — "func TestReflectWriteMeetsAThinAnswerWithAQuestion(t *testing.T) {"
  evidence: internal/core/reflect/write.go:234 — "if f := strings.TrimSpace(ans.FollowUp); f !="
  evidence: internal/core/reflect/write_test.go:109 — "func TestWriteTakesAThinAnswerOnceItsFollowUpIsAnswered(t *testing.T) {"
  evidence: .abcd/development/retrospectives/v0.12.0/README.md:31 — "How we worked."
  evidence: agents/reflection-composer.md:82 — "A thin answer is **not** committed."
- ac-5 — MET: Plan copies every < release-tag>/README.md under .abcd/development/retrospectives into the lifeboat as retrospectives/< tag>/README.md, a record-derived family sealed by the record manifest. The core and CLI tests pack two retrospectives verbatim, leave out files under the store that are not retrospectives, and write the two back on embark.
  evidence: internal/core/lifeboat/plan.go:288 — "if ctx.IsDir(corereflect.RetrospectivesRelDir) {"
  evidence: internal/core/lifeboat/plan.go:299 — "pb.copyRecord(ctx, src, path.Join(retrospectivesLifeboatDir, tag, retrospectiveLeaf))"
  evidence: internal/core/lifeboat/embark_types.go:327 — "// The release retrospectives (itd-24): retrospectives/< release-tag>/README.md,"
  evidence: internal/core/lifeboat/retrospectives_test.go:50 — "func TestPlanPacksEveryRetrospective(t *testing.T) {"
  evidence: internal/surface/cli/reflect_lifeboat_cli_test.go:45 — "func TestDisembarkPackCarriesEveryRetrospectiveAndEmbarkWritesThemBack(t *testing.T) {"
- ac-6 — MET_WITH_CONCERNS: embark from writes the retrospectives back. embark lessons verifies the manifest, ranks every lesson by term overlap against the target's framing chapter or a --brief file, and shows the top three, then the rest as a list, with 'which of these apply?'; the embark page has the host ask that one question. Concerns: embark from runs no press-release interview (the verification matrix marks that interview staged), so the lessons are a separate step the page has the host take. And embark never writes the brief, so on a freshly embarked target without --brief the lessons come back unranked, yet the first three are still headed 'most like this voyage'.
  evidence: internal/surface/cli/cli.go:1437 — "lessonsCmd := &cobra.Command{"
  evidence: internal/core/reflect/lessons.go:15 — "const TopLessons = 3"
  evidence: internal/core/lifeboat/retrospectives.go:54 — "func PredecessorLessons(lifeboatDir, framing string) (LessonsView, error) {"
  evidence: internal/surface/cli/reflect_lifeboat_cli_test.go:77 — "func TestEmbarkLessonsShowsTheFewMostLikeTheBrief(t *testing.T) {"
  evidence: commands/embark.md:83 — "Show the product thinker the `top` few (three), each with the release it came from, and"
  evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:62 — "| Embark press-release interview | **(staged)**"
  evidence: commands/embark.md:59 — "`from` runs the same planner as `probe`, then writes each `create` file into the"
  evidence: .abcd/development/brief/04-surfaces/03-embark.md:104 — "The five record families — ADRs, issues, intents,"
  evidence: internal/surface/cli/cli.go:1473 — "const embarkFramingRel ="
  evidence: internal/core/lifeboat/retrospectives.go:122 — "strings.TrimSpace(framing) =="
  evidence: internal/core/lifeboat/retrospectives.go:151 — "most like this voyage — which of these apply?"
- ac-7 — NOT_MET: The seed lists planned intents whose live target_release equals the tag, the CLI prints a WARNING listing them, the host page asks, and Write refuses until --proceed; the fixtures pass because they create a tag without running a cut. But launch ship's ingest rewrites every target the cut passes to next (launch.MissedTargets, then intent.PlanTargetMoves) in the commit that rolls the changelog, before the tag exists. So for a release cut by launch ship no intent's target_release names the tag by the time reflect can run. Promised: reflect warns about, lists, and asks to confirm past the intents targeted at the release that did not ship. Exists: it can find none. CHANGELOG.md records itd-2610112239022766 as targeted and not shipped in v0.14.2, v0.15.0 and v0.15.1; that record now reads target_release: next, and reflect on any of those tags would not name it.
  evidence: internal/core/reflect/seed.go:173 — "if it.Bucket == intent.BucketPlanned && it.TargetRelease == tag {"
  evidence: internal/surface/cli/reflect.go:357 — "WARNING: %d intent(s) targeted at %s have not shipped"
  evidence: commands/reflect.md:49 — "(`unshipped_targets[]`): warn the product thinker, list them, and ask whether to"
  evidence: internal/core/reflect/write_test.go:153 — "func TestWriteAsksBeforeProceedingPastUnshippedTargets(t *testing.T) {"
  evidence: internal/core/reflect/fixture_test.go:55 — "planned with `target_release: v0.2.0` and never shipped"
  evidence: internal/core/release/emit.go:346 — "cut.Moves = launch.MissedTargets(cut.Targets, cut.NextTag)"
  evidence: internal/core/release/ingest.go:455 — "rewrites, err := intent.PlanTargetMoves(root, moves)"
  evidence: internal/core/launch/target.go:85 — "func MissedTargets(targets []TargetedIntent, nextTag string) []TargetMove {"
  evidence: CHANGELOG.md:23 — "Targeted and not shipped in this release, so each targets the next release (`next`): itd-2610112239022766 (targeted `next`)."
  evidence: .abcd/development/intents/planned/itd-2610112239022766-when-someone-reports-a-security-hole-in.md:14 — "target_release: next"
- ac-8 — MET: launch ship sets retrospective_owed to reflect.Nudge(tag) only when the cut is written, and prints it as the run's last line. release.Ingest, which writes the cut, has no other caller, and no other surface mentions an owed retrospective. The test asserts the word appears exactly once, at the end, and not at all on an emit that writes nothing.
  evidence: internal/core/reflect/nudge.go:10 — "A retrospective for %s is owed: run /abcd:reflect %s when you are ready."
  evidence: internal/surface/cli/ship.go:677 — "res.RetrospectiveOwed = reflect.Nudge(ingested.Cut.NextTag)"
  evidence: internal/surface/cli/reflect_cli_test.go:363 — "func TestLaunchShipSaysOnceThatARetrospectiveIsOwed(t *testing.T) {"
  evidence: internal/surface/cli/reflect_cli_test.go:371 — "strings.Count(string(out),"

Gap audit:
- honoured:
  - The interview opens from what the release shipped: the intents the tag shipped, read the way the release cut reads membership, each with its impact and audit counts, plus the changelog section the cut composed.
    evidence: internal/core/reflect/seed.go:277 — "func shippedBy(root, tag, prev string, corpus intent.Corpus) ([]member, error) {"
    evidence: internal/core/reflect/seed_test.go:22 — "func TestSeedIsTheIntentsTheTagShipped(t *testing.T) {"
  - The output is a structured five-section README in the durable record tier that links to the changelog section and to each intent's audit notes rather than copying them, and one is already in the permanent record (v0.12.0).
    evidence: internal/core/reflect/write.go:216 — "Each intent's audit notes stay on the intent and are linked here, not repeated."
    evidence: internal/core/reflect/write_test.go:83 — "RATIONALE-NEVER-COPIED"
    evidence: .abcd/development/retrospectives/v0.12.0/README.md:11 — "# Retrospective for v0.12.0"
  - Metrics are simple counts and dates computed from the seed, never asked.
    evidence: internal/core/reflect/write.go:241 — "- Intents shipped: %d (%d with audit notes, %d without)"
    evidence: internal/core/reflect/sections.go:17 — "var AskedSections = []Section{WentWell, CouldImprove, Lessons, Decisions}"
  - A release that shipped no intent is refused in the criterion's words, and nothing is written.
    evidence: internal/core/reflect/reflect.go:67 — "no intent shipped in `%s` — nothing shipped to reflect on"
    evidence: internal/surface/cli/reflect_cli_test.go:148 — "func TestReflectRefusesAReleaseThatShippedNothing(t *testing.T) {"
  - The release is the only grain: an intent id is refused, and the refusal names the intent audit.
    evidence: internal/surface/cli/reflect.go:271 — "is an intent; reflect takes a release tag"
    evidence: internal/surface/cli/reflect_cli_test.go:190 — "func TestReflectRefusesAnIntentID(t *testing.T) {"
  - Future lifeboats carry the retrospectives forward: disembark packs every one, and embark writes them back verbatim.
    evidence: internal/core/lifeboat/plan.go:299 — "pb.copyRecord(ctx, src, path.Join(retrospectivesLifeboatDir, tag, retrospectiveLeaf))"
    evidence: internal/core/lifeboat/retrospectives_test.go:102 — "func TestEmbarkFromWritesRetrospectivesBack(t *testing.T) {"
  - Reflection has a single nudge: a written cut ends with one line saying a retrospective is owed, and nothing repeats it or waits on it.
    evidence: internal/surface/cli/ship.go:677 — "res.RetrospectiveOwed = reflect.Nudge(ingested.Cut.NextTag)"
    evidence: internal/surface/cli/reflect_cli_test.go:363 — "func TestLaunchShipSaysOnceThatARetrospectiveIsOwed(t *testing.T) {"
  - The reflection-composer agent runs the interview and asks the section's follow-up question when an answer falls under the floor.
    evidence: commands/reflect.md:55 — "Dispatch the `reflection-composer` agent with the seed."
    evidence: internal/core/reflect/thin.go:33 — "func underFloor(s Section, text string) (bool, string) {"
- diverged:
  - Each intent comes with the audit notes the intent auditor wrote: delivered as a prose test, under which an owed review that carries an amendment note reads as audited. The seed, the record's audited list and its metrics therefore overstate the audits; v0.12.0 lists all eleven intents as audited against five receipts, and this applies to six shipped intents at HEAD.
    evidence: internal/core/reflect/audit.go:81 — "it.Audited = true"
    evidence: .abcd/development/retrospectives/v0.12.0/README.md:7 — "unaudited: []"
    evidence: .abcd/development/intents/shipped/itd-24-reflect-command.md:108 — "Fidelity review OWED (receipt rcp-2215987d5eb7)."
  - Clarifying questions for thin answers add to the thin answer rather than replace it: the thin original is committed verbatim beside the follow-up reply, and the reply is not held to the floor.
    evidence: internal/core/reflect/write.go:234 — "if f := strings.TrimSpace(ans.FollowUp); f !="
    evidence: .abcd/development/retrospectives/v0.12.0/README.md:31 — "How we worked."
  - Embark surfaces predecessor retrospectives during the press-release interview: delivered as a separate embark lessons step, because the press-release interview embark from would run is staged. With no framing chapter and no --brief, the lessons are unranked but still headed 'most like this voyage'.
    evidence: .abcd/development/brief/06-delivery/02-verification-matrix.md:62 — "| Embark press-release interview | **(staged)**"
    evidence: internal/core/lifeboat/retrospectives.go:151 — "most like this voyage — which of these apply?"
  - Reflection becomes a first-class abcd primitive run as /abcd:reflect: since the product thinker's placement of 2026-10-09 it is an agents' page hidden from the person's /abcd: menu, while the cut's nudge still tells the person to run /abcd:reflect < tag>.
    evidence: commands/reflect.md:6 — "user-invocable: false"
    evidence: .abcd/development/brief/04-surfaces/09-reflect.md:41 — "hidden from a person's `/abcd:` menu (the product thinker's placement of"
    evidence: internal/core/reflect/nudge.go:10 — "A retrospective for %s is owed: run /abcd:reflect %s when you are ready."
  - The interview also runs in a plain Terminal (abcd reflect interview, added after delivery). There the verb shows no audit offer before the first question, and the role is told only to use an unaudited intent to sharpen a question.
    evidence: internal/surface/cli/reflect.go:166 — "func newReflectInterviewCommand(asJSON *bool) *cobra.Command {"
    evidence: internal/core/interview/tasks.go:12 — "an intent shipped with no audit"
- missing:
  - The warning about intents targeted at the release that did not ship, for any release cut by launch ship: the cut has already rewritten those targets to next, and the seed does not read the cut's record of the move.
    evidence: internal/core/release/ingest.go:455 — "rewrites, err := intent.PlanTargetMoves(root, moves)"
    evidence: internal/core/reflect/seed.go:173 — "if it.Bucket == intent.BucketPlanned && it.TargetRelease == tag {"
    evidence: CHANGELOG.md:23 — "Targeted and not shipped in this release, so each targets the next release (`next`): itd-2610112239022766 (targeted `next`)."
    evidence: .abcd/development/intents/planned/itd-2610112239022766-when-someone-reports-a-security-hole-in.md:14 — "target_release: next"
  - Future intents reference past lessons: nothing on the intent side reads the retrospective store. Lessons are read only by embark's lifeboat view.
    evidence: internal/core/reflect/lessons.go:28 — "func ReadLessons(readme []byte) []Lesson {"
    evidence: internal/core/lifeboat/retrospectives.go:54 — "func PredecessorLessons(lifeboatDir, framing string) (LessonsView, error) {"
<!-- abcd-review-end receipt=rcp-2215987d5eb7 -->

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
