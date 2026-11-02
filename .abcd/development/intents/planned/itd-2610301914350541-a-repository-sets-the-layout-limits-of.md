---
id: itd-2610301914350541
slug: a-repository-sets-the-layout-limits-of
spec_id: spc-2610301950037469
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610230756259215]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# A repository sets the layout limits of abcd's questions

## Press Release

> **A repository now sets how tall abcd's questions may be, how long an
> option's label may run, and how much a description may say.** A team whose
> people read in taller windows, or want a label of a few more words, sets
> three values once in the repository's configuration: rows to a question,
> words to a label, and sentences to a description. abcd's check, its
> question-writing agent, and the asking rules it teaches all use those
> values, and report them, so a question that passes the check fits the
> window it was set for. The rules that protect the person stay fixed in
> every repository: the thing being decided is quoted in full, there is
> always a decide-later answer, no option is marked as recommended, and a
> value cannot be set so wide or so narrow that it switches one of them off.
>
> "Our screens are forty rows, and every question came split in two for a
> limit written for twenty-four," said Nia, a technical facilitator who runs
> abcd's interviews. "I set rows to forty in the repository, and the next
> question came whole, with the check telling me which file set the limit."

## Why This Matters

Split out of itd-2610230756259215 on 2026-10-30, when both pre-planning
reviews of that intent found that per-repository limits are a promise that
can be built and checked on its own (the split-the-intent doctrine). This
intent builds on it, because the limits are read by the check that intent
adds.

iss-2610301855200679 asks for a question-writing agent whose "layout rules
are configurable per repository, with abcd's rules as the default", and its
refinement of the same day narrows that to rules "configured where a
repository needs it". Today every limit is one value in the binary,
`question.Default` (`internal/core/question/limits.go`), which the hook, the
asking rules, and the generated page blocks all read.

What the 2026-10-30 design review found, for the spec to settle:

- The hook and the asking-rules domain can read a repository's values at run
  time (the rules loader already layers repository overrides), but the page
  blocks `make asking-sync` writes are fixed at build time and ship with the
  plugin, so they can only state abcd's defaults and point at the check, which
  reports the repository's.
- The text widths the row estimate uses are measured from the host's frame,
  not chosen, so a configurable column count would make the estimate wrong;
  only rows, label words, and description sentences are configurable (the
  options count is not, for the reason decision 2 gives).
- A limit with no bounds can switch a protection off with a legal value (a
  thousand rows, or a label the width of a description), so each configurable
  value has a floor and a ceiling, and a value outside them is refused naming
  the key.

## Mechanism

- We expect the check, the hook, and the asking rules to state and apply the
  same limits in any one repository on any one machine, because all three
  take them from one resolver, which reads the repository's three values over
  abcd's defaults, lets the person's own file tighten the result and never
  loosen it, and hands a value outside its bounds, or a personal value looser
  than the limit in force, back as a refusal instead of passing it on. Shown
  wrong if, in one repository on one machine and one abcd version, the
  figures the injected GRILL domain states and the `limits` field `abcd guard
  question` reports ever differ.

## Scope Conditions

- The check counts rows at 80 columns inside a host frame of eight rows, <!-- cond: cond-2610301950032668 -->
  whatever the window is, because the three text measures are calibrated from
  the host's frame at that width and the frame is not configurable. A host
  whose frame grows with the window makes the estimate wrong at wider values,
  and the check does not notice.
- The repository is a checkout whose root `layered.RootsFor` resolves, with <!-- cond: cond-2610301950039896 -->
  `.abcd/config.json` at that root. Outside a repository, in one with no such
  file, and in one whose file has no `question` namespace, the limits are
  abcd's defaults, tightened only where the person's own
  `~/.abcd.noindex/config.json` sets a key below them (decision 5).
- The verb, the hook, and the rules loader run the same abcd version, as <!-- cond: cond-2610301950032451 -->
  itd-2610230756259215's claim does. A binary predating this intent claims no
  `question` namespace: it applies the defaults without a word, and the only
  sign is a check result with no `limits` field.
