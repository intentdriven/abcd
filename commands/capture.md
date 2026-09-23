---
name: capture
description: Capture issues to the structured per-repo ledger and query them, by invoking the abcd binary. Bare invocation is a read-only status render; disposition/link/list/promote/resolve/wontfix act on the ledger.
argument-hint: "[text] | list --open|--resolved|--wontfix|--all | link <iss-N> [--blocked-by <iss-M,...>] [--unblock <iss-M,...>] | promote <iss-N> --grounds \"<token>: <text>\" [--intent <itd-N>] | promote <rdi-N> [--intent <itd-N>] | resolve <iss-N> <note> --impact <additive|breaking|fix|internal> --grounds \"<token>: <text>\" [--intent <itd-N>] [--spec <spc-N>] [--commit <sha>] | wontfix <iss-N> <reason> | disposition <rdi-N> --state <accepted|rejected|declined|held>"
---

# `/abcd:capture` — issue ledger

The lightweight write side of the structured issue ledger under
`.abcd/work/issues/`. Every issue gets a stable `iss-N` id, schema-checked
frontmatter, and folder-as-status (`open/`, `resolved/`, `wontfix/`). Bare
invocation **performs zero writes**.

Every verb here addresses the CHECKOUT's ledger, whichever directory of the
working tree it runs in: the repository root is resolved from the working
directory, never taken to be it. Outside a git checkout there is no ledger to
address, so every verb exits 2 and writes nothing — the ledger is per-repository,
and a record filed outside one is committed by nothing and read by nothing. When
a ledger also sits between the working directory and the checkout root, the verb
names it on stderr and leaves it exactly where it is; report that line to the
user, because the records under it reach no gate and no release cut, and only
they can tell a deliberate fixture store from one a stray capture left behind.

## Status (bare)

To render recent captures and counts:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture --json
```

Summarise the JSON for the user: `open_count` / `resolved_count` /
`wontfix_count`, and for each entry in `recent_open` its `id`, `severity`, and
`slug`. No `iss-*.md` file is created, moved, or mutated by this invocation.

**Which ledger?** A half-formed observation, question, or nitpick goes to
`/abcd:capture "…"`; a user-facing change you want to ship goes to
`/abcd:intent "…"`. For a big, unproven idea there is an optional third route:
`/abcd:ideate` runs the admission gauntlet and records the verdict either way.
It is a pointer, never a precondition — capture friction stays at one line.

## Capture an issue

Append a structured issue from free-form text:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture "<text>" --json
```

Provide provenance and taxonomy through flags when known (each falls back to a
default): `--severity` (`nitpick|minor|major|critical`, default `minor`),
`--category` (default `observation`), `--source` (default `user-observation`),
`--found-during` (session/command context, default `manual-capture`),
`--found-at` (optional repo-relative path), `--lapsed-at` (RFC 3339 instant in
UTC at which a recorded discipline gave way — the lapse itself, never the
write-up), `--slug` (overrides the slug derived from the text), `--blocked-by`
(comma-separated `iss-N` ids this issue depends on; each must already exist in
the ledger, and an edge to a record captured later is written afterwards with
`link`, below), `--production-mode`
(`hand-written|dictated-and-formatted|scribe-transcribed`, default: the repo's
declared mode, else `hand-written`). Report the new `id`, `status`, and `path` from the JSON. Report `redacted`
too whenever it is non-zero: it counts the spans rewritten before the text was
written, and the user needs to know their wording was changed.

`--category lapse` takes `--lapsed-at`, which has no default: a lapse capture
that omits it records no instant, never the write-up time. The refusal on an
omitted instant is parked (iss-2609091009111294) until the rethink of the reading
work settles what a lapse record must carry; the instant the discipline gave way
is still what the flag exists to record, and a value that is given must be an
RFC 3339 instant.

## Disclosure: where a record came from and how its text was produced

Intent, spec and issue records carry two frontmatter keys, written by the
commands that mint them, and no flag carries either as free text. Records of
other families — a disposition, for one — carry neither.

`origin` is **derived from which command ran** and has no flag at all:
`researcher-authored` for text written directly rather than derived (it names
the route, not whether a person or an agent ran the command),
`extracted-from-record` for `capture promote <iss-N>` — an issue is something a
person noticed — and
`contributed-by-reading <rdg-N>/<rdi-N>`, which `capture promote <rdi-N>` mints
when it derives a draft from an accepted reading item, naming the item's run and
id. It is stamped when the record is minted and never rewritten: where a record
came from does not change when it is resolved, and linking an existing draft
with `--intent` leaves its `origin` alone.

