---
id: itd-2610230756259215
slug: questions-still-break-abcd-s-asking
spec_id: spc-2610301917245442
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610030810350727, itd-201, itd-2610030810370060]
severity: minor
related_issues: [iss-2610230745597968, iss-2610301855200679]
origin: extracted-from-record
production_mode: hand-written
impact: additive
---

# Every question abcd asks passes its check before the person sees it

## Press Release

> _Written by the technical facilitator from the diagnosis of iss-2610230745597968 (planning preparation, 2026-10-23), and extended on 2026-10-30 with the question-writing agent of iss-2610301855200679 at the product thinker's request. Settled under the product thinker's delegation of 2026-10-30 (dcn-2610301913469955), which leaves only design choices with an obvious trade-off to them._

> **An abcd question now reaches the person on its first try.** Before an
> agent puts a question to Iris, the product thinker, it hands the draft to
> abcd, which checks it with the same rules the question check enforces at
> the moment of asking. abcd answers in one go: every part that breaks a
> rule, grouped by rule across every tab, how many rows each tab takes in a
> narrow window, and the question ready to ask. Checking changes nothing:
> the status line still says whom the session is waiting on, and no
> question is marked as open.
>
> For example, a draft about one criterion comes back as "tab 2 is 34 rows
> (header and frame 8, question text 16, options 10); the limit is 32". The
> drafter shortens one option's description, checks again, gets 31 rows and
> no faults, and the question appears. Iris sees one question that fits
> their window, with the thing being decided first and the question last,
> and never sees the redrafting behind it.
>
> An agent that skips the drafter and is refused gets the whole list at
> once: "no Decide later option, tabs 1 to 4; no Now: or Change later:
> line, tabs 1 to 4", and so on for each rule, with abcd's check named as
> the way to get it right. It fixes everything in one retry, instead of
> guessing at the faults a shortened refusal left out.
>
> "I used to watch the session sit for ten minutes while something counted
> lines," said Nia, a technical facilitator who runs abcd's interviews.
> "Now the drafter asks abcd, and the question is up before I have read
> the brief again."
>
> The drafter becomes abcd's question-writing agent. It does not hand back a
> question that has not passed: It writes, checks, fixes every fault and
> checks again until abcd's check is clean, and only then returns. A question
> too long for one screen is never cut and never sent over the limit: The
> writer splits it into numbered parts, Q1a and then Q1b, each one a checked
> question; when there are too many answers for one screen, Q1a offers the
> first of them and "None of these", which leads on to Q1b with the rest. A
> question's header is only its number (Q1, Q1a), because the status line
> already says whom it is for. The agent running the interview asks only what
> the writer returns and never writes a question of its own, so the check's
> refusal is a last resort the person does not meet.

## Why This Matters

The asking rules have been tightened round after round, and each round
changed a rule, a wording or a refusal. None gave the asker a way to check a
draft before asking it. The only exact statement of the layout and the row
count lives in the binary, and the binary applies it in one place: the hook,
after the question has been written.

- itd-201 (shipped) made the asking rules a rule domain injected into every
  managed repository. It is text in the model's context; nothing checks the
  draft against it before the ask.
- itd-2610030810350727 (shipped) fixed one layout and a hook that refuses a
  question breaking it. The refusal comes after the question is written, and
  the record assumed a refusal is enough to fix it.
- iss-2610031236155833 (resolved) calibrated the host's frame and fixed
  abcd's own setup questions with a test. That closed the class for
  questions the binary writes, not for questions an agent composes.
- iss-2610071538055431 (resolved) corrected which rows the refusal named.
  The asker still could not count them.
- iss-2610070637562567 (open) stopped refusing a question that is only too
  tall. It removed a refusal without adding a check, so an over-tall
  question is now shown.
- iss-2610100626327722 (resolved) listed the rows overrun apart from the
  parts to fix. A wording change.
- iss-2610090816296964 (open) added the no-repetition rule and the
  abcd:question-drafter agent, with a lab rewriting until the row count
  "computed exactly by abcd" fits. The drafter shipped; the exact count by
  abcd never did, so the drafter counts by hand. This intent supplies that
  count; the issue stays open for its own rule and lab.
- iss-2610161734179773 (resolved) told agents to redraft silently. It hid the
  retries from the person and left their cause in place.