- The values apply to the questions the hook holds to abcd's rules (a header <!-- cond: cond-2610301950035338 -->
  in abcd's number grammar, itd-2610230756259215 decision 36) and to the
  questions the binary composes itself. Another tool's question runs
  unchecked in every repository, whatever the values.

## Acceptance Criteria

_Written by the technical facilitator on 2026-10-30 and accepted under the product thinker's delegation of the same day (dcn-2610301913469955, which leaves only a design choice with an obvious trade-off to the product thinker)._

Seen by a person:

- **Given** a repository whose configuration sets rows to 40, **when** the
  product thinker reads a question the writer prepared there in a window 80
  columns wide and 40 rows tall, **then** no tab is cut off.
- **Given** any repository, whatever values it set, **when** the product
  thinker reads a question abcd asks, **then** the thing being decided is
  quoted in full before the question, a `Now:` line and a `Change later:`
  line stand between them, the last option is "Decide later" or "None of
  these", and no option is marked, styled, or ordered as recommended.
- **Given** the technical facilitator sets `question.rows` to 100 in the
  repository's configuration, **when** the next abcd question is drafted or
  asked, **then** they read a refusal naming `question.rows`, the value 100,
  the bounds, and the file, and the product thinker sees no question until
  the value is fixed.
- **Given** the technical facilitator runs the check by hand in a repository
  that set its values, **when** they read its result, **then** it states the
  three values in force and that the repository's configuration set them.
- **Given** a repository whose configuration sets no value, on a machine
  whose own file sets none, **when** a question is asked there, **then** the
  person sees what they see today but for one figure: the rows limit is 32
  rather than 24 (decision 3), the asking rules state thirty-two, every other
  limit and rule is unchanged, and no configuration is mentioned.
- **Given** a person whose own `~/.abcd.noindex/config.json` sets
  `question.rows` to 24 in a repository that set 40, **when** they read a
  question abcd asks on their machine, **then** it fits 24 rows, and a peer
  on a machine that set nothing reads the same question at the repository's
  40.

Technical, as one list:

- **Given** `.abcd/config.json` carrying `question.rows`,
  `question.label_words`, and `question.description_sentences`, **when** the
  resolver reads it, **then** it returns `question.Default` with `Rows`,
  `LabelWords`, and `MeaningSentences` replaced by those values and every
  other field unchanged, and the hook and `abcd guard question` pass that
  value to `CheckLimits`.
- **Given** a key under `question` the resolver does not claim, such as
  `question.options`, `question.columns`, or `question.tabs`, **when** it
  reads the file, **then** it refuses naming the key and the file, and
  nothing falls through to the default.
- **Given** a `question` key set in `~/.abcd.noindex/config.json` at or below
  the limit otherwise in force (the repository's value where
  `.abcd/config.json` sets the key, else the bundled default), **when** the
  resolver reads it, **then** the personal value is the limit in force for
  that key, and `limits_source` names the machine file for it.
- **Given** a `question` key set in `~/.abcd.noindex/config.json` above the
  limit otherwise in force, **when** the resolver reads it, **then** it
  refuses naming the key, the personal value, the limit in force, and the
  file that set each, and nothing falls through: a personal value tightens
  and never loosens (decision 5).
- **Given** each key at its floor, at its ceiling, one below the floor, one
  above the ceiling, and as a non-integer, in either file, **when** the
  resolver reads each, **then** the floor and the ceiling are accepted and
  the other three are refused, each refusal naming the key, the value it
  found, and the bounds; a personal value is judged by the bounds before it
  is judged against the limit in force.
- **Given** a value outside its bounds and a question whose header is in
  abcd's number grammar, **when** the hook judges it, **then** the deny names
  the key, the value, the bounds, and the file, names no asking rule, and a
  question whose header is not in the grammar runs unchecked.
- **Given** a value outside its bounds, **when** `abcd guard question` judges
  any draft, **then** it exits 2, its JSON carries `refused` naming the key,
  the value, and the bounds, and it carries no `ready`.