`production_mode` is the closed choice `--production-mode` carries:
`hand-written`, `dictated-and-formatted`, or `scribe-transcribed`. Any other
value is refused and nothing is written. On `capture` and `capture promote` an
absent flag takes the repo's declared default from `.abcd/config/identity.json`,
falling back to `hand-written`. On `capture resolve` and `capture wontfix` the
flag **restamps** the record — a resolution note is new text with its own mode —
and an absent flag leaves the record's existing stamp alone. A restamp of a
record that predates disclosure (one carrying no `origin`) is refused before
anything is written, because the pair is written together or not at all; re-run
without the flag. Such a record still resolves normally.

Neither key touches authorship: they are disclosure at field granularity, on the
same footing as the `Assisted-by:` trailer at commit granularity. Population is
forward-only, so a record written before the keys existed carries neither, and
nothing backfills it.

The `record_provenance` record-lint rule reports a record carrying the pair in a
shape no write path produces: a value outside its set, one key without the
other, `extracted-from-record` with no `promoted_from` back-edge, a reading
pointer that resolves to no reading record, or a reading pointer whose item and
`promoted_from` back-edge name different records — the two are one join written
twice, and the rule reads it from both ends. A hand edit that types a legal value
in a legal combination is byte-identical to a command's write, so the rule
catches implausible hand edits, not all of them.

A single whitespace-free word is refused (exit 2, nothing written): a lone
token reads as a mistyped sub-verb, never as issue text. A near-miss of a real
sub-verb is refused the same way, with the correction named, so a two-word input
containing a space is not automatically safe. Neither is a word followed by an
issue id — `abcd capture closeit iss-1 "…"` is a sub-verb call by shape whatever
the word is, so it is refused whether or not any sub-verb is close enough to
suggest, and a refusal with nothing to suggest lists the sub-verbs the verb has.

Priority is **derived, never stored**: an issue is ranked lower while any of its
`--blocked-by` targets is still open, and `blocked_by` records the dependency in
one direction only (the inverse is computed). A target the reader had to skip
counts as open — it is still in `open/`, and being unreadable says nothing about
whether it was resolved — so it goes on blocking, with the skip reported in the
same result.

## Link: add or remove a dependency edge after capture

`--blocked-by` at capture time serves only the case where the blocker already
exists. The ordinary case is the other one — the blocker is captured after the
blocked record, or in another lane — and `link` is the verb that writes the edge
whichever record came first:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture link <iss-N> --blocked-by <iss-M,...> --json
"${CLAUDE_PLUGIN_ROOT}/abcd" capture link <iss-N> --unblock <iss-M,...> --json
```

`--blocked-by` appends the given ids to the record's existing `blocked_by` list;
`--unblock` removes them. At least one of the two is required, both in one call
are allowed, and the removals are applied **before** the additions — so the same
id on both sides is removed and re-added, a net no-op. The subject may sit in
any status folder and never moves: a resolved record's edges are history, still
editable. Report the `id`, `path` and `blocked_by` (the list **after** the
write) from the JSON; the plain render is one line carrying the same three.

The targets are validated exactly as the capture-time flag validates them — one
validator, two callers — and every refusal writes nothing: an id that is not
`iss-N`, a record naming itself (a record cannot block itself), and a target
absent from the ledger in every status folder, which is refused naming the two
places the field is documented (`.abcd/work/issues/README.md`, its "Derived
priority" section, and this page). Blocking on a resolved or wontfix target is
legal — existence is what the edge claims, and whether it still holds anything
up is the derived view's question. Duplicates collapse, so linking an edge the
record already carries succeeds unchanged. An `--unblock` of an id the list does
not currently hold is refused naming the current list: removing an edge that is
not there is a wrong belief about the record, not a no-op. When the last edge is
removed the key is dropped, and the record reads as one captured without the
flag.

The write goes through the ledger's in-place frontmatter rewrite, so the list
is spelled exactly as capture spells it and the derived-priority view picks the
change up on the next `list` or status render with nothing else to run.

## Query the ledger

`list` is the one earned filter-flag exception — a filter is **required**:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture list --open --json      # or --resolved / --wontfix / --all
```

The unfiltered form `abcd capture list` exits 2 with a "choose a filter"
message; there is no implicit default.