- iss-2610150551207288 and iss-2610161739587645 (open) each propose one more
  rule held in the model's head. A rule either adds to the check reaches the
  new verb with no change of its own, because both call one function.

On 2026-10-23 the class showed itself twice, on two accounts.

In a repository abcd manages, an agent put four separate issues to the
person as four tabs of one question, without the drafter. The check found 21
faults: headers that were not role chips, no decide-later option, no
`Now:` or `Change later:` lines, and question text that did not give the
thing being decided first. The refusal names at most ten faults and then
says how many it left out, so the agent saw about two tabs' worth and had
to guess at the rest over further retries. The question carried no abcd
chip; it was held to abcd's rules because the status line was waiting on
the product thinker, while the asking rules the agent had read say the chip
and the drafter apply "in abcd's own interviews".

In abcd's own checkout, a question was refused because its text did not end
with the question, and one of its tabs ran 25 rows against a limit of 24,
which the check reports but does not refuse. The same session's drafter
spent nine and a half minutes and about 174 thousand tokens recounting
over-limit tabs with a counting script it wrote itself, because it is told
to count rows by hand and has no way to have abcd count them.

On 2026-10-30 the class showed itself again, in abcd's own checkout. In one
planning interview the drafter returned a refusal for three of ten drafts,
because the quoted material and its options did not fit 24 rows for one
decision and its rules let it give up; the agent running the interview then
put over-limit questions to the product thinker itself, and once wrote a
question without the drafter, which the check blocked in front of the person
(no decide-later option, 28 rows). The product thinker filed
iss-2610301855200679, noting that they had asked for this to be fixed several
times before, and proposed that a sub-agent keep working on a question until
it conforms and only then put it to the person: a question-writing agent
whose rules can be configured, splitting a question that is too long into
parts such as Q1a and Q1b.

Every earlier round left the asker guessing what the check would say. This
intent lets the asker ask the check, and makes the writer keep asking it
until the answer is clean.

## Mechanism

We expect a question drafted through the new check to be admitted on its
first ask, and a refused question to be admitted on its next ask, because
the check and the hook call one function over the same input, so a draft the
check passes cannot be refused by a hook running the same binary, and a
refusal that names every fault at once leaves nothing for the next ask to
discover. Because the writer returns only a draft the check passed, and the agent asks
only what the writer returns, we expect no refusal to reach the person in an
interview run this way. Shown wrong if a question the check passed is then
refused by a hook of the same version, if a drafter run still writes its own
counting script, if an agent given the grouped refusal needs more than one
retry to clear the faults it named, or if a refusal reaches the person in an
interview where every question came from the writer, judged from the
recorded lab interviews of decision 22, since the hook cannot see where a
question came from.

## Scope Conditions

- The host's question tool takes the input the hook already reads: tabs, each with a header, a question text and options carrying a label and a description. A host whose question tool takes another shape is outside the claim. <!-- cond: cond-2610301917242187 -->
- The verb and the hook are the same abcd version. The hook runs the plugin's installed binary; in abcd's own checkout an agent runs the source tree, and a released binary predating the verb has no verb at all. Across versions the verb's verdict is evidence, not a guarantee, and its reported version is what shows the difference. <!-- cond: cond-2610301917242167 -->
- The drafter, or the agent drafting for itself, can run the abcd binary from a shell. A host that gives its sub-agents no shell falls back to the hand count, and the claim does not cover it. <!-- cond: cond-2610301917241788 -->
- Row counts are the check's estimate of the host's view at 80 columns, with the frame as calibrated. A host that changes its frame makes the check and the new verb wrong together, and neither notices it. <!-- cond: cond-2610301917246661 -->
- The claim covers the rules the check can see. Content rules it cannot see, such as bundling separate issues into tabs of one question, repeating the options in the text, or asking a question with one defensible answer, stay with the drafter and the agent. <!-- cond: cond-2610301917246513 -->
- The verdict holds for the state it was judged in. A mode set, or the person marked away, between the check and the ask can change what the hook does; the verb flags every verdict that depends on that state. <!-- cond: cond-2610301917245587 -->
- An agent that uses neither the drafter nor the new verb is reached only through the refusal, so the one-retry claim for that agent rests on the grouped refusal alone. <!-- cond: cond-2610301917243787 -->

## What's In Scope