- **Given** a value outside its bounds, **when** the rules loader builds the
  GRILL domain for that repository, **then** the layout rule states abcd's
  defaults and says which key is refused and that the check refuses every
  abcd question until it is fixed, the loader's notes carry the same line,
  `abcd rules` prints it, and every other domain loads as before.
- **Given** a repository with values inside their bounds, **when** the GRILL
  domain is injected there, **then** its layout rule states those figures in
  words and names `.abcd/config.json` as having set them, and the machine
  file for any key a personal value tightened; **given** no value set in
  either file, the rule reads as it does today at the new default and names
  no file.
- **Given** a repository with values inside their bounds, **when** `abcd
  rules --json` renders the GRILL domain there, **then** its `source` is
  `bundled` while its layout rule's text names `.abcd/config.json`, and
  `abcd rules` prints its heading with no override label.
- **Given** the same repository, **when** `abcd guard question` reports,
  **then** its JSON carries `limits` with the three values and
  `limits_source`, one entry per key (`bundled`, the repository file's path,
  or the machine file's path where a personal value tightened it),
  `rows_limit` equals `limits.rows`, and the figures in `limits` equal the
  figures the injected layout rule states, held by a test that renders both
  from one fixture checkout.
- **Given** a change to a `question` value during a session, **when** the
  next prompt that recalls GRILL arrives, **then** the domain is injected
  again with the new figures, because its text, and so its signature,
  changed.
- **Given** the writer's agent page, **when** `make asking-sync-check` runs,
  **then** its generated block states abcd's defaults as the defaults, names
  the three keys and their bounds, tells the writer to take the limits in
  force from the check's `limits` field and never from the configuration
  file, and the planning page's block carries the same sentence.
- **Given** a repository whose configuration sets rows to 40, **when** each
  of three recorded lab runs of the writer prepares the 25-row material of
  itd-2610230756259215, **then** every run returns one part the check passes
  at 40 rows, reports `limits` as the check gave them, and reads
  `.abcd/config.json` in no run.
- **Given** a writer whose host gives it no shell, **when** it answers in a
  repository that set its values, **then** its output says it counted by
  hand at abcd's defaults and that neither the repository's values nor the
  person's were read.
- **Given** every combination of the three keys at their floors and
  ceilings, **when** a question with one paragraph of material, both lines,
  two options, and a decide-later option is checked, **then** it is admitted
  in every combination, and **given** the same question without its
  decide-later option, with an option marked recommended, without material
  before the question, or without its `Now:` line, **then** it is refused in
  every combination.
- **Given** every question the binary composes itself (the setup plan, each
  setup value and confirm question, each category approval, the source
  question, the README and sessions questions, the connect guide's
  questions, and the written interview's sample), **when** each is checked
  at the floor of every key, **then** every one is admitted whole, held by a
  test beside the one that holds them to the defaults today.
- **Given** a `.abcd/config.json` the resolver cannot read (a symbolic link,
  not a regular file, or one it cannot parse), **when** the hook judges an abcd question,
  **then** the question is checked at abcd's defaults and the gate says, in
  its `NOT CHECKED` form, that the file was not read, and `abcd guard
  question` exits 2 with a partial result naming the file.
- **Given** a `~/.abcd.noindex/config.json` the resolver cannot read (behind
  a symlinked `~/.abcd.noindex`, writable by group or other, or owned by
  another account), holding no `question` key, **when** the hook judges an
  abcd question, **then** the question is checked at abcd's defaults and the
  gate names that file, and `abcd guard question` exits 2 naming it; **given** a symlinked
  `~/.abcd.noindex` holding no `config.json`, **then** the limits are the
  defaults and nothing is said.
- **Given** a repository whose configuration sets rows to 40, **when** a tab
  of 30 rows is asked, **then** the hook admits it with no rows note and the
  verb reports no rows finding, and a tab of 41 rows gets the note and the
  finding naming 40 as the limit.
- **Given** a call of five tabs, **when** it is judged in a repository that
  set every value at its ceiling, **then** it is refused on the tabs rule as
  today, because the tabs to a call are not configurable.
- **Given** the tree, **when** a test lists the call sites that pass
  `question.Default` to `CheckLimits`, `AskingRules`, `ChipRole`, or the
  asking-sync renderers, **then** the only ones are the resolver, the
  asking-sync command (build time, by decision 12), `rules.Load`'s one-line
  body, and tests; every other caller passes the resolved limits.
- **Given** the bounds, **when** they are read, **then** they are stated once
  in `internal/core/question` beside `Default`, and the resolver's refusal,
  the loader's note, and the generated page sentence all render them from
  there.
- **Given** the brief, **when** the change that lands the keys is read,
  **then** the configuration chapter (`05-internals/03-configuration.md`)
  lists the three keys with their defaults and bounds under the `question`
  namespace and says which layer sets each and that the machine file may
  only tighten, the guard chapter's `question` row names the exit 2 for a
  refused value, the plugin's guard command page says where the limits are
  set and that the check reports them, and `docs-lint` and record-lint's
  surface coverage pass.

## Decisions

On the evening of 2026-10-30 the product thinker asked for every draft of
that day to be planned ready for implementation. dcn-2610301913469955 of the
same day covers the routing, the review findings the record settles, and the
criteria of an intent they asked to plan; the decisions below were taken
under it by the technical facilitator, from the findings of the design review
that split this intent out and of the two adversarial reviews that followed.
The bounds (3), the layering (5), the absence of a writing verb (13), and the
rows floor (17) were flagged provisional for the product thinker, who ruled
on them at the planning interview of 2026-10-31 (decision 18); none is
provisional now.

1. **The keys.** A repository sets three values in `.abcd/config.json` under
   one namespace: `question.rows`, `question.label_words`, and
   `question.description_sentences`. The resolver claims the `question`
   namespace and those three keys through the layered stack, so a misspelt
   or unknown key (`question.options`, `question.columns`) is refused naming
   it rather than ignored. Reason: The layered stack is the one canonical
   primitive for a value more than one party may set, and its contract
   already spells the guarded read, the duplicate-key refusal, and the
   no-fall-through rule this intent needs; a second reader of
   `.abcd/config.json` would restate them.
2. **What stays the binary's.** Columns (80) and the three measured host
   figures (76, 74, and 8), because the row estimate is calibrated against
   the host's frame at that width and a chosen number would make it wrong;
   the header's width and grammar, because the header is the label that
   makes a question abcd's; the tabs to a call (one to four), because a tab
   is its own screen, so it is not a limit of one screen's layout, and the
   host's tool caps it at four; the options to a question (two to four, the
   decide-later option included), because the host's tool lists no more than
   four and a ceiling below it could only split the binary's own setup
   question, which a plain Terminal has no four-option tool to split for
   (decision 16 holds the detail); the `Now:` and `Change later:` prefixes,
   the decide-later labels, and the role words, because the check, the hook,
   and every page match on them by text.
3. **The bounds**, inclusive, stated once in `internal/core/question` beside
   `Default` and rendered from there wherever they are named. Settled on
   2026-10-31 without a question, because the record already settles them:
   the floors are derived from what the binary composes, and the ceilings
   from where the protection stops protecting (decision 18). A limit with no
   bounds can switch a protection off with a legal value, so each key has a
   floor and a ceiling, and the floor of each is drawn so that every question
   the binary composes today is admitted whole at it. The floor and the
   default are different things: the floor is the least a team may set, and
   the default is what a question may take where nobody sets anything, so
   the default may sit above the floor and for rows it does. The smallest question that obeys every
   fixed rule is eighteen rows: the frame's eight, four lines of text (one
   line of material, the `Now:` line, the `Change later:` line, and the
   question), and three options at two rows each (a label row and a
   one-sentence description), the decide-later option among them; composed
   through abcd's structured type, which joins the material, the two lines,
   and the question with blank lines, it is twenty. Sixteen rows is one
   answer beside "Decide later", which the check's floor of two options
   admits but the asking rules forbid, since a question is put only where
   two or more answers are each defensible.
   - `question.rows`: 24 to 60, default 32. The floor is the standard
     screen and the narrowest window abcd promises; every question the
     binary composes is held to it today, and nothing in the record asks
     for a shorter question (decision 17 holds the one trade-off). The
     default rises from 24 to 32 (`question.Default` in
     `internal/core/question/limits.go` is 24 today; decision 18): 32 rows
     of an 80-column window is what a question may take where nobody sets
     anything, and a team whose screens are the standard 24 sets the floor.
     The ceiling is a full-height terminal on a large display; past it,
     "fits one screen" protects no screen a person has, and the writer would
     stop splitting.
   - `question.label_words`: 4 to 8, default 5, unchanged. The floor is the
     longest label the
     binary composes, four words: "Yes, make the change"
     (`internal/core/ahoy/setup_question.go:184`), "Yes, make the changes"
     (`internal/core/ahoy/category_approval.go:35`), "Look up the list" and
     "Type the name myself" (`internal/core/oracle/connect_guide.go:455`
     and `:456`), and "It needs a key" (`connect_guide.go:511`); "None of
     these" is three. Past eight words a label of ordinary words fills its
     row at the option measure and wraps, and reads as a description.
   - `question.description_sentences`: 2 to 4, default 2, unchanged. The
     floor is abcd's default and the longest description the binary
     composes, two sentences (the
     connect guide at `connect_guide.go:455` and `:511`, the source question
     at `internal/core/source/add.go:62` to `:67`, and the decide-later
     meanings of the setup and written interviews); the check counts a
     sentence at a stop before white space, so a key such as `scan.deep`
     ends none. Past four, a description holds material, and material
     belongs in the question text, where the thing decided is quoted in
     full.
   Every combination inside the bounds admits a question that passes every
   fixed rule, and a test holds it (the criteria above).
