---
id: itd-2611041636196213
slug: when-an-agent-goes-wrong-abcd-tells-it
spec_id: spc-2611041649558539
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-3, itd-103]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Agents are corrected when wrong and steered where no check reaches

## Press Release

> When an agent goes wrong, abcd tells it why and what to do instead; where no check can catch the mistake, abcd steers it in a short line as that work starts. Alice, a solo founder, watches an agent try an unsafe delete: it is refused and shown the safe form, in a sub-agent as much as in the main session. The session she leaves running for days holds the work, not forty-five rules no agent read, and her next usage panel has one less thing on it.

## Why This Matters

abcd's rules exist for two jobs: to correct an agent that has gone wrong, and to
steer it towards what no check can catch. Today one mechanism does both, and does
neither well. The prompt hook sends every matching rule in advance. In the seven
days to 4 Nov 2026 the host cut that output 475 times to a 2,000-character preview
(iss-2611041419354106), and the session ledger recorded the cut text as delivered.
At tip, ordinary prompts produce 12,000 to 38,000 characters against a cap of
10,000, out of 72,002 characters in all.

Meanwhile the guard already does the first job well. Its refusal carries why and
what to run instead, at the exact command, and it reaches sub-agents, which never
receive the prompt hook. In the 30 days to 4 Nov 2026 it made 478 refusals in about
145,000 shell commands; 368 were in sub-agents, and in main sessions the latest
prompt named the command in only 11 of 110.

The research is thinner than the vendor limits suggest. McMillan (arXiv 2605.10039,
1,650 sessions) finds no detectable effect of file size from 25 to 500 lines, so no
count of rules is a proven ceiling; this intent rests on the delivery limit and the
logs, not on a number. A single-author test (github.com/B0yko/obey, Haiku 5.5, not
peer reviewed) saw a hook alone reach full compliance with the rule text removed. No
controlled study compares refusal alone with pre-teaching plus refusal.

The 129 rules of today are a snapshot. What lasts is the system: rules come and go,
and each says which job it does.

## Mechanism

We expect agents to behave as they do now on far less text, because the rule text
that is cut today changes nothing and the guard already corrects what it refuses. In
the 30 days to 4 Nov 2026 main sessions took 110 refusals. If that rate more than
doubles on comparable work once the text is gone, the claim is wrong and the most
refused hazards go back into the steering line.

## Scope Conditions

- Holds for a host that runs the pre-tool-use and prompt hooks, where the pre-tool-use <!-- cond: cond-2611041649555302 -->
  hook runs before the host's own permission prompt. A host with neither reads the same
  rules through `abcd rules`.
- The cap is the host's documented 10,000 characters for one hook's output, read <!-- cond: cond-2611041649558257 -->
  2026-11-04; another host's cap may differ, and the budget sits in one place.
- A correction helps only if it comes before the harm. A warned command still runs, so <!-- cond: cond-2611041649557746 -->
  its rule stays a steering rule, and so does a rule whose check fires too late to save
  the work, such as one at preflight.

## Acceptance Criteria

- **Given** a prompt, **when** the hook answers, **then** its whole output, rules, notes,
  status and away block together, is within the host's cap.
- **Given** matched rules that do not all fit, **when** the hook answers, **then** each
  domain goes in full or as its first rule and a line giving the command that shows the
  rest, and the session records only that: nothing the host cut is counted as sent.
- **Given** full rules and a matching note, **when** the hook answers, **then** the note
  still has room, and a note or page too long to fit comes back as one line rather than
  not at all.
- **Given** a prompt about shell work with the guard on, **when** the hook answers,
  **then** the agent gets one line saying a guard judges each command and the rules for
  commands the guard only warns about, none of the text for commands it refuses, and
  `abcd rules SHELL` still lists every hazard.
- **Given** a repository that has switched its guard off, **when** the hook answers,
  **then** every shell rule stays a steering rule, each opens `Hazard (guard off)` under a first
  line saying the guard is off, and the line saying a guard judges each command is not
  sent. Eleven registry entries keep
  wording that names the guard after that opening; rewording them belongs to the
  rule-by-rule change.
- **Given** the bundled registry, **when** its tests run, **then** every entry that refuses
  or warns says why and what to do instead, and every entry appears in exactly one of the
  two lists, so a hazard can never fall out of both.