- A read-only sub-verb that checks a draft question and reports every fault, the rows of each tab, what the hook would do with it now, the binary's version, and the input ready to ask, wired on the command line and on the plugin's command page.
- Composing the host input from abcd's structured question type, so a draft written in that type comes back with its parts in the layout's order.
- The abcd:question-drafter agent calling that verb after every edit and reporting the verb's row counts and version instead of its own count.
- The asking rules naming the verb where they tell an agent to draft through the drafter or to draft for itself.
- The hook's refusal naming every fault, one line per rule and tab, with no cut, and naming the verb and the drafter.
- The guard chapter of the brief (`04-surfaces/17-guard.md`) gaining the `question` row in its sub-verbs table and the verb's exit rules beside `guard check`'s, landed by the build in the same change as the verb.

- The drafter as abcd's question-writing agent: It loops (write, run the check, fix every fault) until the check is clean, at most five checks to a part, and returns only then; its refusals go to the agent that called it, never to the person (decision 32).
- Splitting a question that does not fit into numbered parts (Q1a, Q1b, and so on), as tabs when the parts are independent decisions and as successive questions when a part depends on the answer before it, each passing the check (decision 33).
- Headers that carry only the question's number, such as Q1 or Q1a, with no role word: The agent sets the status line to the person the question is for before asking it, and the answer sets it back (decision 37).
- Only a question whose header is in abcd's number grammar is held to abcd's rules; any other question runs unchecked whatever the status line says (decision 36).
- The asking rules telling the agent that runs an interview to ask only what the writer returns and never to write or edit a question itself.

## What's Out of Scope

- Rewriting a refused question inside the hook: the check refuses and never rewrites, so it never puts words in the agent's mouth.
- Tightening the structured type's own check so a missing `Now:` or `Change later:` line is refused there (decision 14).
- Content rules the check cannot see (bundled issues, repetition, one-answer questions); iss-2610090816296964 and iss-2610161739587645 keep them.
- The required "Unclear" option of iss-2610150551207288.
- Re-calibrating the host's frame or abcd's default rows limit, which itd-2610301914350541 raised from 24 to 32. A repository's own limits are that intent, which builds on this one.
- Refusing a question only for its height: iss-2610070637562567's ruling stands, and the verb reports the overrun as a fault while the hook still shows it.

## Acceptance Criteria

_Seeded by the technical facilitator (planning preparation, 2026-10-23), extended on 2026-10-30 with the question-writing agent, and accepted under the product thinker's delegation of 2026-10-30 (dcn-2610301913469955; decision 30)._

Seen by a person:

- **Given** an agent in one of abcd's interviews drafting a question for the
  product thinker through the drafter, **when** the question is asked,
  **then** the product thinker sees it with no refusal line before it.
- **Given** a question drafted through the drafter, **when** the product
  thinker reads it in a window 80 columns wide, **then** no tab is cut off.
- **Given** an agent that skipped the drafter and was refused for faults in
  several tabs, **when** it asks again, **then** the person sees no second
  refusal line before the question appears.
- **Given** the technical facilitator holding a draft question, **when** they
  run abcd's check on it by hand, **then** they read every fault grouped by
  rule, each tab's rows, and whether the hook would let it through.
- **Given** the status line names the product thinker, **when** the technical
  facilitator checks a draft by hand, **then** the status line still names
  the product thinker afterwards.

- **Given** a question whose material does not fit one screen, **when** the
  writer prepares it, **then** the person sees numbered parts, Q1a and then
  Q1b, each fitting a window 80 columns wide, as tabs of one screen when the
  parts are independent and one after another when a part depends on the
  answer before it, with no refusal line before any of them.
- **Given** a question with more answers than fit one screen, **when** the
  person sees Q1a, **then** it offers the first answers and "None of these",
  and choosing "None of these" brings Q1b with the rest and "Decide later".
- **Given** any question abcd asks through the host's question tool, **when**
  the person reads its header,
  **then** the header is only its number, such as Q1 or Q1a, and the status
  line names whom it is for.

Technical, as one list:

- **Given** a draft in the host question tool's input, **when**
  `abcd guard question` reads it from a file or standard input, **then** it
  exits 0 when the draft has no finding at all.
- **Given** a draft with any finding, a tab over the rows limit alone
  included, **when** the verb judges it, **then** it exits 1.
