---
name: intent
description: Press-release intent lifecycle — status, quoted-text create, the implement-readiness gate, and the human planning interview that turns a draft into a planned, specced intent.
argument-hint: "[text] [--title \"<title>\"] | ready <itd-N> [--grounds \"<pursued|deferred|declined>: <conjecture>\"] | plan <itd-N> [--impact <additive|breaking|fix>] | hold <itd-N> --reason \"<text>\" | unhold <itd-N> | link <itd-N> <spc-N> | audit [<itd-N>] | audit --owed [--max <n>] | audit --issue-drift [--strict] | condition <itd-N> [<cond-id> --disposition <survived|narrowed|falsified|untested> --occasioned-by <rdi-N|itd-N> --grounds \"<why>\" [--narrowing \"<what now holds>\"]]"
---

# `/abcd:intent` — intent lifecycle

The write side of the intent record store under `.abcd/development/intents/`.
Every intent gets a stable `itd-N` id and directory-as-truth lifecycle state
(`drafts/`, `planned/`, `shipped/`, `disciplines/`, `superseded/`). Bare
invocation **performs zero writes**.

## Status (bare)

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent --json
```

Summarise the JSON for the user: counts per bucket, open/closed spec counts,
the intent↔spec links, and `reviews_owed` — the shipped intents whose fidelity
review is owed, the same total bare `intent audit` lists (below). Nothing is
created or moved by this invocation.

**Every `intent` verb addresses the checkout's store, from anywhere in the
tree.** The verb resolves the repository root before it reads or writes, so the
counts are the checkout's and a reported `path` is relative to that root, not to
the directory you happen to be standing in. Outside a repository there is no
intent store to address, and the verb refuses (exit 2) rather than reading an
empty one or laying a new one where it stands — a draft filed outside every
checkout is committed by nothing, read by nothing, and its spec can never be
closed against it. Resolving is a question, not a write, so bare invocation still
performs zero writes. If an intent store also exists below the repository root,
the verb names it on stderr and leaves it alone; relay that line, because records
sitting there reach no gate and no release cut.

**Which ledger?** A half-formed observation, question, or nitpick goes to
`/abcd:capture "…"`; a user-facing change you want to ship goes to
`/abcd:intent "…"`. For a big, unproven idea there is an optional third route:
`/abcd:ideate` runs the admission gauntlet and records the verdict either way.
It is a pointer, never a precondition — filing a draft without it is a normal
thing to do.

## Decompose before filing (itd-84, hand-run)

One proposal is rarely one record. Before running the create command below,
run the itd-84 decomposition as an advisory analysis the human confirms —
capture routes the pieces, it never files a monolith:

1. **Route.** Split the proposal into parts and route each to its record
   home — user-facing capability → an intent; trust-boundary rule → an ADR
   (plus a brief invariant); standing stance → a principle; plumbing → the
   brief. Render the result as a table: part | type | home.
2. **Link.** Surface the existing records the proposal touches, with typed
   links — `supersedes` / `reverses` / `duplicates` / `refines` — never
   "related". A reversal ("this reverses invariant X") is *flagged for the
   human to confirm*, never auto-classified.
3. **Verdict.** Propose one of three outcomes — FILE-AS-IS / SPLIT / HOLD.
   The human adopts the routing; only the part they confirm as an intent
   proceeds to the create command, and the other parts go to their homes in
   the same session (or are captured so they are not lost).
4. **Grade.** Append the confirmed table — and whether the initial routing
   survived the human's confirmation — to the dated decomposition-calibration
   note under the development record's `research/notes/`. That corpus (about
   50 graded captures) is what gates the automated rung.

**Not yet automated.** The deterministic pre-pass and the capture-time
validator are future rungs of the itd-84 discipline; until they ship, this
documented protocol is the gate.

## Create a draft

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent "<text>" [--title "<title>"] [--impact <additive|breaking|fix>] [--production-mode <hand-written|dictated-and-formatted|scribe-transcribed>] --json
```

Files `drafts/itd-N-<slug>.md`. **The text is the press release**: the whole
of it seeds the `## Press Release` section as prose, so write it as the user
moment — the paragraph a shipped intent opens with. The H1 title is the text's
first sentence: the split is at the first `.`, `!` or `?` followed by
whitespace or the end of the text, the terminator is dropped from the title,
and a sentence longer than the slug cap is cut on a word boundary.
`--title "<title>"` replaces it with a heading of your own — one line,
non-empty, redacted like the text. The slug is derived from the text either
way. `## Why This Matters` is seeded with a prompt, not with the text again.
Report the new `id` and `path`, and tell the user the seeded Why This Matters
and Acceptance Criteria sections are placeholders that must be replaced — the
criteria with real Given-When-Then bullets, via the planning interview below —
before the draft can be planned.