- **Given** a person wants a new rule, **when** a machine can catch it in a shell command,
  **then** they add an entry to `.abcd/guard.json`, and otherwise a line to `rules.json`,
  neither of which changes format, so no existing override file has to change.
- **Given** `abcd rules --json`, **when** a reader asks what each domain sends, **then**
  it separates the steering rules from the ones a check carries and gives their sizes, so
  the review of the rule set can be run again as rules change.

## Open Questions

None. The product thinker confirmed the reversal below at sign-off, 2026-11-04.

## Decisions

- itd-103 is refined, not superseded: the registry, the refusal and `abcd rules SHELL`
  stay, and only what is sent ahead of time changes.
- itd-2609292109214516 stays separate: it makes injection a seam for another back end,
  and this intent changes what the native loader sends. Whichever lands second carries
  the budget and the two lists through its seam.
- A delete entry counts as refused though it sits in the warn tier: the guard refuses it
  unless it proves the target safe, so only what is sure to run is sent ahead of time. A
  repository's rewording of one is still sent, since a refused delete shows the proof's
  words. Its lesson keeps its `Warned by the guard` opening until the rule-by-rule
  change.
- Whatever of the 2,500 bytes kept for notes they leave unused goes back to the rules,
  so a short note does not shorten a domain that fits beside it.
- With the guard off, the rules cannot all fit one turn, so the first rule says the guard
  is off and the rest are read with `abcd rules SHELL`.
- Refusal counts per hazard are out of scope: the mechanism's threshold reads the logs
  already kept, so a counter waits for a check that needs it.

## Related

- **Reverses** the teaching plane of itd-103 as wired by the product thinker's rulings
  J10 (dcn-2609290000000014) and CK1 (dcn-2609300000000008): refused hazards are no longer
  taught ahead of time, as the product thinker confirmed on 2026-11-04.
- **Narrows** dcn-2610161612269647 on teaching the command forms the host can check: a
  form the guard refuses is corrected by the refusal, which comes before the host's own
  prompt, so the agent does not stall waiting for a person.
- **Refines** spc-2610111627213138: its brackets and its shortened form are kept, and the
  same shortened form now also answers an overflow; rules still come back in full whenever
  they fit, so its ruling stands.
- This is the whole delivery: the same change resolves iss-2611041419354106 and carries
  `Delivers:` for this intent.
- Applying the rule-by-rule review of the snapshot is a later change, once the product
  thinker has marked up the review page; nothing here rewords a rule.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-889ac8de461c -->
Fidelity review — receipt rcp-889ac8de461c (verifier intent-auditor claude-opus-5-5).

Provenance: intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:2cbe68c614f253dcde6f3c474a28772651b24871b49a8c52d9bf74513094ee2a
Input attestations: diff:8d7dc5040..d287ede44@sha256:c0ca881d6004cb2b3c9ceb34d121708537957b51d0ec411ecef1bedc3bcb2316;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 5 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET_WITH_CONCERNS: Code bound: InjectionBudgetBytes is 9,500, the router reserves the status and away bytes first, the rules and notes share what is left, including on the re-render at rules_prompt.go:39, and two joins add a few bytes. Read, not run: TestTheHooksWholeOutputStaysUnderTheHostCap covers rules, notes and away. Concerns: the test does not exercise status (its own comment says so); status and away are not clamped, so their bound rests on capped line counts and validated fields; the away systemMessage (cleaned to at most 1,024 bytes) is outside the reservation; and neither the 10,000 figure nor how the host counts it, field by field or whole, can be checked from the tree, and no live host capture is in the delivery.
  evidence: internal/core/rules/inject.go:47 — "const InjectionBudgetBytes = HostOutputCap - hostOutputMargin"
  evidence: internal/surface/cli/cli.go:1946 — "reserved := len(status.Text) + len(away.Block)"
  evidence: internal/surface/cli/rules_prompt.go:39 — "res = rules.InjectWithin(rs, prompt, prev, backstop, bracket, max(0, rules.InjectionBudgetBytes-reserved-len(mem.Text)))"
  evidence: internal/surface/cli/memory_recall_budget_test.go:113 — "each domain as a whole form. Status has no origin to read here; its block is"
  evidence: internal/surface/cli/hook_away.go:189 — "SystemMessage: termsafe.CleanProseLine(message, 1024),"