- **Given** an input that is not JSON, exceeds the hook's size bound, or has
  neither shape, **when** the verb reads it, **then** it exits 2 and names the
  shape it expected.
- **Given** a draft in abcd's structured question type, **when** the verb runs
  with `--ask`, **then** it judges the draft as composed into the host input.
- **Given** a managed fixture checkout with the mode on the product thinker
  and no question open, **when** the verb runs on any draft, **then** the
  mode file, the open-question marker and every other file in the checkout
  are byte-identical afterwards, and no file is created.
- **Given** a fixture corpus (the 21-fault four-tab case, the 25-row case,
  one fault per rule, chipped and unchipped drafts under each mode state, the
  person marked away, and each store the hook cannot read), **when** the verb
  and `abcd guard hook` of the same build judge the same input in the same
  checkout state, **then** the verb's findings and its `hook` outcome equal
  the hook's own decision.
- **Given** the person marked away, **when** the verb judges any draft,
  **then** its `hook` outcome is `refuse-away`.
- **Given** a mode or away store the verb cannot read, **when** it judges a
  draft, **then** its `hook` outcome is `unchecked` and the result is marked
  partial, naming each store it could not read and each rule it could not
  apply.
- **Given** a partial result with no finding among the rules it could apply,
  **when** the verb exits, **then** it exits 2, never 0.
- **Given** a draft whose header is not in abcd's number grammar, **when**
  the verb judges it, **then** it reports the draft is not abcd's question and
  is not checked, whatever the mode.
- **Given** the 25-row case, **when** the verb reports rows, **then** that tab
  reads 8 for the header and frame, 7 for the question text, 10 for the
  options and 25 in total, the split the check itself computes.
- **Given** the 21-fault case, **when** the verb runs, **then** its JSON lists
  all 21 findings.
- **Given** the 21-fault case, **when** the verb groups its findings, **then**
  each group is one rule and one tab.
- **Given** the 21-fault case, **when** the hook refuses it, **then** the deny
  names every refusing rule on every tab it hits and carries no "and N more"
  line.
- **Given** any refusal, **when** the hook writes it, **then** it names
  `abcd guard question` and the abcd:question-drafter agent.
- **Given** a payload of thousands of tabs and options, **when** the hook
  refuses it, **then** the deny stays within a fixed bound of one line per
  rule and checked tab plus its head, why and pointer lines.
- **Given** a tab whose question text holds thousands of list lines carrying
  an em dash, **when** the hook refuses it, **then** that rule takes one line
  for the tab, echoing one offending line and counting the rest.
- **Given** the same findings, **when** the verb's text render and the hook's
  deny are produced, **then** their finding lines are byte-identical, because
  one renderer writes both.
- **Given** a structured draft carrying material, `Now:`, `Change later:`,
  an ask and a decide-later option, **when** the verb composes it with
  `--ask`, **then** the composed text gives the material first, the `Now:`
  and `Change later:` lines next and the question last, and the decide-later
  option is the last option.
- **Given** a structured draft missing its `Now:` line, **when** the verb
  composes it, **then** it reports the now-and-change-later finding rather
  than inventing the line.
- **Given** a host-shaped draft carrying keys the check does not read, such as
  `multiSelect`, **when** the verb admits it, **then** `ready` is the input's
  bytes unchanged.
- **Given** the verb, **when** it reports, **then** the JSON carries the
  binary's version.
- **Given** the drafter's agent page, **when** `make asking-sync-check` runs,
  **then** the generated block tells the drafter to chip every draft, run the
  verb after every edit, and return the verb's rows, ready input and version.
- **Given** a drafter whose verb call exits 2, **when** it answers, **then** it
  returns its refusal form naming the fault and never falls back to the hand
  count.
- **Given** a drafter whose host gives it no shell, **when** it answers,
  **then** its output says its rows were counted by hand.
- **Given** abcd's own checkout before a release carries the verb, **when**
  the drafter checks a draft, **then** it runs the verb from the source tree
  and records the version it reports.
- **Given** a managed repository whose installed binary predates the verb,
  **when** the drafter answers, **then** its output says the draft is checked
  only by the hook on the ask, and it does not count rows by hand.
- **Given** the GRILL domain as generated, **when** it renders, **then** the
  drafter rule and the rule for a host with no agents both name
  `abcd guard question`, it tells the agent running an interview to ask only
  what the writer returns, its header rule names the number form, and the
  "in abcd's own interviews" scope words are unchanged.
