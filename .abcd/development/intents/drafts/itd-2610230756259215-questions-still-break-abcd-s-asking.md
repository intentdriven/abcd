---
id: itd-2610230756259215
slug: questions-still-break-abcd-s-asking
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2610230745597968]
origin: extracted-from-record
production_mode: hand-written
---

# A question is checked by abcd before it is asked

## Press Release

> _Written by the technical facilitator from the diagnosis of iss-2610230745597968 (planning prep, 2026-10-23). Not yet confirmed by the product thinker; step 3 of the planning interview confirms or refines it._

> **An abcd question now reaches the person on its first try.** Before an
> agent puts a question to Iris, the product thinker, it hands the draft to
> abcd, which checks it with the same rules the question check enforces at
> the moment of asking. abcd answers in one go: every part that breaks a
> rule, grouped by rule across every tab, how many rows each tab takes in a
> narrow window, and the question ready to ask. Checking changes nothing:
> the status line still says whom the session is waiting on, and no
> question is marked as open.
>
> For example, a draft about one criterion comes back as "tab 2 is 25 rows
> (header and frame 8, question text 7, options 10); the limit is 24". The
> drafter shortens one option's description, checks again, gets 23 rows and
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
  abcd never did, so the drafter counts by hand.
- iss-2610161734179773 (resolved) told agents to redraft silently. It hid the
  retries from the person and left their cause in place.
- iss-2610150551207288 and iss-2610161739587645 (open) each propose one more
  rule held in the model's head.

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

Every earlier round left the asker guessing what the check would say. This
intent lets the asker ask the check.

## Mechanism

We expect a question drafted through the new check to be admitted on its
first ask, and a refused question to be admitted on its next ask, because
the check and the hook call one function over the same input, so a draft the
check passes cannot be refused by the hook, and a refusal that names every
fault at once leaves nothing for the next ask to discover. Shown wrong if a
question the check passed is then refused by the hook, if a drafter run
still writes its own counting script, or if an agent given the grouped
refusal needs more than one retry to clear the faults it named.

## Scope Conditions

- The host's question tool takes the input the hook already reads: tabs, each with a header, a question text and options carrying a label and a description. A host whose question tool takes another shape is outside the claim.
- The drafter, or the agent drafting for itself, can run the abcd binary from a shell. A host whose sub-agents have no shell falls back to the hand count, and the claim does not cover it.
- Row counts are the check's estimate of the host's view at 80 columns, with the frame as calibrated. A host that changes its frame makes the check and the new verb wrong together, and neither notices it.
- The claim covers the rules the check can see. Content rules it cannot see, such as bundling separate issues into tabs of one question, repeating the options in the text, or asking a question with one defensible answer, stay with the drafter and the agent.
- An agent that uses neither the drafter nor the new verb is reached only through the refusal, so the one-retry claim for that agent rests on the grouped refusal alone.

## What's In Scope

- A read-only sub-verb that checks a draft question and reports every fault, the rows of each tab, what the hook would do with it now, and the input ready to ask, wired on the command line and on the plugin's command page.
- Composing the host input from abcd's structured question type, so a draft written in that type comes back in the layout by construction.
- The abcd:question-drafter agent calling that verb after every edit and reporting the verb's row counts instead of its own.
- The asking rules naming the verb where they tell an agent to draft through the drafter or to draft for itself.
- The hook's refusal naming every fault, grouped by rule across all tabs, with no cut, and naming the verb and the drafter.

## What's Out of Scope

- Which questions the status line makes abcd's when they carry no chip: the open question below. The hook keeps its present behaviour until it is answered.
- Rewriting a refused question inside the hook: the check refuses and never rewrites, so it never puts words in the agent's mouth.
- Content rules the check cannot see (bundled issues, repetition, one-answer questions); iss-2610090816296964 and iss-2610161739587645 keep them.
- The required "Unclear" option of iss-2610150551207288: when it is built into the check, the new verb reports it with no change of its own.
- Re-calibrating the host's frame or the 24-row limit.
- Refusing a question only for its height: iss-2610070637562567's ruling stands, and the verb reports the overrun as a fault while the hook still shows it.

## Acceptance Criteria

_Seeded by the technical facilitator (planning prep, 2026-10-23) and unconfirmed. Step 8 of the planning interview walks the person-visible criteria one at a time as stories, and the technical criteria as one list to accept, edit or strike._

Seen by a person:

- **Given** an agent in one of abcd's interviews drafting a question for the
  product thinker through the drafter, **when** the question is asked,
  **then** the product thinker sees the question with no refusal line before
  it, and every tab fits 24 rows at 80 columns.
- **Given** an agent that skipped the drafter and was refused for faults in
  several tabs, **when** it asks again, **then** the person sees at most the
  one refusal line before the question appears.
- **Given** the material of the 2026-10-23 case in abcd's own checkout,
  **when** the drafter prepares the question, **then** it is ready within two
  minutes and without the drafter writing any file of its own.