A single whitespace-free word is refused (exit 2, nothing written): a lone
token reads as a mistyped sub-verb, never as a draft title. A near-miss of a
real sub-verb is refused the same way, with the correction named. So is a word
followed by a record id — `abcd intent shipit itd-5` is a sub-verb call by
shape whatever the word is, so it is refused whether or not any sub-verb is
close enough to suggest, and a refusal with nothing to suggest lists the
sub-verbs the verb has. What still files is prose: several words, or one quoted
argument carrying a space.

`--impact` is optional: a draft is "not judged yet", so an unset impact writes
no field. When you do set it, the value is validated (one of `additive`,
`breaking`, `fix` — never `internal`, since an intent is user-facing by
definition) and stamped onto the draft, where it travels unchanged through
planning to `shipped/`, which the `intent_impact_valid` gate requires. A draft
filed without one is judged later, at the planning interview — and `abcd intent
plan <itd-N> --impact <value>` is the verb that stamps it then, at the same
bar (see step 10 of the interview). Never hand-edit the field in: the verbs
carry the validators.

## Disclosure: where a record came from and how its text was produced

Intent, spec and issue records carry two frontmatter keys, written by the
commands that mint them, and no flag carries either as free text. Records of
other families carry neither.

`origin` is **derived from which command ran** and has no flag at all. A draft
filed from quoted text is `researcher-authored`; a draft `abcd capture promote
<iss-N>` mints is `extracted-from-record`; a draft `abcd capture promote <rdi-N>`
mints from an accepted reading item is `contributed-by-reading <rdg-N>/<rdi-N>`,
naming the item's run and id, because a reading item is something an instrument
returned rather than something a person noticed. It is stamped at mint and never
rewritten — linking an existing draft with `capture promote --intent` writes the
`related_issues` back-edge and leaves the `origin` where it was.

`production_mode` is the closed choice `--production-mode` carries:
`hand-written`, `dictated-and-formatted`, or `scribe-transcribed`. Any other
value is refused and nothing is written. An absent flag takes the repo's
declared default from `.abcd/config/identity.json`, falling back to
`hand-written`. `abcd intent plan` takes the same flag, which stamps the **spec**
it mints; the intent's own stamp was written when its draft was created and is
never rewritten.

Neither key touches authorship: they are disclosure at field granularity, on the
same footing as the `Assisted-by:` trailer at commit granularity. Population is
forward-only, so a record written before the keys existed carries neither, and
nothing backfills it. The `record_provenance` record-lint rule reports a
record carrying the pair in a shape no write path produces — but a hand edit
that types a legal value in a legal combination is byte-identical to a command's
write, so it catches implausible hand edits, not all of them.

## THE RULE: no implementation without a planned, specced intent

Before implementing ANY `itd-N` — or whenever the user asks you to "build",
"implement", or "work" an intent — run the gate first:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent ready <itd-N> --json
```

- **Exit 0 (ready):** proceed. The linked spec's body is the design record to
  build against.
- **Exit 1 (not ready): DO NOT IMPLEMENT.** Do not improvise acceptance
  criteria, do not write code toward the intent, and do not run
  `abcd intent plan` on your own authority. Tell the user plainly:
  "`<itd-N>` is not specced, so it cannot be implemented yet", present each
  failing check's `detail` and `remedy` from the JSON, and **offer the
  planning interview** below.
- **Exit 2 (fault):** the id is malformed, the intent is unknown, a record is
  unreadable, or the working directory has no repository above it (or one git
  will not answer for), so there is no intent store to address — report the
  diagnostic; there is nothing to gate.

## Grounds: why this is being pursued

The gate also records the CONJECTURE behind the decision, at the granularity of
the thing being pursued rather than of the architecture:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent ready <itd-N> --grounds "pursued: <conjecture>" --json
```

The vocabulary is closed — `pursued`, `deferred`, `declined` — and the text is
free prose. Grounds are recorded on a draft or a planned intent alike, and
refused on a shipped or superseded one. The flag writes one entry to the
record's `## Grounds` section and then reports the gate exactly as it would
without it: the report is unchanged by
the flag, the exit code is the gate's own, and a failed write exits 2 rather than
borrowing the gate's exit 1.

**With the flag, `--json` emits an envelope, not the bare readiness result:**

```json
{ "grounds": { "intent_id": "…", "path": "…", "token": "pursued",
               "text": "…", "entries": 1, "redacted": 0 },
  "ready":   { "…the usual ReadyResult…" } }
```

Read the verdict from `ready`, and report `grounds.path` and `grounds.entries`
so the user knows a record was written. **Report `grounds.redacted` whenever it
is non-zero** — the text is scanned before it is committed, and the user needs to
know their wording was changed. There is no `degraded` member here: a scanner
that cannot be built, or whose pattern set a per-repo override weakened, refuses
the write at exit 2 rather than writing under a weakened detector. Without the
flag the payload is the readiness result unchanged.

The write is also announced before the gate runs — on stdout in the text render,
on stderr under `--json` — as `recorded grounds on <path> (<n> entries)`. Relay
it. Recording is append-only, so a caller who retries after missing the receipt
adds a second entry rather than replacing the first.