- ac-2 — MET_WITH_CONCERNS: The ladder sends full or first rule plus command, whole blocks only, and the ledger records only domains sent (read TestInjectWithinGivesEveryDomainItsFirstRuleAndACommand and TestADomainWithNoRoomIsTriedAgain). The earlier concern about a short note shortening a domain that fits is resolved: unused floor bytes go back before the final render, and the test now asserts both the short-note and long-note cases. Concerns that hold: two outcomes outside the criterion's two forms exist, a heading-and-command pointer with no first rule and a domain left out for a turn. New residual: a domain above the 7,000 first-pass limit is shortened first, which frees room that notes may take past the floor; once notes take 2,500 bytes or more, no re-render happens and the short form stands until a reset or the backstop, even where the domain would have fitted had the notes stayed within the floor. By my count the bundled LOAD domain's rules total about 7.5 KB, which puts it in this window.
  evidence: internal/core/rules/inject.go:232 — "ledger[k.domain.Name] = Signature(k.domain)"
  evidence: internal/core/rules/inject.go:323 — "func renderWithinBudget(fresh []ResolvedDomain, rulesFrom map[string]string, shrinks bool, budget int) (string, []sentDomain) {"
  evidence: internal/core/rules/rules.go:1080 — "func renderPointer(d ResolvedDomain) string {"
  evidence: internal/surface/cli/rules_prompt.go:38 — "if len(mem.Text) < rules.MemoryFloorBytes {"
  evidence: internal/surface/cli/memory_recall_budget_test.go:183 — "a short note did not hand back the floor it left unused"
  evidence: internal/core/rules/defaults/rules.json:58 — ""LOAD": {"
- ac-3 — MET_WITH_CONCERNS: RulesBudget always leaves the 2,500 floor on the first pass, notes take what the rules left, and the fresh row has a line bound, so a note or page that misses full form comes back as one line (read TestRulesAndNotesShareOneTurnBudget, TestAnItemLargerThanTheWholeBudgetDoesNotBlockTheRest, TestAPageLargerThanTheBudgetIsNamedAndTheRestComeBack). Concerns that hold: a page over 128 KiB is refused at read and never recalled, so it comes back not at all, which the criterion rules out and the brief admits; and once the 2 KiB line bound or the budget is spent, remaining items are omitted until a later turn.
  evidence: internal/core/rules/inject.go:54 — "const MemoryFloorBytes = 2_500"
  evidence: internal/core/rules/brackets.go:81 — "{Bracket: BracketFresh, MinFreePercent: 70, ShrinkRules: false, MemoryFullBytes: MemoryAll, MemoryLineBytes: 2 * 1024},"
  evidence: internal/core/memory/recallpages.go:217 — "it is larger than the %d bytes a recall reads, so it is never recalled"
  evidence: internal/core/recall/recall.go:211 — "func tooLarge(block string) bool { return len(headerLine)+len(block) > rules.InjectionBudgetBytes }"
- ac-4 — MET_WITH_CONCERNS: With the guard on and the bundled registry, Rules is the steering line plus the three commands sure to run (rm-rf-working-directory, git-reset-hard, git-clean). The five delete entries and all refused entries are in Checked, and the block renders Rules only (read TestADeleteTheGuardCanRefuseIsNotSentAsWarned, TestAShellPromptGetsNoTextForWhatIsRefused and the hook-level TestHookPromptRouterTeachesShellHazards). RenderListing lists both lists. The earlier concern about bundled delete text being sent is resolved. Concerns that hold: a repository's rewording of a delete entry is still sent ahead of time, opening `Warned by the guard`, for a command the guard refuses unless proven safe; this is a recorded decision, but it departs from 'none of the text for commands it refuses'. In a bracket that shrinks rules, or on overflow, SHELL goes as the steering line alone, so the warned rules are not sent then. New: the listing heads the second list 'read at the refusal', which is not so for the five delete entries, since a refused delete shows the proof's words, and those five still open `Warned by the guard` in that list.
  evidence: internal/core/guard/teach.go:61 — "if r.Disabled || (e.Tier == TierWarn && (!deleteEntryIDs[id] || own)) {"
  evidence: internal/core/rules/shell.go:67 — "d.Rules = append([]string{guard.SteeringLine}, warned...)"
  evidence: internal/core/rules/steering_test.go:48 — "func TestADeleteTheGuardCanRefuseIsNotSentAsWarned(t *testing.T) {"
  evidence: internal/core/rules/steering_test.go:157 — "if strings.Contains(res.Text, "Refused by the guard") {"
  evidence: internal/core/guard/teach_test.go:291 — "func TestARepositorysWordsOnADeleteAreSentAheadOfTime(t *testing.T) {"
  evidence: internal/core/rules/rules.go:975 — "const checkedLead = "Carried by the guard, read at the refusal and not sent ahead of time:\n""
  evidence: internal/core/rules/brackets.go:82 — "{Bracket: BracketModerate, MinFreePercent: 40, ShrinkRules: true, MemoryFullBytes: 8 * 1024, MemoryLineBytes: MemoryAll},"
