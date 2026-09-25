---
name: reading
description: "Render the cold-reading assembler's state: Writes nothing; refuses any argument."
argument-hint: "[] | assemble --position <widening|entailment|comparative|detection> --target <HEAD|sha> [--out <dir>] [--dry-run] | ingest --reading-json <path>"
block: agents
---

# `/abcd:reading` — cold-reading input assembler

Blindness is a property of the input, not a promise the reader makes. A
positive include table names what may travel; fields are projected out of
records rather than files copied whole; and every run emits a manifest naming
what was passed, by path and field, hashed, so a reader can judge contamination
rather than accept a disclosure on trust. Bare invocation **performs zero
writes**.

Two things this surface does not do. It never runs a reading: it produces the
input a reading would be given, and dispatching that input to a reader is host
work. And it carries no free text at any position — the operator supplies a
position and a target state, each in a closed grammar, and the reading's object
and question come from its definition, so there is no channel through which
ledger content can travel in the framing of a request.

## Status (bare)

To render the assembler's state:

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reading --json
```

Summarise the JSON for the user: `assembler_version`, `include_rows` and
`exclusion_rows` (what the table admits and what it refuses), `definitions`
(the reading definitions the locator RESOLVED), `staged_runs` (runs an
assembly has parked in the local tier), `orphaned_ingests`, and
`leftover_stages`. Zero writes.

**`definitions` is what resolved, not what is present.** A definition is
resolved by the position its filename holds, and it must state that same
position and the regime that position carries — the regime is the definition's
property, which is why nothing an operator types can set one. So a
`cold-reading-<name>.md` naming no closed position is not an instrument and is
invisible here. A definition that IS at a position and is silent about its
position or its regime, states a position its filename does not hold, or states
another position's regime, is worse than an absent one: it reports an instrument
that is not there. That refuses **the whole verb with exit 2**, naming the file
and what it got wrong, rather than being listed or quietly skipped — so a short
list is a repository with fewer definitions, and a malformed one renders no
status at all.

**`orphaned_ingests` is not routine.** Each name is a run whose ingest reached
the ledger and never reached its commit marker, so its reading records are
sitting in the committed ledger for a run that never happened. Report it
whenever it is non-empty, and say that the next ingest that validates rolls
those records back.

**`leftover_stages` is the other thing a stage can be, and it is not an
orphan.** Each name is a run that DID commit — its `run.json` is down — and
whose stage merely failed to clear afterwards. Its records stay. Report it,
and say that the next ingest that validates clears the stage alone; never say
its records will be rolled back, because they will not be.

## Assemble one reading's input

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reading assemble \
  --position widening --target HEAD --json
```

**The invocation is two operands and nothing else.** `--position` takes one of
four closed tokens — `widening`, `entailment`, `comparative`, `detection`. An
unknown token is refused by name. `--target`
takes `HEAD` or a hexadecimal commit sha of 7 to 40 digits; a branch name or a
tag is refused, because it moves and the manifest's re-runnability rests on a
reference that cannot. Both are required, and any other operand is refused by
name.

What the reading is **handed** comes from the committed preset entry for that
position, in `.abcd/config/reading-presets.json`, applied by the assembler with
no operand naming it. The entry intersects what the position already admits and
can only narrow it. **No repository path is accepted at the invocation** — a
path may be named only inside the committed preset file, where it is reviewed,
shape-validated and inside the dirty gate (adr-2609021016286571, which
supersedes adr-58).

Changing what a position reads is a commit to that file, reviewed like any
other change; there is no override at the invocation and nothing to stamp. One
file, one entry per position: a repository that wants a wider reading commits a
wider entry, and the manifest shows which entry a run applied.