**The gate reports a planned record that carries no entry, and does not refuse
it.** The `grounds` check is the eighth and last row of the report and is
advisory: its remedy names this exact command, and the verdict ignores the row
until the rethink of the reading work settles what a human is asked for here
(iss-2609091009111294). Relay the row; do not treat it as a refusal. Terminal buckets are exempt on the same rule the claim checks follow:
`shipped/` and `superseded/` records are never backfilled, and a discipline
record carries no conjecture of its own. The write enforces that rule too: this
verb REFUSES a `shipped/` or `superseded/` record, so no grounds this tool
writes can ever land on one.

That is a statement about the TOOL, not about the corpus, and the difference is
the migration exception. Three shipped intents do carry a `## Grounds` section —
itd-177, itd-182 and itd-188 — relocated by hand from the pre-tooling
`## Grounds (pursued)` section on the matching spec. A relocation is not a
backfill: the text was authored at the moment of pursuit and nothing was
reconstructed. The refusal covers both, deliberately, because nothing in the
enforcement can tell relocated text from invented text — which is why the state
those three records are in is not reachable through this verb.

**Ask for the expectation and its falsifier.** "Planned it because it is next"
restates the decision and records nothing; "planned it because we expect a
stamped identity to survive rewording, which nothing else does" is a conjecture
somebody can later find wrong. abcd refuses only the degenerate texts — empty,
too short, or the vocabulary word repeated back — and cannot tell a conjecture
from a restatement. That part is yours: put the question to the human and write
down their answer, not a paraphrase of the route taken. A hand-typed bullet is
held to the same floor: `- pursued: yes` is not an entry, and the gate reports
the record as carrying none.

Recording is append-only: a second decision on one record adds an entry beside
the first, because the earlier conjecture is what a later reader checks the
outcome against.

## The claim recording gradient

An intent carries up to three kinds of claim, and the gate holds each to its
own recording requirement:

| Claim | Section | Requirement |
| --- | --- | --- |
| Criterion | `## Acceptance Criteria` | Mandatory — at least one Given-When-Then bullet |
| Mechanism | `## Mechanism` | Prompted, nullable — an absent section passes; a heading with nothing under it is named on an advisory row |
| Context | `## Scope Conditions` | Reported — top-level bullets, or the explicit nullity; an absent or malformed section is named on an advisory row and never withholds readiness (iss-2609091009111294) |

The nullity is one exact token, `None stated.`, alone on its line under the
heading — the same grammar for both sections. Three byte states carry three
meanings and are never collapsed: an absent section is a claim not carried, an
empty section is a gate fault, and the token is a claim considered and
declined. Discipline-kind records are exempt: their template carries no claim
sections, and both checks report the exemption.

Each scope condition carries a stamped identity marker
(`<!-- cond: cond-<16 digits> -->`) so a later disposition attaches to the
condition rather than to a sentence that may since have been reworded. The
marker is read anywhere in the bullet, so rewrapping the text cannot orphan it.
Two markers in one bullet, a near-miss of one, a fenced block or an HTML comment
in the section, or a second `## Scope Conditions` heading are each reported by
name. A fenced or commented section and a duplicated heading are refused by the
stamp outright; a bullet carrying two markers, a near-miss, or an identity
another bullet already uses is skipped by the stamp and named by the gate. Either
way the gate never names a remedy that cannot run. **The
markers are stamped by `abcd intent plan`, never hand-typed**, and the gate
refuses a missing or duplicated one by name rather than repairing it — a
reporter that writes is a reporter whose output depends on who ran it. That
remedy runs on a planned record too: `abcd intent plan <itd-N>` on an intent
already in `planned/` does the identity step alone — it mints for every
unmarked bullet, moves no bucket and touches no spec — so a condition written
after planning still reaches the mint. That re-run also takes `--impact`,
under the rules step 10 gives, so a planned record filed without a judgement
gets one before its close through the verb rather than an editor. With nothing
unmarked (and no judgement to add) it refuses and says so, rather than exiting
quietly having done nothing. The
identities are rendered by `abcd intent ready <itd-N> --json` under
`conditions`, which is where a consumer reads them; bare `abcd intent` is a
corpus-wide count-and-link status and carries no per-record body.

Population is forward-only: `shipped/` and `superseded/` records are never
backfilled, because an absent stamp is information — so both checks report as
not applicable there rather than naming work nobody may do. An unanswered
scaffold prompt is reported as unanswered, never as a recorded claim.

## Planning interview (host-run, with the human present)

The interview turns a draft into an intent the maintainer has signed off. Run
it only in a live session with the human; deferral of any question is a valid
answer, but silence is not consent.

**How every question is asked (the GRILL rule domain).** One question at a
time, through the harness's interactive question tool, never as a numbered
list inside prose. Each question carries one sentence of context, one concrete
example of what each answer means in practice, and options that widen rather
than recommend: no starred default, no recommended label, the null answer
always offered. A recommendation the human asks for is given in prose apart
from the question. The next question waits for the last answer. The register follows
the addressee: a product thinker gets outcomes in product terms with no
record ids or internals; a technical facilitator gets the mechanism and the
ids. Where the hat is unknown, that is the first question.