- ac-5 — MET_WITH_CONCERNS: As now worded, the criterion holds at the list level: under a disabled registry Rules opens with OffLine, every other rule opens `Hazard (guard off) (`, Checked is empty, and the steering line is absent (read TestGuardOffKeepsEveryShellRuleSteeringAndClaimsNoGuard and TestShellLessonsUnderACommittedDisabledGuardSayItIsOff). Eleven matches a text search of the rendered lessons, one of them only through the id rm-unguarded-variable-path. The help text and brief no longer claim every rule is sent. Concerns that hold: by my computation the 38 bundled lessons alone are at least 17,877 bytes against a 9,500 budget, so the hook sends OffLine and a command line and no hazard text, and an agent reads a hazard only by running `abcd rules SHELL`; 'stays a steering rule' therefore describes the list, not what arrives. No CLI-level test runs the hook with the guard off; the assertions are on the generated domain. This criterion was reworded after the earlier audit to match this behaviour, and the intent's three new decisions name no sign-off for that change.
  evidence: internal/core/rules/shell.go:64 — "d.Rules = append(append([]string{guard.OffLine}, warned...), own...)"
  evidence: internal/core/guard/teach.go:39 — "const OffLine = "The shell guard is off in this repository, so nothing refuses or warns about a hazardous command. `abcd rules SHELL` lists the hazards.""
  evidence: internal/core/rules/shell_test.go:494 — "if !strings.HasPrefix(r, "Hazard (guard off) (") {"
  evidence: internal/core/rules/steering_test.go:115 — "if d.Rules[0] != guard.OffLine {"
  evidence: .abcd/development/brief/04-surfaces/17-guard.md:151 — "sends the first line and the command for the rest."
  evidence: .abcd/development/intents/shipped/itd-2611041636196213-when-an-agent-goes-wrong-abcd-tells-it.md:114 — "- With the guard off, the rules cannot all fit one turn, so the first rule says the guard"
- ac-6 — MET: Judged on tests read, not run. TestEveryBundledEntryCorrectsWithWhyAndWhatInstead requires a non-empty why and successor for every bundled entry, and I confirmed all 38 entries in guard.json have both. TestEveryLessonIsInExactlyOneList requires each lesson, the three with no entry included, to be in exactly one list with the guard on and off. SplitLessons is a two-way branch, and parse admits only two tiers.
  evidence: internal/core/guard/teach_test.go:341 — "func TestEveryBundledEntryCorrectsWithWhyAndWhatInstead(t *testing.T) {"
  evidence: internal/core/rules/steering_test.go:73 — "func TestEveryLessonIsInExactlyOneList(t *testing.T) {"
  evidence: internal/core/guard/teach.go:49 — "func (r Registry) SplitLessons(bundled Registry) (warned, refused []string) {"
- ac-7 — MET: The diff to guard.go changes only a comment, and Domain gains only a `json:"-"` field, so neither file format moves and no file can declare the second list (TestNoRuleFileCanDeclareChecked read, not run). mergeDomain copies base, so Checked survives overrides. A guard.json entry lands in Rules or Checked by SplitLessons, and the help text states the routing. An old SHELL rules override still loads; it now draws a stderr note naming the steering line as withheld, which requires no change to the file.
  evidence: internal/core/rules/rules.go:78 — "Checked []string `json:"-"`"
  evidence: internal/core/rules/steering_test.go:184 — "func TestNoRuleFileCanDeclareChecked(t *testing.T) {"
  evidence: internal/surface/cli/cli.go:2687 — ".abcd/guard.json, and any other in .abcd/rules.json."
- ac-8 — MET: ResolvedDomain.MarshalJSON adds checked, rules_bytes and checked_bytes for every domain, and both the bare path (rulesView.Domains) and the scoped path marshal that type. Judged on TestListingJSONSeparatesTheListsAndGivesSizes read, not run; that test marshals the type directly, and no CLI test asserts the new fields, but the code path is direct.
  evidence: internal/core/rules/rules.go:165 — "func (d ResolvedDomain) MarshalJSON() ([]byte, error) {"
  evidence: internal/surface/cli/cli.go:2639 — "Domains []rules.ResolvedDomain `json:"domains"`"
  evidence: internal/core/rules/steering_test.go:204 — "func TestListingJSONSeparatesTheListsAndGivesSizes(t *testing.T) {"

Gap audit:
- honoured:
  - The prompt hook no longer sends the 64 KiB budget's worth of rule text; rules, notes, status and away share one budget under the host's cap
    evidence: internal/core/rules/inject.go:47 — "const InjectionBudgetBytes = HostOutputCap - hostOutputMargin"
  - Refused hazards, and bundled deletes the guard refuses unless proven safe, are not sent ahead of time and stay listed by abcd rules
    evidence: internal/core/rules/shell.go:68 — "d.Checked = append(refused, own...)"
    evidence: internal/core/rules/rules.go:994 — "func renderChecked(d ResolvedDomain) string {"
  - Abcd steers in a short line as shell work starts, and says so plainly where the guard is off
    evidence: internal/core/guard/teach.go:34 — "const SteeringLine = "A guard judges each shell command before it runs."
    evidence: internal/core/guard/teach.go:39 — "const OffLine = "The shell guard is off in this repository"
  - Rules still come back in full when they fit beside a short note: unused floor bytes go back before the final render
    evidence: internal/surface/cli/rules_prompt.go:38 — "if len(mem.Text) < rules.MemoryFloorBytes {"
    evidence: .abcd/development/intents/shipped/itd-2611041636196213-when-an-agent-goes-wrong-abcd-tells-it.md:112 — "- Whatever of the 2,500 bytes kept for notes they leave unused goes back to the rules,"
- diverged:
  - Each domain goes in full or as its first rule and a line: delivered with two more outcomes, a heading-and-command pointer and a domain left out for a turn
    evidence: internal/core/rules/rules.go:1080 — "func renderPointer(d ResolvedDomain) string {"
    evidence: internal/core/rules/inject.go:396 — "func packPointers(fresh []ResolvedDomain, room int) (sent []sentDomain, omitted, left int) {"
  - Rules still come back in full whenever they fit: a domain above the first-pass limit is shortened before the notes are taken, and notes of 2,500 bytes or more in the room that freed leave it short with no re-render
    evidence: internal/surface/cli/rules_prompt.go:35 — "res := rules.InjectWithin(rs, prompt, prev, backstop, bracket, rules.RulesBudget(reserved))"
    evidence: internal/surface/cli/rules_prompt.go:38 — "if len(mem.Text) < rules.MemoryFloorBytes {"
  - None of the text for commands the guard refuses: a repository's rewording of a delete entry is sent ahead of time for a command refused unless proven safe, opening `Warned by the guard`; this is a recorded decision
    evidence: internal/core/guard/teach.go:61 — "if r.Disabled || (e.Tier == TierWarn && (!deleteEntryIDs[id] || own)) {"
    evidence: .abcd/development/intents/shipped/itd-2611041636196213-when-an-agent-goes-wrong-abcd-tells-it.md:107 — "- A delete entry counts as refused though it sits in the warn tier: the guard refuses it"
  - With the guard off, every hazard stays steering in the list, but the hook sends only the line saying the guard is off and no hazard text
    evidence: .abcd/development/brief/04-surfaces/17-guard.md:151 — "sends the first line and the command for the rest."
    evidence: internal/surface/cli/cli.go:2682 — "guard switched off, every rule is steering, more than one turn holds, so a line"
  - A note or page too long to fit comes back as one line rather than not at all: a page over 128 KiB is still never recalled
    evidence: internal/core/memory/recallpages.go:217 — "it is larger than the %d bytes a recall reads, so it is never recalled"
  - The listing says the second list is read at the refusal, but a refused delete shows the proof's words rather than the entry's lesson
    evidence: internal/core/rules/rules.go:975 — "const checkedLead = "Carried by the guard, read at the refusal and not sent ahead of time:\n""
    evidence: .abcd/development/brief/05-internals/03-configuration.md:1447 — "for them. The delete entries' rules are there too, since a delete warns only once its"
  - The closed spec no longer describes the delivery: it says a turn no note comes back on gives the floor back, and that guard-off lessons stay in Rules as today with no first line
    evidence: .abcd/development/specs/closed/spc-2611041649558539-when-an-agent-goes-wrong-abcd-tells-it.md:22 — "the rules actually left, and a turn no note comes back on gives the floor back to the"
    evidence: .abcd/development/specs/closed/spc-2611041649558539-when-an-agent-goes-wrong-abcd-tells-it.md:45 — "variable-built `-c`, unprovable delete). With the guard off, every lesson stays in"
- missing:
  - Press release: her next usage panel has one less thing on it. Nothing in the delivery measures or shows this, and no live host capture proves the host no longer cuts the output
    evidence: .abcd/development/intents/shipped/itd-2611041636196213-when-an-agent-goes-wrong-abcd-tells-it.md:19 — "and her next usage panel has one less thing on it."
  - No test exercises the status block in the whole-output bound
    evidence: internal/surface/cli/memory_recall_budget_test.go:113 — "each domain as a whole form. Status has no origin to read here; its block is"
  - No CLI-level test runs the prompt hook with the guard off; OffLine is asserted only on the generated domain
    evidence: internal/core/rules/steering_test.go:115 — "if d.Rules[0] != guard.OffLine {"
    evidence: internal/core/rules/shell_test.go:490 — "if d.Rules[0] != guard.OffLine {"

Scope-condition dispositions:
- cond-2611041649555302 — untested: The delivery changes what the prompt hook sends and leaves the refusal path as it was; nothing in the diff exercises the order of the pre-tool-use hook against the host's own permission prompt, and that order cannot be read from the tree. The `abcd rules` half holds in code through RenderListing, but the assumption the design rests on stays unexercised.
  evidence: hooks/hooks.json:3 — ""UserPromptSubmit": ["
  evidence: internal/core/rules/rules.go:980 — "func RenderListing(domains []ResolvedDomain) string {"
- cond-2611041649558257 — survived: The earlier narrowing no longer holds: the session-start hook now takes its cap from HostOutputCap, so the cap figure has one definition in code and another host's cap needs one edit. The 500 margin is still written separately for each hook, but that is abcd's own margin, not the host's cap. The 10,000 figure itself comes from host documentation outside the tree; nothing in the tree contradicts it, and I could not check it.
  evidence: internal/core/rules/inject.go:35 — "const HostOutputCap = 10_000"
  evidence: internal/surface/cli/hook_checklist.go:26 — "sessionStartOutputCap = rules.HostOutputCap"
  evidence: internal/surface/cli/hook_checklist.go:28 — "sessionStartMargin = 500"
- cond-2611041649557746 — narrowed: Rules for commands sure to run stay steering, and only SHELL fills the second list, so rules checked later, such as at preflight, stay steering. But the warn tier does not guarantee a command runs: five delete entries sit in that tier and are refused in a repository unless proven safe, and the delivery now treats them as refused, as the intent's new decision records. Outside a repository the delete proof does not apply, so those commands warn and run with no rule sent ahead of time.
  narrowing: A warned command is sure to run, and its rule stays steering, only for rm-rf-working-directory, git-reset-hard and git-clean, and for a delete entry a repository rewords. A bundled delete entry runs only once proven safe in a repository, or with no proof outside one, and its rule is not steering in either case.
  evidence: internal/core/guard/teach.go:61 — "if r.Disabled || (e.Tier == TierWarn && (!deleteEntryIDs[id] || own)) {"
  evidence: internal/core/guard/deletes.go:121 — "if r.worktrees == nil {"
  evidence: internal/core/guard/deletes.go:37 — "var deleteEntryIDs = map[string]bool{"
  evidence: .abcd/development/intents/shipped/itd-2611041636196213-when-an-agent-goes-wrong-abcd-tells-it.md:107 — "- A delete entry counts as refused though it sits in the warn tier: the guard refuses it"
<!-- abcd-review-end receipt=rcp-889ac8de461c -->

## Grounds

- pursued: agents behave as they do now on far less text, since text the host cuts changes nothing and the guard already corrects what it refuses; shown wrong if main-session refusals more than double from 110 per 30 days on comparable work