4. **The rules that stay fixed in every repository**, the one list this
   intent holds for itd-2610230756259215 (its decision 34): The thing being
   decided is quoted in full before the question, and the text ends with
   the question; a `Now:` line and a `Change later:` line stand between
   them, each with a value; the last option is a decide-later or
   none-of-these option, exactly one; no option is marked, styled, or
   ordered as recommended; the addressee's register holds (the product
   thinker sees no record number and no command); no bold, no side preview,
   no em dash in a list item; one question is held to the rows limit in
   force, whatever that limit is. Reason: These are the rules that protect
   the person rather than fit the window, and the bounds of decision 3 are
   drawn so that no value reaches one of them.
5. **The repository sets the limit; a person's own file may only tighten
   it.** Ruled on 2026-10-31 (decision 18). The limit otherwise in force for
   a key is the repository's value where `.abcd/config.json` sets it, else
   the bundled default. A `question` key in `~/.abcd.noindex/config.json` at
   or below that limit applies on that person's checks; one above it is
   refused naming the key, the personal value, the limit in force, and the
   file that set each, and nothing falls through. The bounds of decision 3
   hold for a personal value as for a repository's, and are judged first.
   This is neither `interview.list`'s shape
   (`internal/core/layered/interview.go:22`, the machine layer alone) nor the
   plain layering every other key has, where the repository's value simply
   wins. Reason: The project's value wins in every direction that could
   loosen it, as every other abcd setting's project value wins, so a
   personal value can never pass a question the team's limit would refuse;
   and a person whose window is smaller than the team's can still have
   questions that fit it, because the hook and the check run on the asking
   person's own machine, which is the screen the question is shown on. The
   consequence, stated plainly: a question can split differently on two
   machines when one person has tightened, and the check reports which file
   set each figure so the difference is readable.