**At entailment, a draft or planned intent travels only when the entry names
it.** The drafts and planned rows are admitted at that position alone, and they
narrow by the entry's object-set record list exactly as the shipped row does —
so the reading is handed the object set's drafts and planned intents and no
others. The readings companion's section 6.2 makes them *admissible* there, and
admissible is a permission rather than a scope: an entry that wants the
permission whole declares `"admit_drafts_and_planned": true`, which hands every
draft and planned intent in the repository. The key defaults to off, means
nothing at any other position, and is refused at load if an entry for one
declares it (ruled 2026-09-02).

**The comparative position derives its candidate set from the record.** Its
object is the widening reading's pre-admission output, and the run that supplied
it is not named by any operand: the assembler selects **the one committed
widening run at the target whose items carry no disposition and no admission**
(adr-2609021016272867). A run is **at the target** when the commit its own record
names *is* the target, or is an **ancestor** of it across which nothing changed
outside the readings store and the issue ledger's own record families. That is
what lets the loop run: `reading ingest` leaves a widening run's reading records
uncommitted, the candidate row below reaches them, so the next assembly refuses
until they are committed — and committing them moves HEAD off the commit the run
read. A commit that moved only the instrument's own record leaves the object set
where it was, which is what the run was about. A run whose target is not an
ancestor is not a run at this target and is not listed; a run across which
anything else changed is listed and refused, naming the first path that moved.
The manifest records both commits — `candidate_run_target` beside
`target_commit` — so a reader can diff them. *This reading of "at the target" is
an interpretation, and the maintainer's ruling is owed* (iss-2609021857343626).
That run's items travel projected to two body fields —
the configuration and what admits it — keyed by the item identifier the
comparative body cites, and nothing else from the readings store travels with
them: no disposition, no admission, no surprise, no other run, no manifest. The
committed entry for the position names the repository material passed beside the
candidates, which at this position is the criteria discipline and nothing else.

Two refusals, and both **list the widening runs at the target** with each run's
item count and the fate of its items, so the operator can see what to
disposition:

- **None qualifies.** No committed widening run at the target has every item
  free of a disposition and an admission — because there is none at all, because
  one never reached its commit marker, because one is already answered, or
  because the object set moved between the commit a run read and this target.
  The candidate set is defined as pre-admission, and a candidate whose fate is
  recorded is not one.
- **More than one qualifies.** Nothing names which, and the remedy is the act
  the design places after the comparative reading in any case: disposition one
  run's items, and the selection is unambiguous.