**Prerequisite — two adversarial reviews.** Before the interview, the draft
has been through two independent adversarial reviewers with different lenses
(design/feasibility and record-discipline are the proven pair), and their
surviving findings are applied or explicitly rejected — per
`adversarial-review-scales-with-blast-radius` in the development record's
principles. An unreviewed draft does not reach the interview; the readiness
gate that will refuse the move mechanically is a recorded seed until built.

1. Read the draft record; summarise it back: the press release, why it
   matters, the current Acceptance Criteria (say explicitly when they are
   facilitator- or agent-seeded and unconfirmed), and any open questions.
2. **Decomposition (itd-84):** run the hand-run table above over the draft —
   parts → homes, typed links, advisory reversal flags. A part that is not
   this intent moves to its home (or is captured) before planning proceeds;
   grade the run into the calibration note either way.
3. **Press release:** confirm or refine the user moment with the human.
4. **Open questions:** resolve each with the human, or record an explicit
   deferral in the draft. An open question that gates scope blocks planning.
5. **Mechanism claim (prompted, nullable):** ask why the authors expect this
   to work, and record the answer in `## Mechanism` as a falsifiable "we
   expect X because Y" — not the outcome restated. Declining is a real
   answer: record it as the exact token `None stated.` alone on its line.
   Silence is not a decline, and the draft's scaffold line is not a claim.
6. **Scope conditions (required):** elicit the population, platform, scale, or
   assumptions the claim holds under, one per top-level bullet under
   `## Scope Conditions`, so a later reuse outside them is a visible
   re-decision. If the human states none, record the same exact token. Leave
   the identity markers to `plan` — never type one; a condition added after
   planning is stamped by re-running `abcd intent plan <itd-N>`.
7. **Grounds (required at the gate):** ask why this is being pursued NOW —
   the expectation, and what would show it wrong. Record the human's answer
   with `abcd intent ready <itd-N> --grounds "pursued: <their words>"`. A
   conjecture that is being left for later is `deferred:`; one being turned
   down is `declined:`. Never write the restated decision; if the honest
   answer is "it is next in the queue", say so to the human and ask what they
   expect the work to prove.
8. **Acceptance criteria:** walk EVERY Given-When-Then bullet; the human
   accepts, edits, or strikes each, and adds what is missing. Seeded criteria
   are proposals, never approvals.
9. Edit the draft file to the confirmed content.
10. Only after the human explicitly confirms the criteria are theirs, run:

   ```bash
   "${CLAUDE_PLUGIN_ROOT}/abcd" intent plan <itd-N> [--impact <additive|breaking|fix>] [--production-mode <mode>] --json
   ```

   This invocation IS the maintainer's sign-off act — never run it unattended
   or infer consent. It mints the spec stub, links both sides, stamps an
   identity onto every unmarked scope condition, and moves the intent
   `drafts/ → planned/`. Every relative markdown link that named the draft's
   path, from any file in the tree, is repointed at `planned/` in the same
   operation; the JSON lists each rewrite under `relinked` (`file`, `line`,
   `from`, `to`) — report them.

   **`--impact` is the judgement the interview settled**, stamped here because
   this is the moment it is made: a draft filed without one gets it now, in the
   same shape the create path writes (`impact: <value>`), validated at the same
   bar — one of `additive`, `breaking`, `fix`, never `internal`. Ask the human
   for the class if the draft does not carry it, and pass their answer; never
   type it into the frontmatter. The rules are the close's: a value that
   disagrees with one the record already carries is refused before anything
   moves (a plan does not revise a recorded judgement — the human edits the
   record they meant to change), the same value is accepted as a no-op, and
   without the flag the verb leaves the field as it found it, so the judgement
   stays owed to the close that ships. On an intent already in `planned/` the
   flag works the same way alongside the identity stamp.

   **"Plan" means this act and nothing else here.** The build plan the phase
   docs hold, a dated design plan, and a session's planning brief are three
   other senses — see the glossary entry
   [`plan`](../.abcd/development/brief/glossary/core/plan.md).