**One id, one status folder.** Every read — `list`, the bare `abcd capture`
board, and `abcd <iss-N>` — refuses when one id is claimed by two record files,
naming both. The status folder *is* the record's status, so an id sitting in
`open/` and `resolved/` at once has no defined status to report, and rendering it
would mean printing two contradictory rows or picking one arbitrarily. The state
is a merge artefact rather than a hand edit: a record committed to the default
branch after a branch was cut from it, and then resolved on that branch, arrives
as an add on one side and a delete-plus-add on the other, which rename detection
does not pair. Relay the refusal; the fix is to move or remove one of the two
files so the ledger says which status the record is in. Summarise each issue's `id`, `status`,
`severity`, and `slug`. The list is returned in **derived-priority order**:
unblocked issues first, then by severity (`critical` → `nitpick`); rows still
blocked by an open dependency are demoted and annotated `[blocked-by iss-N,…]`.

## Which open issues may already be fixed

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture mentions --json          # or --ref <branch>
```

`mentions` reads the default branch's history and lists the open records its
commit messages name. It is **advisory and strictly read-only**: it resolves
nothing, moves nothing, and writes nothing. A mention is not a fix, and the row
exists so a human reads the commit and decides.

The rule it complements is the resolution gate's: a commit message, or a
pull-request title **or body**, that names an `iss-N` must declare its relation
to it — `Resolves: iss-N` for a change that fixes it, `Refs: iss-N` for one that
touched it without fixing it. One line may name several records
(`Refs: iss-1, iss-2`); the two spellings are the whole vocabulary. That gate
runs before a merge and cannot reach backwards, so this listing is what reads
the history a repository already has.

Each row carries the strongest evidence found for the record, rows are ordered
strongest first, and a row's own evidence is ranked the same way — so the commit
shown is the one to read, not merely the latest one that named the record:

| Strength | What it means |
|---|---|
| `resolves` | a commit declared `Resolves: iss-N` and the record is still in `open/` — somebody said it was fixed and the ledger never moved |
| `tree` | a commit that changed something outside `.abcd/` named the record |
| `record` | only the record tiers changed — somebody wrote *about* the record |

Two mentions are deliberately silent. A commit that **filed** the record names
the id it is filing: that is provenance, not evidence, and it is the commonest
mention in any ledger's history. A commit that declared `Refs: iss-N` said in
so many words that it did not fix it, and the listing takes the author at their
word — reporting it anyway would teach people to stop declaring.

Relay the `id`, `strength` and the naming commit; the JSON carries every mention
under `evidence`. The next move is a human's: read the commit, then
`capture resolve <iss-N> "<what fixed it>" --commit <sha>` with its impact and
grounds, or leave the record open.

## Grounds: why this triage, not just which one

Every triage route records the CONJECTURE being acted on. The vocabulary is
closed — `pursued`, `deferred`, `declined` — and the text is free prose:

```bash
--grounds "pursued: <what is expected, and what would show it wrong>"
```

`promote <iss-N>` and `resolve` record it when it is given and write no entry
when it is not: the refusal on an absent value is parked (iss-2609091009111294)
until the rethink of the reading work settles what a human is asked for at a
triage. A malformed value is still refused — an unknown token, a missing colon,
or a text below the substance floor. Every grounds refusal is a usage error at
exit 2, on all three routes.
`promote <rdi-N>` is the one route that takes no grounds and refuses one handed
to it: a reading item states its conjecture in its disposition, which promote
already refuses to act without, so a second one here would reach no record.
`wontfix` does not, because its reason is already mandatory — it stamps
`declined: <reason>` from the reason it already takes, and
`--grounds "declined: <text>"` overrides that text for the case where the
conjecture and the user-facing reason are not the same sentence.
The token there stays `declined`: a wontfix IS the non-action that value names.

**Ask for the expectation and its falsifier.** "Promoted it because it is next"
restates the decision and records nothing; "promoted it because we expect a
stamped identity to survive rewording, which nothing else does" is a conjecture
somebody can later find wrong. abcd refuses only the degenerate texts — empty,
too short, or the vocabulary word repeated back — and cannot tell a conjecture
from a restatement. That part is yours: put the question to the user and write
down their answer.

The value is APPENDED as a `- <token>: <text>` bullet under the record's
`## Grounds` heading, never set in frontmatter, so a record promoted and then
resolved carries both conjectures: the earlier one is what a later reader checks
the outcome against. It is scanned before it is committed, so report `redacted`
whenever it is non-zero.