- **Given** a managed lab repository on a plugin binary carrying the verb,
  **when** the verb runs there, **then** it works without abcd's own checkout.
- **Given** the 25-row material, in a lab repository whose
  `.abcd/config.json` sets `question.rows` to 24 (since itd-2610301914350541
  the default is 32, which the material fits), **when** each of three
  recorded lab runs of the drafter prepares the question, **then** in every run the drafter writes
  no file and calls the verb at most three times.
- **Given** a managed lab repository, **when** each of three recorded live
  sessions asks the 21-fault question and is refused, **then** in every
  session the second ask is admitted.
- **Given** the verb tree, **when** the plugin's guard command page and the
  generated command-line reference are read, **then** both document
  `abcd guard question`, and its help line follows the "Writes nothing;
  refuses …" form.
- **Given** the brief's guard chapter, **when** the change that adds the verb
  lands, **then** its sub-verbs table carries a `question` row, its exit rules
  state the verb's exit codes beside `guard check`'s, and record-lint's
  surface coverage passes.

- **Given** a draft the check refuses, **when** the writer works on it,
  **then** it runs the check after every fix, at most five times to a part,
  and returns either a draft on which the check exits 0 or, once the five are
  spent, its refusal carrying the check's last findings, to the calling agent
  only; it never refuses for length.
- **Given** a writer's refusal, **when** the calling agent receives it,
  **then** it re-briefs the writer (shorter material, or a further split) and
  asks the person nothing until a draft passes.
- **Given** a header made of `Q`, a number and an optional part letter (`Q1`,
  `Q1a`, `Q2/5`), **when** the check reads it, **then** it is abcd's question
  and is held to the asking rules; a header in the older form with a role
  word (`Product Q2`) is still read as abcd's, so no agent asking today is
  refused, and the writer writes only the number form.
- **Given** the status line names the product thinker, **when** a question
  whose header is not in that grammar is asked, **then** the hook lets it
  through unchecked.
- **Given** a question with a number header, **when** the hook judges its
  register, **then** it follows the person the status line names.
- **Given** the material the drafter refused for length on 2026-10-30,
  **when** each of three recorded lab runs of the writer prepares it,
  **then** every run returns parts on which the check exits 0, and none
  returns a refusal.
- **Given** a managed lab repository, **when** each of three recorded
  interviews of at least ten questions runs through the writer, **then** the
  hook refuses none of the questions asked.
## Decisions

Each decision below was taken on 2026-10-23 by the technical facilitator on the product thinker's delegation, in planning preparation, and revised the same day after a design and feasibility review and a record-discipline review. The product thinker reviews them at the planning interview.

1. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The verb is `abcd guard question`, a sub-verb of `guard`, because the question check already lives in the guard (`abcd guard hook` judges the question tool's payload) and `guard check` is the read-only sibling for shell commands; a new top-level verb would add a family for one act, and overloading `guard check` would put two input grammars behind one positional.
2. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: One decision function, lifted out of the hook into `internal/core/question`, takes the field view, the mode state and the away state and returns the outcome, the addressee, the findings and the row split; the hook and the verb both call it, and the store reads that feed it stay one helper in `internal/surface/cli` both use, because two callers of one function cannot drift apart as two copies of the same rules would.
3. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The away state is read inside that path, so the outcome set is `admit`, `admit-with-rows-note`, `refuse`, `refuse-away`, `admit-unchecked` (a question whose header is not in abcd's grammar, in any mode, per decision 36) and `unchecked` (each fail-open path of the hook), and an `unchecked` result is marked `partial` with `unread` naming each store the verb could not read and the rules it therefore could not apply, because the hook refuses every question while the person is away before any limit is checked, and a verb that omitted it would report admission the hook then refuses.
4. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: Input is the host question tool's input (`{"questions": [...]}`, decoded as the hook decodes it) from a file or `-` for standard input, or abcd's structured question type with `--ask` (decoded strictly, since abcd owns that schema), read under the hook's size bound; the flag chooses the form and nothing is guessed, because both shapes open with a `questions` key and a wrong guess reports the wrong faults.
5. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The verb reads the mode and away stores and never writes them, so its `hook` outcome is true for that checkout at that moment; every result that depends on either store is marked `depends_on_state`, because a mode set between the check and the ask can turn an unchecked admission into a refusal.
6. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The JSON (`_type` `abcd/question-check/v1`) carries `version`, `hook`, `depends_on_state`, `addressee` per tab with an `addressee_changes` note, `findings` (every finding with tab, part, rule, value, limit, remedy and whether it refuses), `by_rule` (one entry per rule and tab, with one echoed value and a count), `rows` (per tab: `header_and_frame`, `question_text`, `options`, `total`, the shape the drafter already returns) with `rows_limit`, and `ready`; the drafter copies fields rather than translating them.
7. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: `ready` is the input's own bytes for a host-shaped draft and the composition through `Fields` for a structured one, present unless the hook would refuse, because rebuilding host input from the decoded view drops keys the check does not read, such as `multiSelect`, which the host still needs.
8. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: Exit 0 when there is no finding at all and every store was read, 1 on any finding including a rows overrun alone, 2 when the input or any store that decides which rules apply cannot be read, a partial result included even when it found nothing, so a check that could not run in full never reads as clean (invariant 16); this departs from `guard check`, which exits 0 on a warning, because a warned command is meant to run while the verb answers whether a draft is ready, and the hook shows an over-tall question only because the person may already be answering it, which a draft never is.
9. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The verb writes nothing, neither a mode nor an open-question marker, a branch record or a log, because a check that marked a question open would leave the status line naming a person for a question never asked, which is why `guard hook` is no stand-in.
10. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The verb reports its binary version and the drafter records it, and the "never refused" claim holds only between a verb and a hook of the same version, because the hook runs the plugin's installed binary while an agent in abcd's own checkout runs the source tree, and a released binary predating the verb has none.
11. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The drafter labels every draft with its number header (decision 37), runs the verb after every edit through the plugin's binary resolution, and returns the verb's `ready` as its `questions`, its `rows` and its `version`, with `checked_by` set to `abcd`; the hand count stays in its page only for a host that gives the drafter no shell, with `checked_by` set to `hand`, and an exit 2 from the verb is returned as the drafter's refusal and never triggers the hand count, because a failed check read as a reason to count by hand would restore the script this intent removes.
12. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The refusal names every fault as one line per rule and tab, each with one echoed value and a count of the rest, then one line naming `abcd guard question` and the drafter, and the ten-part cut goes; the bound becomes the rules times the checked tabs (the check stops at the tab and option ceilings), because rule 12 adds one finding per matching line of text and so grows with the payload, which one line per rule and tab does not.
13. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: One renderer in `internal/core/question` writes the grouped finding lines, and both the deny text and the verb's text render use it, so the words an agent is refused with and the words the verb shows cannot drift.
14. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The structured type's own check is not tightened; composition guarantees the order of the parts it carries, and a missing `Now:` or `Change later:` line or decide-later label is still named by the limits check, because `Fields` leaves a missing line out on purpose so the gap is reported rather than invented, and tightening `Check` would change what abcd's setup interview and the plain-Terminal runner accept, which is a separate change made after checking those callers.
15. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: A fix that adds or changes a chip can bring a second refusal, since a `Product` chip turns on the register rule; the verb shows the new addressee and its findings before the ask, which is where that second round now happens.
16. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: No filled layout skeleton is appended to the refusal, because the `--ask` composition gives the order of the parts by construction and the refusal stays a list of faults.
17. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation (revised 2026-10-30 by decisions 35 to 37, which add a sentence that the agent running an interview asks only what the writer returns, and move the chip rules to the number form and the status line): The GRILL text changes in two sentences only, the drafter rule (it runs `abcd guard question` on the draft and returns what the check counted) and the fallback for a host with no agents (which checks its own draft with the same verb before asking); the "in abcd's own interviews" words and the line the refusal adds for an unchipped question wait on the open question, because changing either now would settle it by the back door.
18. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The verb is wired on the command line, documented on the plugin's guard command page, in the generated command-line reference and in the brief's guard chapter (its sub-verbs row and its exit rules, landed by the build in the same change, never ahead of it), and generated into the drafter page by `make asking-sync`, because a verb lands with its brief row (spec-moves-with-the-surface) and a verb that is not wired is not done.
19. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: This intent resolves iss-2610230745597968 and, since 2026-10-30, iss-2610301855200679 when it ships; it supplies the exact count iss-2610090816296964's lab assumed without resolving that issue, so that issue is named in prose and not in `related_issues`, because a join there would be one-sided until the issue names this intent back and would read as unresolved when this intent ships.
20. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The three intents this builds on (itd-2610030810350727, itd-201, itd-2610030810370060) are written into `builds_on` through `abcd intent edge`, never by hand.
21. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: Before a release carries the verb, the drafter in abcd's own checkout runs it from the source tree, as every abcd call there does, and records the version it reports; in a repository whose installed binary has no `guard question`, the drafter says its draft is checked only by the hook on the ask and counts nothing by hand, because a hand count is the cause this intent removes, and the hook's refusal is the check that binary still has. Only a host that gives the drafter no shell counts by hand.
22. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: The criteria that rest on a live model run (the drafter's file writes and verb calls, and the second ask admitted) are judged by three recorded lab runs on one pinned binary and one fixed material, each transcript kept in the lab's record as the evidence; a criterion passes only when every run passes, and one failing run fails it, because a live run varies between attempts and a repeatable lab procedure is what a later run can check again, which a unit test with a fake model cannot show.
23. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: itd-2610030810350727 is refined, not kept beside: this intent changes that intent's refusal (no ten-part cut, grouped by rule and tab, naming the verb and the drafter) and lifts its hook's decision into a shared function, while its layout and rules stay as they are.
24. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: itd-2610030810370060 is kept beside this intent: its structured question type and composition already exist in the code, so the `--ask` form consumes them without waiting on its open spec.
25. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: itd-200 is kept beside this intent: the verb only reads the mode and never sets it, so the status line itd-200 promises is unchanged whichever way the open question is answered.
26. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: itd-2609292108373494 is kept beside this intent: handing the grill to an installed skill is a different capability; the open question's answer decides whether that skill's unchipped questions are checked, and is the fact both records cite.
27. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: itd-169 is kept beside this intent: itd-169 decides when and to whom a stopping loop speaks, this intent whether the question it asks passes the check, and neither needs the other's scope.
28. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: itd-90 is kept beside this intent: this check covers only the rules a deterministic check can see, while which questions reach the product thinker at all is itd-90's judgement.
29. 2026-10-23, the technical facilitator on the product thinker's delegation, planning preparation: itd-201's relation to this intent stays open with the open question: whether this intent only builds on it or reverses its decision 10 depends on the product thinker's answer. Resolved 2026-10-30 by decision 36: It builds on itd-201, and decision 10 stands.

Taken on 2026-10-30 by the technical facilitator under the product thinker's
delegation of the same day (dcn-2610301913469955: the technical facilitator
settles what is sensible, and only real design trade-offs go to the product
thinker), when iss-2610301855200679
was folded in:

30. iss-2610301855200679 is planned inside this intent, not as an intent of its own, because this draft is not planned yet, its scope already has the drafter call the check after every edit, and the issue extends exactly that loop; two plans would share one surface (the drafter page and the GRILL text) and one would wait on the other. The issue is joined both ways (`related_issues`, `related_intents`), and this intent resolves it when it ships (decision 19). The criteria above, the 2026-10-23 ones included, are accepted under the same delegation, recorded as dcn-2610301913469955; the open question was put to the product thinker (decision 36).
31. The agent keeps its name, `abcd:question-drafter`, and becomes the question-writing agent by contract, because the issue's remedy asks to turn the drafter into that agent, not to add a second one, and a rename would oblige retiring the old name (retire-the-name) across every interview page and generated block for no change in what it does; its page describes it as abcd's question-writing agent.
32. The writer loops until the check exits 0, at most five checks to a part (the lab criterion's three plus a margin), because a model's fixes can swing between two rules and an unreadable store makes exit 0 unreachable, and the record bounds every agent loop (the drain's fix rounds). Its refusals have three causes, each returned to the calling agent and never to the person: a missing or contradictory part of its brief, the check's exit 2 (decision 11), and five checks spent, which carries the check's last findings. The calling agent re-briefs and asks nothing until a draft passes. Where the binary has no check or the host gives the writer no shell (decision 21 and the scope conditions), the writer cannot loop, and the claim that no refusal reaches the person does not cover that case.
33. A question that does not fit is split into numbered parts, Q1a and Q1b, in one of two ways, settled by whether a part depends on the answer before it. Parts that do not depend on each other are separate decisions, such as the criteria of one feature, and are asked as tabs of one question, up to four to a screen, as the asking rules already allow. A part that depends on the earlier answer is asked as its own question after it, never as a tab, because a tab shows every part before any is answered and the asking rules ask a dependent question alone, after that answer. Too many answers for one screen are the dependent case: Q1a offers the first answers and "None of these", which leads on to Q1b with the rest and the decide-later answer, and "Decide later" anywhere ends the question; no grouping is invented, since asking rule 2 forbids made-up options. One decision whose quoted material alone is too long cannot be spread over parts without asking the same choice twice, so the writer quotes it shorter and faithfully, keeping every answer's gain and cost, or, where the material holds several decisions, splits it into those decisions. The part letter is there for the person's reading order, and the header grammar admits it (decision 37).
34. Per-repository layout limits are their own draft, itd-2610301914350541, which builds on this intent, because both reviews of 2026-10-30 found them a promise that can be built and checked on its own (the split-the-intent doctrine), touching the rules loader, the hook, the generated pages and the configuration schema; that draft holds the one list of the rules that stay fixed.
35. The agent running an interview asks only what the writer returns and never writes or edits a question itself, stated in the generated asking rules. This half is taught, not gated: Nothing in the question tool's input can show where a question came from, and the verb writes no receipt (decisions 7 and 9), so it is judged from the recorded lab interviews (decision 22), and the hook's refusal stays the last resort for an agent that ignores the rule.

Ruled by the product thinker on 2026-10-30:

36. **Only abcd's label makes a question abcd's.** A question whose header is not in abcd's number grammar runs unchecked whatever the status line says; the status line no longer turns an unlabelled question into abcd's. Chosen over the status line deciding, at the cost that an abcd question whose label was forgotten goes through unchecked, which decision 35's rule and the writer, who always labels, carry. This answers the open question, keeps itd-201's decision 10 as written (decision 29 becomes builds-on), and binds brief invariant 1 to every question abcd asks, each of which carries the label; the change that lands this rewords the invariant to say so.
37. **A question's header is only its number**, such as Q1, Q1a or Q2/5, never a role word, because the status line already tells the person whether a question is for the product thinker or the technical facilitator. The agent sets the status line to the addressee before it asks (`abcd mode product-thinker` or `abcd mode facilitator`) and the answer sets it back; the register rule follows the mode; the header grammar becomes `Q`, a number, an optional part letter and an optional total. The older form with a role word (`Product Q2`) stays readable as abcd's, settled without a question because refusing it would break every agent and page that asks that way today for no gain to the person, who no longer sees it once the writer writes only the number form.

38. Taken on 2026-10-30 under the product thinker's delegation, when the spec was written: The number-only header holds for questions asked through the host's question tool, where the status line names the addressee. The interviews abcd draws itself in a plain Terminal (setup, the written interviews and the connect guide) have no status line, so their header keeps its role word, the only place there that says whom a question is for. Decision 36 also settles two earlier items: the outcome `admit-unchecked` now means any question outside abcd's grammar in any mode (decision 3), and the refusal line decision 17 held back for an unchipped question goes, since that path is gone.

## Open Questions

_All resolved: The one open question (which questions the status line makes abcd's) was answered by the product thinker on 2026-10-30, decision 36._

## Typed Links

- **refines `itd-2610030810350727`** (one layout and its check): the verb runs that check before the ask, and the refusal's form changes (decision 23); the `builds_on` edge stays, since the check must exist first.
- **builds on `itd-201`** (the asking rules as a domain): two of its generated sentences change (decision 17).
- **built on by `itd-2610301914350541`** (a repository sets the layout limits of abcd's questions): It makes the check's limits per-repository, with fixed bounds.
- **builds on `itd-2610030810370060`** (interviews in a plain Terminal): the `--ask` form reuses its structured question type and its composition into the host input.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect one command that checks a draft question exactly as the hook would to end the repeated refusals, because the drafter and agents would see every fault and the true row count before asking instead of counting by hand; shown wrong if a draft the command passed is later refused, or if a drafter run still writes its own counting script
- pursued: the product thinker proposed that a question-writing agent keep working on a question until it passes abcd's check and only then put it to the person; we expect no refusal to reach the person in an interview run through the writer, and we are wrong if the recorded lab interviews still show a question refused in front of the person, or if the writer hands back a refusal for length.