6. **One resolver, outside the rules loader.** The resolver is one function
   in `internal/core/layered`, beside `InterviewList`, taking the `Roots`
   that `layered.RootsFor(cwd)` builds and returning one of three states:
   the defaults (no key set), the limits in force with the file that set
   them, or a refusal naming the key, the value, the bounds, and the file.
   It cannot live in `internal/core/rules`, because `layered` imports
   `rules` (`layered.go:64`) and Go admits no cycle, so the loader takes the
   limits as an argument instead: `rules.LoadWith(repoRoot, limits, note)`
   generates the GRILL domain from the limits handed to it and writes the
   note into the layout rule and its notes, and `rules.Load(repoRoot)` stays
   as `LoadWith` with `question.Default` and no note, the form every test
   and every caller outside a repository keeps. The two front doors that
   load the rules, the prompt hook (`cli.go:1865`) and `abcd rules`
   (`cli.go:2646`), resolve the limits first and pass them in, as the
   repository's guard registry already reaches the loader for the SHELL
   domain (`withRepoShellDomain`, `rules.go:252`). The hook's question gate,
   the check verb, the plain-Terminal interview's check, and the binary's
   own question composers call the resolver directly; `question.Default`
   stays the one statement of the defaults and the bundled layer the
   resolver reads last, and a test lists the call sites still allowed to
   pass it by name. The root is the one `RootsFor` resolves, the rules
   loader's (`rules.Resolve`, `internal/core/rules/root.go:145`), so a
   monorepo member with its own `.abcd/` reads its own file; that root can
   differ from the mode store's (`mode.Root`, git's toplevel) in such a
   member, and the limits follow the rules root so that the injected domain
   and the hook's check read one file. Reason: itd-2610230756259215
   decision 2 made the verb and the hook one function so they cannot drift;
   a limit read in two places would reopen the drift one layer up, and a
   loader that read the file itself would be that second reader.