11. **Spec build:** replace the minted spec body's `_Draft:` placeholder with
    the real design record — scope, approach, and how it satisfies each
    acceptance criterion. Where the work is larger than one implementer can
    hold and land, list its steps under the minted `## Steps` section (see
    [Steps](#steps-the-unit-below-a-spec)); a spec that lists none is one step.
12. Re-run `abcd intent ready <itd-N>` and report READY to the user.

## Steps: the unit below a spec

A spec may split its work into **steps**: ordered, independently landable
pieces, each one lane and one pull request (adr-2609212115255771, decision 4).
The author writes them; nothing proposes a split. `abcd intent plan` mints every
spec with an empty `## Steps` section, and a spec that lists no step is built as
one step. The section is a numbered list, each step's title on its numbered
line and its footprint indented beneath it:

```markdown
## Steps

1. The parser
   - packages: internal/core/spec
   - tests: the parser over a stepped and an unstepped spec
   - landed: #123
2. The remainder copy
   - packages: internal/core/intent
   - tests: the remainder carries the unlanded steps
```

`- landed:` names what landed the step, a pull request or a commit; a step
without it is not landed. Any other line indented under a step is the author's
and travels with it. `abcd intent ready` reports the section's shape on its
advisory `steps` row: the steps listed and how many have landed, or none and so
one step. A section that is not a numbered list is named there with the shape
above, and never withholds readiness. The close below reads it too: a
`--remainder` close carries the steps not marked landed into the spec it mints,
and refuses, writing nothing, when the section cannot be read as steps.

## Ship: close the spec in the change that lands the work

The loop has a last step, and nothing runs it for you. `abcd intent plan` moves
a draft to `planned/`; the only verb that moves a planned intent to `shipped/`
is the spec store's close, which ships the linked intent as its close-hook —
but only on the close after which no open spec names it:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" spec close <spc-N> --json    # open/ -> closed/, and planned/ -> shipped/ when it was the last open spec
"${CLAUDE_PLUGIN_ROOT}/abcd" spec close <spc-N> --impact fix --json   # …stamping the judgement the record lacks
"${CLAUDE_PLUGIN_ROOT}/abcd" spec close <spc-N> --remainder <slug> --json   # partial delivery: close this spec, mint the rest, leave the intent planned
```

**An intent owns one or more specs.** Where the work did not fit one piece of
scheduled work, the spec that delivered part of it is closed on its own terms
and a new spec is minted for the remainder and attached to the same intent —
`--remainder <slug>` does both in one command, and the visible state afterwards
is exactly what happened: spec closed X, spec open Y, intent still `planned/`.
The remainder carries the closing spec's steps not marked landed, renumbered
from one, and the close names each step it carried; a landed step stays with the
spec that landed it.
The intent is never narrowed to match what was built; it stands as written, and
it ships on the close after which no open spec names it. A close that ships
nothing refuses `--impact`, because that judgement is written only at the close
that ships (adr-2609151513118583, invariant 17). Report the specs the close
names as still open — they are the reason the intent did not move.

**The close repoints every link that named a record it moved.** A record's
folder is its status, so the close renames two files — the spec out of
`open/`, and on the close that ships, the intent out of `planned/` — and in the
same operation it rewrites every relative markdown link in the tree that named
either old path: a spec already closed that pointed at `../open/<the spec>`, an
ADR or a plan naming the intent's `planned/` path, a draft naming both, and the
closed spec's own links, which were written from `open/`. A link that never
resolved is left as written. The JSON lists each rewrite under `relinked`
(`file`, `line`, `from`, `to`), and the text render prints them; report them,
because they are files the close changed beyond the two records. The tree the
close leaves passes record-lint's `links_resolve` with no hand repair. If the
repoint fails part-way, the close still stands and a warning on stderr says so.
After an attempt that failed before or during the repoint, re-running the same
`spec close` repoints every link other files still hold to either old path. The re-run reads the moved records' own links from the folders
they are in, because they may have been edited there since the move, so it
never rewrites them; any of those the failed attempt left unrewritten is one
`links_resolve` names, to repair by hand.

Run it in the **same change** that lands the intent's work — the commit or
pull request that makes the acceptance criteria true — the way a captured
issue is resolved in the change that fixes it. The reason is the release cut:
`abcd launch ship` composes the changelog from the records in terminal
folders, and a planned intent is not a refusal, it is simply not seen. An
intent whose code is on `main` but whose spec is still open ships with no
changelog line and exits 0 doing so; two intents delivering a breaking CLI
change were caught that way only by a reviewer. So the change declares the
delivery in a commit trailer, the way it declares an issue it resolves:

```text
Delivers: itd-N
```

The merge gate (RS005, beside RS001 in `scripts/check-issue-resolution.sh`)
refuses a change whose `Delivers:` intent does not enter `shipped/` in that
same change, and names every spec still open that names it, each with its
`abcd spec close`. The trailer takes the intent id, never the spec id, and it
means the change FINISHES the intent: a partial delivery closed with
`--remainder` leaves the intent planned and carries no trailer. A change that
declares no delivery is refused nothing, so a planned intent nobody claims to
have built stays the ordinary state of the backlog. The close that ships needs the
intent's `impact` — `shipped/` is the bucket `intent_impact_valid` requires one in, and
there is no default, because the judgement decides the derived version. A
record that already declares it — at create time, or where the interview
settled it, at `abcd intent plan --impact` — needs nothing; a record that does
not takes `--impact additive|breaking|fix` on the close, which stamps it before the move
(`internal` is a category error on a press-release-first intent, and is
refused). The close refuses rather than shipping a record with neither, and it
refuses a `--impact` that disagrees with one already written down: a close does
not revise a recorded judgement. It also refuses when the linked planned
intent is held (see Hold below), naming the reason and `intent unhold`, with
nothing closed and nothing moved. `spec close` is CLI-only — there is no
`/abcd:spec` page.
Both spec verbs — the close and the bare `abcd spec` status render — resolve the
repository root first, so they address the checkout's spec store from anywhere in
the tree and refuse with exit **2** outside a repository, where there is no spec
store to address; a spec store found below the repository root is named on
stderr and left alone.
Report the returned pair (the spec's new path, the intent's new path), then
queue the audit below.

## Autonomous runs

In an unattended run, exit 1 from `ready` is a SKIP: journal the rendered
findings and move to the next item. The planning interview, acceptance-criteria
authoring, and `abcd intent plan` are human-session-only acts.

An unattended run MAY prepare an interview without performing it: for a
plannable draft, write a planning brief to the local work tier
(`.abcd/.work.local/scratch/planning-briefs/`) — the summary-back with per-AC
provenance (seeded vs human-confirmed), the itd-84 hand-run as an ungraded
proposal, proposed acceptance criteria and open-question resolutions, and any
blocks-planning flags. The SOTA fit-challenge runs as a separate, independent
pass (evaluator outside the loop), filed alongside the brief. The pre-pass
reads the records the draft touches — a contradiction with a recorded
invariant is exactly what it exists to catch. It never edits the draft, never
files the routing, and never runs `plan`; the interview then starts from the
brief instead of a cold read, and grading into the calibration note still
happens only when the human confirms the routing.

## Hold

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent hold <itd-N> --reason "<one line: why>" --json
"${CLAUDE_PLUGIN_ROOT}/abcd" intent unhold <itd-N> --json
```

A hold is a frontmatter **state**, not prose: `hold` writes `held: "<reason>"`
onto a record in `drafts/` or `planned/`, and `unhold` removes the line. While
it stands, every lifecycle move refuses before anything moves, naming the
reason and `intent unhold` as the remedy: `abcd intent plan <itd-N>` — the
draft's plan and the planned record's identity-only re-run alike — and
`abcd spec close <spc-N>` on a spec realising the held record (no spec is
closed, no intent moves, the key is never stripped). `abcd <itd-N>` reports
the hold as the first next move. The hold is the mechanism under the "never
run `plan` unattended" convention: a lane that follows its own brief rather
than this page meets it.

The reason is required, one line, and redacted through the store's scanner
before it is written; report `redacted` from the JSON when it is non-zero, the
way the other write verbs do. A record already held is refused naming the
standing reason — an updated reason is `unhold` then `hold`, so the lift is a
visible act. Both verbs refuse a shipped, superseded or discipline record: a
hold on a record nothing will plan means nothing. `ready` is unchanged by a
hold — readiness is about the spec and the criteria.

The value is written by the verb, and a hand-typed `held: "<reason>"` is
byte-identical to that write, so nothing can tell the two apart and both stop
`plan` and `spec close`. What record-lint's `record_provenance` rule reports
is a `held` value in a shape the verb never writes — blank, null, a list, a
map, a block scalar, or a legal value on a record in a bucket the verbs refuse
— and `plan` refuses those too (fail closed) while `hold` and `unhold` send
you to the line to repair it by hand. A key spelled by hand in a way the reader
accepts but the verb never writes (`held : "…"`, a space before the colon) is
honoured as a hold and refused by `unhold` as a hand repair, never reported as
a lift that did not happen.

## Link

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent link <itd-N> <spc-N> --json
```

Retroactively writes a planned intent's `spec_id` when a spec already claims
it (the one-sided-link remedy `ready` reports). Report the linked pair.

## Review / ingest

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent audit --json                               # list the owed fidelity reviews (read-only)
"${CLAUDE_PLUGIN_ROOT}/abcd" intent audit --owed [--max <n>] --json            # drain them oldest first (see Drain below)
"${CLAUDE_PLUGIN_ROOT}/abcd" intent audit <itd-N> --json                       # re-emit a shipped intent's review request
"${CLAUDE_PLUGIN_ROOT}/abcd" intent audit ingest --verdict-json <file> --json  # apply a host-produced verdict
```

**Bare `intent audit` lists the debt and writes nothing.** Every `spec close`
that ships an intent parks an OWED review marker, so a fidelity review is owed
by construction. The listing reads the first marker of every intent in
`shipped/` and returns one entry per shipped intent (`intent_id`, `state`,
`receipt_id`, and `re_emit` where the review is owed), with the totals `owed`,
`dead_lettered` and `ingested`. The owed set is `OWED` plus `none`: a shipped
intent with no marker at all owes the review too, and its re-emit mints the
receipt. A `DEAD_LETTER` review is listed under its own heading, unreviewed,
with the reason the quarantine recorded, and is not counted as owed; an
`INGESTED` one is not listed in the text form. Report the owed total and, for
each owed intent, its receipt and its re-emit command. The listing names the
re-emit, never the request file: the request lives in the gitignored local tier
and may have been swept. It exits 0 whatever it finds; no gate reads it.

An intent this checkout does not hold is refused; when a peer holds it (a
sibling worktree or a local branch, see `/abcd:peers`) the refusal names the
peer's branch, path and bucket instead of answering not found.

Ingest is fail-closed: report the returned status (`ingested`, `dead_letter`,
or `noop`) and, for `dead_letter`, the reason. A second ingest for a receipt
already ingested is a `noop` when its payload renders to the block on the record,
replaces that block in place when it renders differently (`ingested`, reported
as `replaced`), and is refused with nothing written when it does not validate:
a bad re-ingest never dead-letters a verdict already ingested.

**Model-tier routing.** Both `intent audit <itd-N>` and `intent audit ingest`
dispatch the `intent-auditor` agent, and each resolves that agent's model tier
before anything else runs: an invocation override, over the repository's
`.abcd/config/oracle-routing.json`, over the machine's
`~/.abcd/oracle-routing.json`, over abcd's bundled proposal (which applies only
once a table is accepted). The override is `--route
<agent>=<tier>[@<connection>][?k=v,...]`, naming the one agent this invocation
dispatches (a second `--route` is refused, not merged), with the tier one of
`local`, `economy`, `frontier` or `host-decides`; it governs this run alone. The
emit's `--json` result carries the request block as a `routing` member (`agent`,
`tier`, `fan_out`, `source`, `origin`, `override`, `connection`, `fallback`),
its text a `routing:` line, and the request file a `## Routing` section after
the provenance block: run the auditor at that tier where the harness lets you
choose one. The section sits outside the hashed prompt, so it never moves
`prompt_hash`. The request `spec close` emits when it ships an intent carries
the same section; a routing table that cannot be read leaves that request
without one, one stderr warning names `intent audit <itd-N>` as the re-emit that
adds it, and the close stands. The bare listing and `--issue-drift` dispatch no agent and refuse `--route`. The
ingest's `--json` result carries a `route` receipt (`tier_asked`,
`connection_tried`, `connection_used`, `fallback_reason`, `override`,
`settings_sent`, `model_reported`) and its text a `route:` line; relay it with
the result. When no configured provider can serve the tier, one stderr line says
the step goes through the harness instead. A `--route` naming an agent this
invocation does not dispatch, a tier outside the set, a connection this machine
has not configured, or a routing table that cannot be read exits 2 before
anything is written. With no table accepted and no `--route`, the step asks for
`host-decides` and nothing is printed.