**Fewer than two candidates is the interpretation fixed in advance.** A widening
run that returned one configuration leaves the comparative reading nothing to
compare, so the position is **not exercised** — and that outcome is recorded
rather than left unstated. The verb refuses, names the interpretation, and still
stages a run whose bundle carries no candidate item and whose manifest carries
`candidate_run`, `candidates` (the derived run's own item count) and
`exercised: false`. Ingest that run and it commits a comparative run with an
empty item set naming the widening run, so the outcome of a widening run is one
shape either way.

Report from the JSON at this position: `candidate_run`, `candidates`,
`not_exercised`, and — on either derivation refusal — `widening_runs`.

**The loop, at this position.** Assemble at widening, dispatch, ingest, then
**commit what the ingest wrote** — the reading records under
`.abcd/work/issues/readings/<run>/` and the run's own artefacts under
`.abcd/development/readings/<run>/` — and then assemble at comparative. The
commit is the step between the two readings, and without it the assembly refuses
on the dirty gate naming the ingest's own records.

Assembly reads the working tree, so it refuses unless HEAD resolves to the
target **and** no included path is uncommitted. The preset configuration is in
that dirty set, for the same reason the record configuration is: an
uncommitted edit to it reshapes the assembly. Both refusals exit 2, as does
an unknown position, a missing operand, and any positional argument.

Report from the JSON: `run_id`, `position`, `target_commit`, `item_count`,
`manifest_hash`, and — where the run wrote — `out_dir` and `artefacts`.

Report `preset` too: `preset.selectors`, the committed entry the run applied,
resolved to its clauses. The written manifest carries the same block under
`preset`, beside `preset_hash`, the entry's content hash — which is what makes a
run reproducible from the commit it names, and what lets a reader tell two runs
apart by the entry they applied. There is **no** override stamp and no scope
source: nothing at the invocation can depart from the committed entry, so there
is no departure to report.

Also report `size`, on every run including a dry run: the total `bytes` and
`tokens_est`, and each row of `by_kind` (`kind`, `items`, `bytes`,
`tokens_est`). Report `size.unscanned` too when it is above zero: it is how many
of the run's items the exclusion floor did not examine. The include table
declares, per row, whether the floor parses what the row admits; a row it does
not parse — source, tests and configuration — hands each item over whole, and
every manifest item carries a `scan` mark saying `parsed` or `unscanned`. The
manifest's key and heading exclusions are asserted for the items marked `parsed`
and for no other, so `unscanned` is a disclosure rather than a warning: it is
what an operator weighs before dispatching a bundle. Report `tokens_est` as an
estimate and say so, quoting the
report's own `basis` — it is bytes over a measured constant, not a tokenizer's
count, and it mis-states each kind by a few per cent in directions spc-68
records. There is no budget and no threshold: the assembler cannot know what a
given reader accepts, so it reports the weight and the operator decides whether
to dispatch it.

Report `size.window` and `size.exceeds_window` beside those figures. The
committed entry for the position declares the estimated-token window it was
calibrated for, together with the figure it measured (`measured_tokens_est`,
`measured_bytes`) and the commit it measured on (`measured_at`). A file at
preset schema version 1 declares none, and the report says so rather than
showing a zero. Nothing is refused for either: `exceeds_window` is what the
cold-reading eval lane fails on, and the operator is told here. Report
`size.over_target` too when it is true — the total is over the two hundred
thousand estimated tokens an entry aims at, which is a target and not a limit,
and the reader's window decides whether it is acceptable.

At the **entailment** position only, report `size.mechanism`: `stated`,
`none_stated` and `absent` over `intents`, the projected intent files this run
carried. It is the yield bound stated beside the findings — how many of the
intents the reading is about carry a causal claim, how many state none, and how
many carry neither. No other position's report has the field.

### Where the artefacts land

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reading assemble \
  --position entailment --target HEAD \
  --out .abcd/.work.local/scratch/reading-runs/manual --json
```

With `--out`, the assembled input (`bundle.json`) and the manifest
(`manifest.json`) are written into that directory as two separate files.
Without it, they land in the local-tier run directory
`.abcd/.work.local/scratch/reading-runs/<run-id>/`. With `--dry-run` and no
`--out`, nothing is written anywhere and the result is rendered only.

An output directory the include table can reach is refused, and the refusal
names the item that would be admitted. Writing a run where the table reaches it
commits the next run's contamination: the artefacts land as ordinary files, a
later commit puts them in the tree, and the instrument reads its own output.
Write outside the repository, or under the local tier. Both artefacts are also
refused as INPUT wherever they are found, by their `_type` tag, so a run
committed before this was true cannot ride in either.

`--out` must name an empty or absent directory: one run's artefacts are one
run's evidence, and dropping them beside another run's leaves a directory whose
manifest describes half of what is in it. Both files are written through a
temporary name and renamed into place, so a reader never opens a half-written
bundle.

### The host obligation this binary cannot discharge

The assembled input carries no repository path: each item is an ordinal key, a
material class and its text, and only the manifest maps a key back to a path,
a field and its `scan` mark. That is the half of the isolation the binary
enforces.

The other half is yours. When you dispatch an assembled input to a reader,
grant that reader **no repository access** — no file tools, no path, no working
directory. Hand it the bundle's items and nothing else. A reader that can open
the repository is instructed blindness with extra steps, and no manifest can
detect it after the fact.

## Ingest one reading's output

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" reading ingest \
  --reading-json ./reading-output.json --json
```

`--reading-json` names the JSON the reading returned. It is the only operand
that names the output: the output states its own run, position and regime, and
there is no flag that could set one. A missing operand, a positional argument,
and every refusal below exit 2. The one other flag, `--route`, routes the agent
the output came from and sets nothing the output carries.

**Model-tier routing.** The ingest dispatches the cold-reading agent of the
position the output names (`cold-reading-<position>`), and resolves that agent's
model tier before anything else runs: an invocation override, over the
repository's `.abcd/config/oracle-routing.json`, over the machine's
`~/.abcd/oracle-routing.json`, over abcd's bundled proposal (which applies only
once a table is accepted). The override is `--route
<agent>=<tier>[@<connection>][?k=v,...]`, naming the one agent this invocation
dispatches (a second `--route` is refused, not merged), with the tier one of
`local`, `economy`, `frontier` or `host-decides`; it governs this run alone.
`assemble` takes no `--route`: its invocation is a position and a target and
nothing else, so run the reading at the tier you mean to and pass that route to
the ingest, which records it as an override. The receipt's `model_reported` is
the output's `instrument.model`. The ingest's `--json` result carries a `route`
receipt (`tier_asked`, `connection_tried`, `connection_used`, `fallback_reason`,
`override`, `settings_sent`, `model_reported`) and its text a `route:` line;
relay it with the result. When no configured provider can serve the tier, one
stderr line says the step goes through the harness instead. A `--route` naming
an agent this invocation does not dispatch, a tier outside the set, a connection
this machine has not configured, or a routing table that cannot be read exits 2
before anything is written. With no table accepted and no `--route`, the step
asks for `host-decides` and nothing is printed.