7. **A value outside its bounds, of the wrong type, or a personal value that
   loosens, is refused and nothing falls through.** The hook refuses every
   question held to abcd's rules, naming the key, the value, the bounds (or,
   for a loosening personal value, the limit in force), and the file and no
   asking rule, until the file is fixed; `abcd guard question` exits 2 with
   `refused` naming the same, so the writer returns its refusal to the
   calling agent (itd-2610230756259215 decision 11) and the person meets no
   question; the rules loader renders the GRILL domain from abcd's defaults
   with the refusal written into the layout rule and into its notes, and
   loads every other domain. Reason: The layered contract is that a refused
   value never falls through to a lower layer or the default, because a
   configuration that silently does less than it says is the failure it
   exists to close; a check that quietly judged at the defaults would pass a
   question written to figures that are not in force. The asking rules are
   still rendered because most of them are not about limits, and an agent
   reading them learns of the refusal before it drafts.
8. **A file the resolver cannot read is loud, in the hook's own form, and
   leaves the question checked at the defaults.** A `.abcd/config.json` that
   is a symbolic link, not a regular file, or one the resolver cannot parse,
   and a `~/.abcd.noindex/config.json` behind a symlinked
   `~/.abcd.noindex`, writable by group or other, or owned by another
   account, even one holding no `question` key, and an unresolved home, take
   the path the spec's approach 3 names for an unread layer: abcd's question
   is checked at abcd's defaults, as it was before any file could set a
   limit, and the gate writes one line in its `NOT CHECKED` form naming that
   file, on stderr at exit 1 for a question the defaults admit, and at the
   head of the reason or the note otherwise:

   ```text
   abcd guard: NOT CHECKED at this repository's question limits — they were not read (<reason>), so the question is checked at abcd's defaults instead.
   ```

   The verb exits 2 with a partial result naming the file. A symlinked
   `~/.abcd.noindex` holding no `config.json` reads as absent, as the
   home-declaration guard already rules (`internal/fsutil/home.go:560`), and
   a test lays out a dotfiles-symlinked `~/.abcd.noindex` both ways. The
   `layered` package exports the distinction the gate needs, an unreadable
   layer against a value it read and refused, so the hook can tell decision
   7's fail-closed path from this one without reading the error's words.
   Reason: This is the guard's loud contract for a store it cannot read,
   here holding the question to the defaults rather than failing open, and a
   machine file the account cannot vouch for is such a store
   whatever keys it holds, because the stack reads both layers before it
   knows which keys either carries. The defaults are what the gate checked
   at before a file could set a limit, so a file that cannot be read never
   leaves abcd's question less checked than an unconfigured repository's
   (settled on 2026-11-01 in the build of spec step 2).