**Hand the auditor the whole request file.** `intent audit` writes it to the
reported `request_path`, and its `## Provenance` block states the
`rubric_hash` and `prompt_hash` the host computed. The auditor echoes both
verbatim into `policy`; it never computes either itself. The ingest recomputes
them and refuses a verdict carrying any other value, leaving the receipt parked
so the request can be re-emitted and the audit re-run — so a made-up hash costs
a whole review rather than quietly writing provenance nobody issued.

The verdict also disposes the intent's scope conditions, keyed to the `cond-…`
identity each one carries: every condition receives exactly one of `survived`,
`narrowed`, `falsified` or `untested`, and a `narrowed` condition states what it
now holds under. Coverage is exact in both directions — a conditionless intent
takes an empty block, a conditioned one a full one — so a partial or invented
disposition quarantines the whole payload rather than applying half of it.
Report the returned split alongside the acceptance rollup.

## Drain: pay the owed reviews, oldest first

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent audit --owed --max <n> --json
```

`--owed` is the bounded command that pays the review debt. It returns `queue`
— the owed reviews, oldest shipped first (`shipped` is the day the intent
entered `shipped/`, and `shipped_state` says which fact holds: `dated`;
`uncommitted`, shipped in the working tree and not yet committed, with no day
and last; or `unknown`, when the history could not be read, which leaves every
day unknown and the queue in mint order), at most `max` of them — with `owed`, the whole total, and
`remaining`, how many the cap left out. `next` names the oldest one's
`request_path` and its `routing`: the command has just emitted that request,
exactly as `intent audit <itd-N>` does — its routing section included, and a
`--route intent-auditor=<tier>` override applied the same way — minting the
receipt if the intent had none. It runs no reviewer. Nothing owed is `owed: 0` and no `next`; report it and stop.
`--max` without `--owed` is refused, as are `--owed` with an intent id or with
`--issue-drift`.

Run the loop one audit at a time, never in parallel — the cap and the one
auditor at a time are what bound the cost:

1. **Check for an auditor first.** The `intent-auditor` agent must be in the
   host's agent listing. When it is not, or its launch is refused, run no
   audit: every entry stays owed, nothing is marked failed or dead-lettered for
   want of a reviewer, and the summary says why nothing ran ("0 audited; 12
   owed left owed: no intent-auditor available — <what the host said>"). A
   refused launch part-way through stops the loop the same way, and the
   summary names the entries it did not reach.
2. **For each entry in `queue`, in order:** run `intent audit <itd-N> --json`,
   with the same `--route` when the drain was given one (for the first entry
   the request is already written, and the re-emit is idempotent), hand the whole request file to the `intent-auditor` agent,
   write the verdict it returns to `.abcd/.work.local/scratch/`, and run
   `intent audit ingest --verdict-json <file> --json`. The verdict lands exactly
   as a single audit's does — the Audit Notes block, the receipt, the scope-
   condition dispositions. Report the ingest's status, then take the next
   entry; start the next audit only after this ingest has returned.
3. **A NOT_MET verdict is captured, never fixed.** Every intent the drain
   reaches has already shipped, so a criterion it did not meet is a finding
   against delivered work: file it with
   `abcd capture "<itd-N> fidelity audit NOT_MET: <criterion> (receipt <rcp-…>)" --category drift --severity <minor|major> --source review-followup`,
   naming the receipt, and continue the loop. The drain changes no code and
   re-opens nothing; the fix round belongs to the build that owns the work. A
   `dead_letter` ingest is reported with its reason and is listed apart by
   bare `intent audit` from then on.
4. **Summarise:** how many were audited, the ingest outcome of each, the
   captures filed for NOT_MET (their ids), how many stay owed — the command's
   `remaining` plus any entry the loop did not reach — and why the loop
   stopped: the queue ran out, the cap was reached, or no auditor was
   available.

A host without this page drives the same pair by hand: the text form prints
the ordered list and the `next:` request path, and after the verdict is
ingested the next `--owed` run finds the queue one shorter. Nothing starts the
drain on its own: the spec close still only parks the OWED marker, and no hook,
gate or schedule runs a reviewer.

## Condition: disposition one scope condition from a reading or a delivery

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent condition <itd-N> --json          # read-only: each condition's standing
"${CLAUDE_PLUGIN_ROOT}/abcd" intent condition <itd-N> <cond-id> \
    --disposition <survived|narrowed|falsified|untested> \
    --occasioned-by <rdi-N|itd-N> --grounds "<why>" [--narrowing "<what now holds>"] --json
```