### What the output carries

One JSON document per run. The envelope names the run the assembly parked
(`run_id`), the position it read at, the regime it claims, the content hash of
that run's manifest, and the instrument — the model, the definition's content
hash and the assembler version, all three required. Each item is a flat object
carrying `pattern`, the pattern the reading read under, and exactly the body
fields its position declares:

| Position | Body fields |
| --- | --- |
| `widening` | `configuration`, `what_admits_it` |
| `entailment` | `claim_surfaced`, `claim_type`, `what_implies_it` |
| `comparative` | `candidate_id`, `criterion`, `characterisation` |
| `detection` | `tension`, `constraint_in_play`, `why_a_tension` |

**An item carries no identifier.** Identifiers are minted by the verb, and a
payload supplying its own is refused as an unknown field. Unknown fields are
refused at every level, so every violation names a field rather than guessing
at one.

### What is refused, and how far

An **item-level** violation refuses that item and lands the rest: an empty or
absent `pattern` at any position, a field the position's body does not declare,
or a reserved name carried as one of the item's own keys. At the **comparative**
position two more, both checked against the run's own manifest rather than
against the payload's account of itself: a `candidate_id` naming an item the
recorded widening run does not hold (`unknown-candidate`), and a `criterion` the
criteria discipline does not declare (`undeclared-criterion`). A **list-level**
violation refuses the whole run: a wrong `_type`, a run id that resolves to no
parked manifest, a manifest hash that disagrees, a position whose definition
does not resolve (absent, malformed, or stating another position's licence), an
instrument claiming a definition hash or an assembler version the artefacts do
not carry, a regime disagreeing with the definition, or a payload in which no
item survived. Blankness is judged on a folded copy everywhere the verb judges
it — the pattern, every body field, the instrument's three parts — so a value
that renders as nothing (a zero-width rune, a variation selector, the braille
blank) is empty.

**A refusal leaves a record once the run's identity is proven** — that is, once
the run id resolves to a parked manifest whose content hash matches. From that
point every list-level refusal writes `refusal.json` under the run's directory,
carrying the run metadata and the named reason and no items, and the refusal
message and the JSON render both name it. A run refused because its definition
did not resolve records the same way, with no `regime` in the record — the verb
resolved none, and the reason names the definition instead. The refusal list in
a reason is bounded, and its elision entry is not an item: it renders under its
rule alone, and there is never an item 0. A refusal reached BEFORE that point —
a wrong `_type`, a run id that resolves to nothing, a manifest hash that
disagrees — writes nothing durable anywhere, because there is no proven run to
record against.

**A rerun is a new run with a new run id, never an amendment.** Once a run id
has an outcome — a commit marker or a refusal record — ingesting it again is
refused. Assemble again, and ingest the run that assembly parked.

**A run that returned NO items is committed, at every position.** An empty item
list is the clean-run idiom the design framework's section 13 fixes: the null
result is recorded as a run with an empty item set, never refused, and refusal
is reserved for a malformed payload. The comparative position's not-exercised
outcome is one instance of that rule — its run record carries `candidate_run`,
`candidates` and `exercised: false` — and an empty output at widening,
entailment or detection commits the same way. A run whose every item was refused
is a different fact and is still a list-level refusal: recording it as a run
that returned nothing would lose what happened.

### The supply regime is the definition's

Each position's definition states the regime the reading reads under, and the
verb reads it from there. An output whose self-declared regime disagrees is
refused. No operand and no configuration key sets a regime, by design.

Per regime, the reserved names — an item carrying one as a field of its own is
refused with the licence stated. The table is read at the run's own regime, one
row per regime:

- `evaluative`: `order`, `rank`, `recommended`, `score`. Arrangement order is
  never refused; items arrive in document order by mandate.
- `registrative`: `fix`, `remedy`, `resolution`.
- `explicative`: `disposition`, `status`.
- `generative` has no reserved names. Its licence is the widest, and the
  constraint on it falls at admission rather than here.

**The gate refuses only a real decision field.** A reserved name is matched
against the KEYS of the item — its own fields, and the keys of any nested object
the contract does not define — and never against the words inside a value. A
reading REPORTS: it quotes the record's `disposition:` line, says what a clause
settles, what a paper recommends, what a suite scores, and which section says a
fix is merged while another says pending. All of that lands at every regime. The
same word carried as a field of the reading's own output is the decision the
licence withholds, and it refuses. Keys are compared folded, so a reserved name
respelled in code points that render the same refuses as itself.

The gate reads no prose. It once carried a registry of semantic signatures over
an item's text; measured over thirty-four realistic outputs it caught fourteen,
every one for quoting the document it read, so the registry is gone rather than
softened.

### Where the records land

No OTHER run's durable state is written to or deleted from until the whole
payload validates; a refusal after the run is proven writes its refusal record
and nothing else, and before its identity is proven a run writes nothing durable
anywhere. Once the payload validates the reading records land in the
reading-record family as one batch,
the run's manifest is promoted beside its run metadata, and the run metadata is
written **last** as the commit marker — a run without one never happened.

An ingest interrupted before that marker leaves a stage in the local tier.
Every later invocation names that orphan; the next one whose payload validates
sweeps it: it **rolls that run's reading records out of the committed ledger** —
the run never happened, so it must leave none — and clears the stage. Until
then the bare verb reports it as `orphaned_ingests`. A stage left behind AFTER
the marker — the commit path could not clear it — is reported as
`leftover_stages` instead: that run is complete, and the sweep clears the stage
and leaves its records alone. A refused run destroys no other run's records and
reports the orphans it left in place; the one rollback a refusal does perform is
of its OWN earlier crashed attempt, because a refused run leaves no reading
records. The ids a sweep removed are reported however the invocation ends. One
ingest runs at a time in a checkout: a second waits, and reports contention
rather than sweeping the first one's records away.

Report from the JSON: `run_id`, `records`, `refused_items`,
`cleared_stages`, `rolled_back_records`, `pending_stages`, and `run_record` —
or, on a refusal that recorded one, `refusal_record`. A refusal renders the
JSON whenever it has one of these to disclose, so read it on exit 2 as well.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.

**User input:** $ARGUMENTS