9. **The check reports the values in force.** Its JSON gains `limits`
   (`rows`, `label_words`, `description_sentences`) and `limits_source`, one
   entry per key (`bundled`, the repository file's path, or the machine
   file's path where a personal value tightened it), and `refused` on
   decision 7's path; `rows_limit` of itd-2610230756259215 decision 6 stays and equals
   `limits.rows`. The three fields are additive to `abcd/question-check/v1`,
   so the type name does not change. The text render states the same line.
   Reason: The writer copies fields rather than translating them, and the
   person running the check by hand reads where a figure came from.
10. **The writer takes the limits from the check, never from the file.**
    Its page tells it to use `limits` from every check result and to write
    to those figures; the hand count a host with no shell leaves it is made
    at abcd's defaults, and its output says that neither the repository's
    values nor the person's were read, so the claim does not cover either
    reaching such a writer. Reason: A writer that opened `.abcd/config.json`
    or the person's file would be a second reader with its own parsing, and
    an agent's reading of a file is not the check's; the no-shell fallback
    counts at the defaults because they are the figures its page states
    (decision 12), the only ones such a writer has, and the hook on the ask
    still judges the question at the limits in force.
11. **The GRILL domain keeps its bundled provenance**, and its layout rule
    names `.abcd/config.json` when a value came from there, and the machine
    file when a personal value tightened one. Reason: The
    `(repo override)` label means the repository replaced a domain's rules
    through `.abcd/rules.json`, which a repository may still do; setting a
    figure the generated rule states is a different act, and the rule's own
    text is where the figure's origin is read. The dedup ledger re-injects
    the domain on the next prompt that recalls GRILL once its text changes,
    so a changed value reaches that prompt with no new mechanism.
12. **The shipped page blocks state abcd's defaults and point at the
    check.** `make asking-sync` keeps rendering from `question.Default`, and
    each block gains one sentence: these figures are abcd's defaults, a
    repository sets rows, label words, and description sentences in
    `.abcd/config.json` under `question` within the bounds it names, and
    `abcd guard question` reports the values in force. Reason: The blocks
    are written at build time and ship with the plugin, so they cannot state
    a repository's values; a block that stated figures as absolute would
    teach an agent in a configured repository the wrong ones.
13. **No verb writes the values.** Settled on 2026-10-31 without a question,
    because the record already settles it: a writer can be added when a team
    asks for one (decision 18). The keys are edited by hand, as `pace.*` and `interview.list`
    are; no verb writes them, `abcd ahoy install` asks no question about
    them, and `abcd lint` gains no rule for them in this intent. Reason: The
    read is where a wrong value is met, at the first question, and it is
    refused there naming the key; a writer or a lint rule is a separate
    surface the record can add when a team asks for it.
14. **Documentation lands with the change.** The brief's configuration
    chapter lists the keys, defaults, and bounds; the guard chapter's
    `question` row (added by itd-2610230756259215) gains the exit 2 for a
    refused value; the plugin's guard command page says where the limits
    are set and that the check reports them. Reason: The spec moves with
    the surface, as itd-2610230756259215 decision 18 applies it.
15. **Relation to itd-2610230756259215.** This intent builds on it and
    changes one figure it states, the default rows, which rises from 24 to
    32 (decision 3); everything else it promises at the defaults holds: the
    check's JSON gains three additive fields (decision 9) and one more cause
    of exit 2, a refused value, beside the parent's unreadable input and
    unread store; the hook gains one read before `CheckLimits`; and every
    criterion of that intent stated at twenty-four rows holds at thirty-two
    once this intent lands, in a repository that sets no value, on a machine
    that tightens none. The parent's lines that name 24 as the limit in
    force are edited when this intent lands, in the change that moves the
    default. The parent spec's verb step (spc-2610301917245442, its
    JSON approach) must list `limits`, `limits_source`, and `refused` and
    the exit 2 for a refused value, so that the verb's contract is stated
    once; this intent's spec names that edit. Only the three keys are
    configurable; every other figure the check reads (columns, the host's
    three measures, the header's width, the tabs to a call, and the options
    to a question) stays the binary's, and a repository that needs one of
    them changed is outside this intent.