The verdict ingest above is one writer into a shipped intent's scope-condition
dispositions; this is the second. It records that a reading item, or a later
delivery, changed an assumption's standing, keyed to the `cond-…` identity the
condition carries rather than to its wording, and joined to what occasioned it.

- **With the intent alone** it writes nothing. Report every condition's standing
  disposition and the block it came from (`from verdict rcp-…` or
  `from condition <occasion>, <date>`), or `untested (no block)`. `--json`
  carries the whole history under `dispositions` and the fold under `standing`.
- **With a condition id** it appends one dated block to `## Audit Notes`:
  the identity, the value, the occasion and the grounds, with the narrowing
  under a `narrowed` value. The occasion is a reading item at any position, or
  an intent in `shipped/` whose delivery changed the condition's standing. Ask
  the researcher for the value and the grounds; the reading names the tension
  and never marks the condition itself.

A condition's standing is its latest reading-occasioned block where it has one,
and otherwise its latest verdict: a fidelity verdict leaves a reading-occasioned
block standing unless the verdict's rationale names that block's occasion,
wherever the two sit in `## Audit Notes`. The verdict ingest reports each block
it leaves standing (`still standing: …`); an auditor who meant to override one
names its occasion in the rationale and ingests again for the same receipt,
which replaces the ingested verdict (reported as `replaced`).