- **Given** the technical facilitator holding a draft question, **when** they
  run abcd's check on it by hand, **then** they read every fault grouped by
  rule, each tab's rows, and whether the hook would let it through, and the
  status line reads as it did before.

Technical, as one list:

- **Given** a draft in the host question tool's input, **when**
  `abcd guard question` reads it from a file or standard input, **then** it
  exits 0 when the draft has no finding at all, 1 when it has any finding
  (one the hook refuses on, or a tab over the rows limit), and 2 when the
  input cannot be read; with `--ask` it reads abcd's structured question type
  instead, and an input in neither shape exits 2 naming what it expected.
- **Given** a managed fixture checkout with the mode on the product thinker
  and no question open, **when** the verb runs on any draft, **then** the
  mode file, the open-question marker and every other file in the checkout
  are byte-identical afterwards, and no file is created.
- **Given** a fixture corpus (the 21-fault four-tab case, the 25-row case,
  one fault per rule, and chipped and unchipped drafts under each mode
  state), **when** the verb and `abcd guard hook` judge the same input in the
  same checkout state, **then** the verb's findings and its report of the
  hook's decision equal the hook's own, because one function computes both.
- **Given** the 25-row case, **when** the verb reports rows, **then** that tab
  reads 8 for the header and frame, 7 for the question text, 10 for the
  options and 25 in total, the split the check itself computes.
- **Given** the 21-fault case, **when** the verb runs, **then** its JSON lists
  all 21 findings and groups them by rule, each group naming every tab it
  hits.
- **Given** the 21-fault case, **when** the hook refuses it, **then** the deny
  names every refusing rule and every tab, carries no "and N more" line,
  names `abcd guard question` and the drafter; **and given** a payload of
  thousands of tabs and options, the deny stays within a fixed bound of one
  line per rule plus its head, why and pointer lines.
- **Given** any structurally valid draft in the structured type, **when** the
  verb composes it with `--ask`, **then** the composed input carries no
  header, `Now:`/`Change later:`, thing-first or decide-later finding, and it
  is the `ready` input the verb returns.
- **Given** the drafter's agent page, **when** `make asking-sync-check` runs,
  **then** the generated block tells the drafter to run the verb after every
  edit and to return the verb's rows and ready input, keeps the hand count
  only as the fallback where the binary cannot run, and the drafter's output
  says which of the two produced its rows.
- **Given** the GRILL domain as generated, **when** it renders, **then** the
  drafter rule and the rule for a host with no agents both name
  `abcd guard question`, and the "in abcd's own interviews" scope words are
  unchanged.
- **Given** a managed lab repository on the released plugin binary, **when**
  the verb and the drafter run there, **then** they work without abcd's own
  checkout; a live lab run of the drafter on the 25-row material writes no
  script and makes at most three calls to the verb, and a live session asking
  the 21-fault question is admitted on its second ask.
- **Given** the verb tree, **when** the plugin's guard command page and the
  generated command-line reference are read, **then** both document
  `abcd guard question`, and its help line follows the "Writes nothing;
  refuses …" form.

## Decisions

1. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the verb is `abcd guard question`, a sub-verb of `guard` — the question check already lives in the guard (`abcd guard hook` judges the question tool's payload), `guard check` is the read-only sibling for shell commands, and a new top-level `question` verb would add a verb family for one act. Extending `guard check` itself was set aside because its positional is a shell command line and overloading its grammar would blur two inputs.
2. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): one canonical decision function, lifted out of the hook, computes the addressee, the findings, the row split and the hook's decision; the hook and the verb both call it — two callers of one function cannot drift apart, which two implementations of the same rules would.
3. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): input is the host question tool's input (`{"questions": [...]}`, decoded as the hook decodes it, unknown keys ignored) from a file or `-` for standard input, and abcd's structured question type with `--ask` (decoded strictly, since abcd owns that schema); the form is chosen by the flag, never guessed — both shapes open with a `questions` key, and a guess that picks the wrong one reports the wrong faults. Input is read under the hook's own size bound.
4. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the verb holds a draft to the same addressee the hook would use at that moment, reading the mode store without writing it — so its report of the hook's decision is true for that checkout now, and a draft with no chip under a managed mode is reported as one the hook would let through unchecked, with its findings still listed.
5. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the JSON (`_type` `abcd/question-check/v1`) carries `hook` (`admit`, `admit-with-rows-note`, `refuse`, or `admit-unchecked`), `findings` (every finding with tab, part, rule, value, limit, remedy and whether it refuses), `by_rule` (each rule with the tabs it hits and its remedy), `rows` (per tab: `header_and_frame`, `question_text`, `options`, `total`, the shape the drafter already returns) with `rows_limit`, and `ready` (the host input to ask, present unless the hook would refuse); the text render prints the same grouped lines the refusal carries — the drafter copies fields rather than translating them, and a person reading the CLI sees the words the agent sees.
6. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): exit 0 when there is no finding at all, 1 on any finding including a rows overrun, 2 when the input cannot be judged — `guard check`'s convention, so a caller never reads silence as clearance, and a draft over the rows limit is not ready for a drafter whose promise is to fit.
7. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the verb writes nothing — no mode, no open-question marker, no branch record, no log — because a check that marks a question open would leave the status line naming a person for a question never asked (the reason `guard hook` is not a safe stand-in).
8. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the drafter runs the verb on its draft after every edit, through the plugin's binary resolution, returns the verb's `ready` as its `questions` and the verb's `rows` as its `rows`, and adds `checked_by` (`abcd` or `hand`); the hand-count section stays in its page only as the fallback for a host where the binary cannot run — the verb removes the cause of the counting script, and `checked_by` makes a fallback run visible rather than silent.
9. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the refusal names every fault, one line per rule naming every tab it hits and each tab's value where they differ, then one line naming `abcd guard question` and the drafter; the ten-part cut (`maxRefusalParts`) goes — the check already stops at the tab and option ceilings, so findings are bounded by the limits and not by the payload, which keeps the bound review-askGuard-security finding 1 asked for; a test over a payload of thousands of tabs and options proves the bound.
10. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): no filled layout skeleton is appended to the refusal — the verb's `--ask` composition gives the layout by construction, and the refusal stays a list of faults.
11. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the GRILL text changes in two sentences only, the drafter rule ("it runs abcd's check on the draft, `abcd guard question`, and returns what the check counted") and the fallback for a host with no agents (which checks its own draft with the same verb before asking); the "in abcd's own interviews" scope words and the line the refusal adds for an unchipped question wait on the open question below — changing either now would settle that question by the back door.
12. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): the verb is wired on the command line, documented on the plugin's guard command page and in the generated command-line reference, and generated into the drafter page by `make asking-sync` — wired or it isn't done.
13. 2026-10-23, the technical facilitator on the product thinker's delegation (planning prep, abcd-43 for the coordinating session): this intent resolves iss-2610230745597968 when it ships, and supplies the exact count iss-2610090816296964's lab was waiting on without resolving that issue, whose no-repetition rule and tier lab stay its own.

## Open Questions

1. **When the status line is waiting on a person, is any question asked
   then held to abcd's asking rules, or only a question that carries abcd's
   chip?** For the product thinker.

   Today the hook holds both. The guard's command page says: "A question is
   abcd's when a header is in abcd's chip grammar (a role word, then `Q` and
   the question's number, such as `Product Q2`) or when `/abcd:mode` names
   somebody; any other question belongs to another tool and runs
   unchecked." A refused unchipped question is told: "This question carries
   no abcd chip: it is treated as abcd's because the mode names the product
   thinker; another tool's question asked now is held to abcd's rules."
   The asking rules the agent reads, meanwhile, scope the chip and the
   drafter to "abcd's own interviews", and itd-201's decision 10 scopes the
   mode gate to abcd's own interviews too.

   Example: in a repository abcd manages, an agent hands the product
   thinker a step to run, so the status line reads "waiting on the product
   thinker". Later the agent asks the same person, with no chip, which of
   two test names they prefer. Today that question is refused until it has
   a chip, a decide-later option and `Now:`/`Change later:` lines.

   - **The status line alone makes it abcd's (keep today's behaviour, and
     make the asking rules say so).** Every question asked while someone is
     waited on looks the same and is checked, so a forgotten chip cannot
     slip an abcd question past the check. The cost is that an ordinary or
     another tool's question asked at that moment must follow abcd's layout
     too, and agents need to be told so in the rules they read.
   - **Only a chip makes it abcd's.** An unchipped question runs unchecked
     whatever the status line says, matching the "own interviews" wording.
     The cost is that an abcd question whose chip was forgotten goes
     through unchecked, so the chip rule carries the whole weight.

## Typed Links

- **builds on `itd-2610030810350727`** (one layout and its check): the verb runs that check before the ask.
- **builds on `itd-201`** (the asking rules as a domain): two of its generated sentences change (decision 11).
- **builds on `itd-2610030810370060`** (interviews in a plain Terminal): the `--ask` form reuses its structured question type and its composition into the host input.
- **refines `iss-2610090816296964`**: supplies the exact count by abcd its lab assumed; the issue stays open for its own rule and lab.
- **keeps both with `iss-2610150551207288`**: a rule added to the check reaches the verb through the shared function.
- **flagged for the product thinker to confirm, not classified**: the open question may reverse or restate itd-201's decision 10; neither answer is recorded as a reversal until it is given.
- Typed edges in the frontmatter are added by the record verbs at planning, not by hand.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect one command that checks a draft question exactly as the hook would to end the repeated refusals, because the drafter and agents would see every fault and the true row count before asking instead of counting by hand; shown wrong if a draft the command passed is later refused, or if a drafter run still writes its own counting script