16. **Out of scope.** A configurable options count (`question.options`):
    with the only bounds that keep two defensible answers beside "Decide
    later", three to four, the key's one effect would be to split the
    binary's own setup question (`internal/core/ahoy/setup_plan.go:190` to
    `:195`, "Proceed", "Customise", "Cancel", and "Decide later"), whose
    second part would offer "Cancel" beside "Decide later", a
    recommendation by shape, and a plain Terminal has no four-option tool to
    split for; a later intent that redesigns the setup question could take
    the key up. Also out: configurable columns or host measures (the
    review's second finding); a personal value that loosens a limit
    (decision 5); configurable tabs, header width, prefixes, or labels (decision 2); a
    verb or an install question that writes the keys (decision 13);
    re-calibrating the host's frame; and what itd-2610230756259215 keeps out
    of its own scope.
17. **The rows floor stays 24, the standard screen.** Taken on 2026-10-30
    under the product thinker's delegation while they were away, and settled
    by their ruling of 2026-10-31 (decision 18): a team cannot set questions
    shorter than a standard screen. Every question the binary composes
    already fits 24, so none of abcd's own questions has to change. The
    alternative floor of 18, the least a question obeying every fixed rule
    can take (decision 3), was put to them and not taken.
18. **The planning interview's rulings of 2026-10-31.** The product thinker
    ruled on the items flagged provisional. The rows floor stays 24, the
    standard screen, so a team cannot set a question shorter than one
    (decision 17; the open question is closed). abcd's bundled default for
    `question.rows` rises from 24 to 32, the other two defaults stay, and
    the bounds stay 24 to 60, 4 to 8, and 2 to 4 (decision 3). A person's
    own `~/.abcd.noindex/config.json` may set the three keys, but only to
    tighten the limit otherwise in force, the repository's value or else the
    default; a looser personal value is refused naming the key, the personal
    value, the limit in force, and the file that set each, and the bounds
    hold for it too (decision 5, rewritten from the repository-only layer
    the draft held). The bounds of decision 3 and the absence of a writing
    verb (decision 13) were not put as questions and stand as recorded,
    because the record already settles them: the floors are derived from
    what the binary composes, and a writer can be added when a team asks.

Typed links: builds on itd-2610230756259215 (every question abcd asks passes
its check before the person sees it), whose check, hook, and GRILL text read
the limits this intent makes per-repository; that intent names this one in
its own typed links as built on by it.

## Open Questions

_None open. The one question deferred here, whether a team may set questions
shorter than the standard screen with a `question.rows` floor of 18, was put
to the product thinker on 2026-10-31 and answered: the floor stays 24
(decisions 17 and 18)._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: Recorded by the technical facilitator under the product thinker's delegation of 2026-10-30 (dcn-2610301913469955), and reviewed by the product thinker at the planning interview of 2026-10-31 (decision 18). We expect a repository's own values for rows, label words, and description sentences, tightened and never loosened by a person's own file, read through one resolver by the check, the hook, the asking rules, and the writer, to let a team fit abcd's questions to its windows without any person-protecting rule becoming reachable; shown wrong if a question the check passed is cut in a window of the configured height, if the injected asking rules and the check ever state different figures in one repository, or if any value inside its bounds admits a question without a decide-later option, with a recommended option, or without the thing decided quoted first.