Every refusal exits 2 with nothing written: an intent not in `shipped/` (the
refusal names its bucket), a condition id the intent does not carry or carries
twice, a value outside the four, grounds below the substance floor, `narrowed`
without `--narrowing` or `--narrowing` with any other value, an occasion that
does not resolve (an absent reading item, or an intent that is absent or not in
`shipped/`), and the intent named as its own occasion. The grounds and the
narrowing are redacted before the write; report a `redacted` count when there
is one. When the reading item's `constraint_in_play` cites a different
condition's identity, the result carries `occasion_citation` and the render a
`note:` line: report it as a question for the researcher, never as a refusal.

## Issue drift: does every promote join read from both ends?

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" intent audit --issue-drift            # warnings on stderr, exit 0
"${CLAUDE_PLUGIN_ROOT}/abcd" intent audit --issue-drift --strict   # exit 1 on any finding (CI)
```

An intent promoted from a ledger record names it in `related_issues`, and the
record names the intent back in `related_intents` (`/abcd:capture promote`
writes both). The drift check walks the intent store and the ledger, readings
included, and reports each join that does not hold: `one_sided` (an intent names
a record that does not name it back; from a reading item's end, the reverse —
an issue's one-way `related_intents` is a loose relation and is not reported),
`dangling` (either end names a record the tree does not hold),
`shipped_unresolved` (a shipped intent names an issue that is not in
`resolved/`), and `retired_field` (a record still carrying a retired back-link
key — `/abcd:capture migrate --apply` rewrites it). Report the finding count,
each finding's kind and records, and the receipt path the run left under
`.abcd/.work.local/logs/audit/`. It writes to neither store. `--strict` without
`--issue-drift` is refused.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