## Resolve / wontfix

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture resolve <iss-N> "<resolution-note>" --impact <additive|breaking|fix|internal> --grounds "<token>: <text>" --json
"${CLAUDE_PLUGIN_ROOT}/abcd" capture wontfix <iss-N> "<reason>" --json
```

Each moves the issue out of `open/` and records the note; report the `id` and
the `from_status -> to_status` transition from the JSON. Report `redacted` too
whenever it is non-zero: these paths redact the note exactly as `capture` does,
but their human render stays silent, so the caller learns their wording was
rewritten only if you relay it.

An id this checkout's ledger does not hold is refused. When a peer holds it —
a sibling worktree or a local branch (see `/abcd:peers`) — the refusal names
the peer's branch, path and folder instead of answering not found: the record
lives there, so relay that rather than capturing it again here.

`resolve` requires `--impact`: a resolved issue is in the release set, so it
carries the product judgement the version derivation reads (`additive`,
`breaking`, `fix`, or `internal` — plumbing invisible to users). There is no
default; an absent or misspelled impact is refused rather than guessed, so the
record always satisfies the `issue_impact_valid` gate. `wontfix` takes no impact
(a non-action ships nothing). Both take `--production-mode`, which restamps the
record rather than defaulting it (see the disclosure section above).

`resolve` also takes optional provenance — the structured `resolved_by` pointer
to what fixed the issue: `--intent <itd-N>`, `--spec <spc-N>`, `--commit <sha>`,
in any combination. Supply them whenever the fixing record is known — that is
what keeps the trail six months later. Ids must exist in their record store
(any bucket); the sha is shape-checked only (7–64 hex chars — its home may be
the remote). An unknown id or malformed value refuses the whole resolve and
writes nothing.

`resolve` also takes `--shipped-in <vX.Y.Z>`, a MIGRATION flag for the
ledger-hygiene case: closing a record whose fix was RELEASED LONG AGO. A
repository abcd manages from its first commit should never need it — resolution
rides the fixing commit there, so the release cut is right without it. The release derivation leaves such a
record out of the current cut, so a sweep that closes old records cannot make the
next release announce their fixes as new. Use it only when the work genuinely
shipped in a named earlier release; absent means "this cut", and abcd never
infers it. `resolve` shape-checks the value only (`vMAJOR.MINOR.PATCH`); that the
tag exists and the release being measured from can reach it is enforced later, at
release derivation, which keeps a wrong-version record IN the cut with a stated
reason rather than dropping it silently.

With no provenance flags the record is byte-identical to a
plain resolve: provenance is optional, never guessed. The written members come
back in the JSON as `resolved_by`. `wontfix` takes no provenance — a non-action
points at nothing.

## Answer a reading item

A reading record is what an instrument returned; the researcher's answer to one
is a **separate record**, keyed to the item:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture disposition <rdi-N> --state accepted --grounds "<why>" --json
```

The two are never one write, so the ledger can always show that a finding
existed before it was answered. Report the `id`, `item`, `state`, `position` and
`path` from the JSON.

**Where an `rdi-N` comes from.** This surface answers and promotes reading items;
it does not produce them. The one writer of the `rdi-N` family is the
cold-reading ingest verb, which owns the output contract a reading is validated
against — until that verb lands there is no reading item to answer, and these two
sub-verbs have nothing to act on.

Four states ship: `accepted` (at the widening position, acceptance IS
admission), `rejected` (asserts a purpose a later run tests), `declined` (the
widening position's own: the proposal was admissible and the researcher chose
otherwise), and `held` (directional, and requires `--exit-condition`). Which
states are available depends on the item's **position**, which the verb reads
off the keyed reading record — never from a flag, because a caller-supplied
position would let a disposition assert the rule it has to satisfy.

`--grounds` is required on every state except `held`. A second answer to one
item must cite the standing one with `--supersedes <dsp-N>`. Where **more than
one** answer already stands — two branches each answered the item and merged
cleanly — the verb refuses instead: a fresh answer supersedes at most one of them
and adds its own, so the contest would never shrink. Write
`supersedes_disposition` into the records that are no longer meant to stand, by
hand, until exactly one does.

The standing disposition of an item is the one no sibling supersedes, and the
superseded record stays in place, because a hold that vanished when it was
answered would take its own exit condition with it. `--recurs` cites prior item ids — the
recorded form of a warm recognition that something has come back, never a
mechanical join and never a state of its own.

`--hold-frame-location` and `--hold-moscow` are **reserved and dormant**: the
grammars are stated and a populated value is refused until activation is ruled.
Nothing means "already covered" — an item nobody has answered is reported as
outstanding by `abcd lint`, never named as a state.

**Admissions and surprises are written by hand.** A widening proposal admitted
into the candidate set carries an **admission record** (`adm-N`, under
`.abcd/work/issues/admissions/<run-id>/`) whose `grounds` say what it was
admitted on; a **surprise entry** (`srp-N`, under
`.abcd/work/issues/surprises/`) records what was unexpected, keyed by
`occasioned_by` to whatever occasioned it and never folded into a disposition. A
declined proposal is not a third record: it is the disposition above in its
`declined` state. Neither shape has a sub-verb — this surface writes no `adm-N`
and no `srp-N`, and the command-side refusal is the next iteration's. What holds
today is the committed-tree gate: `record_schema` refuses an admission whose
`grounds` carries no value on the key's own line, an admission with no
`proposal`, a surprise whose `occasioned_by` names a record the corpus does not
hold, and either record filed in the other's store.
Carrying no value is judged by the kind of YAML node the value is, not by the
literal it is spelled with, so there is no list to fall outside of: empty,
whitespace, quoted-empty, quoted-whitespace, an empty flow collection (`[]`,
`{}`), a YAML null however it is written (`~`, `null`, `!!null`, `!!null null`,
`!<tag:yaml.org,2002:null>`), a node that is nothing but a tag, an anchor or an
alias (`!!str ''`, `!!seq []`, `&anchor`, `*alias`), and a block scalar holding
nothing all carry nothing alike. A trailing comment is stripped before the value
is judged, so it hides none of them.
`abcd lint` reports a widening proposal carrying neither an admission nor a
decline, at `info`.

## Promote an issue into an intent

When a one-line issue turns out to be a capability, graduate it without
retyping:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" capture promote <iss-N> --grounds "pursued: <conjecture>" --json
"${CLAUDE_PLUGIN_ROOT}/abcd" capture promote <rdi-N> --json   # a dispositioned reading item
```

One invocation mints a new intent draft under
`.abcd/development/intents/drafts/` — slug reused from the issue, body carrying
a by-id pointer ("Graduated from `iss-N`"), never a copy of the issue body —
and stamps the issue's `promoted_to` with the minted `itd-N`. The draft's
frontmatter records `promoted_from: iss-N`, so the edge is two-sided, and its
`origin` reads `extracted-from-record` — the one arrival path a command derives
from what it did. The issue
keeps its status folder: promotion is orthogonal to fix-status and is not
resolution. An issue already carrying `promoted_to` is refused with the
existing `itd-N`.

A reading item (`rdi-N`) graduates the same way, with one refusal in front: only
an item whose **standing disposition is `accepted`** may be promoted. Acceptance
is one record and the action is a separate admission, so an item that carries no
disposition is refused (promoting it would collapse the two acts, and then
nothing could show the finding was weighed before it was acted on), and so is one
whose standing answer is `rejected`, `declined`, or `held` — the first two would
put a refusal and the admission it refused in the same ledger, and the third
would settle by action exactly what the hold left open. Where the answer needs to
change, supersede it: `capture disposition <rdi-N> --state accepted --grounds
"…" --supersedes <dsp-N>`. Nothing is minted when the promote is refused.

The draft a reading item mints carries `origin: contributed-by-reading
<rdg-N>/<rdi-N>`, naming the run the item sits in and the item itself, beside the
`promoted_from: rdi-N` back-edge — so the join reads from both ends, and the
record shows what a reading caused as well as whether a reading occasioned an
intent. Its Press Release seed names no item ("Seeded by promotion from a reading
item"): that section is projected to a later reading, and no reading sees
another's output. The item's own text stays in the reading record.

For a reading item the JSON's `issue_status` carries the **standing
disposition's state** (`accepted`), not a status folder: that family's status
signal is the keyed disposition, and it has no folder to name.

`--intent <itd-N>` is the stamp-only mode: it links an *existing* draft instead
of minting — the repair path when a stamp failed after the mint (the error
names the orphan draft and this exact remedy, the promotion's own `--grounds`
included, so it runs as printed), and the path for "I already
filed the intent by hand; link them". Report the `issue_id`, the minted (or
linked) `intent_id`, and both paths from the JSON.

On the reading route `--intent` writes both edges: `promoted_from` on the draft
and `promoted_to` on the item. It never touches the draft's `origin`, which was
stamped at mint — a draft filed from quoted text stays `researcher-authored` and
says so.
A draft whose `promoted_from` already names another record keeps it: an intent
occasioned by several items is promoted from one and joined to the rest by their
own `promoted_to`, so the item is still stamped forward and the result reports
`back_edge_kept` (`back_edge: kept <rdi-N>` in the plain rendering). Report that
line when it is present.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
