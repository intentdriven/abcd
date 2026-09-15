# evals — repository evals

Two lanes share one harness and one built binary: a self-discovering smoke test
for the `abcd` binary, and the cold-reading read-block eval.

## The smoke harness

A self-discovering smoke test for the `abcd` binary. It builds the real binary,
walks the Cobra command tree **in-process** (via `cli.NewRootCommand()`) to
discover every command and flag, and exercises each against the built binary — so
a command added tomorrow is covered here with no edit.

## What it checks (v1)

- **Every** command and subcommand: `abcd <cmd> --help` exits 0, produces output,
  and never panics. This catches the failure unit tests miss — a command that
  compiles but crashes when actually invoked.
- **Read-only, no-argument verbs** (`version`, the bare status board) run for real
  to a graceful exit.
- **Flag hygiene:** an unknown flag is a clean non-zero error, not a panic.

## The cold-reading evals

`coldreading_*_test.go` falsifies the cold-reading input assembler's read-block
rather than restating it. It plants a sentinel token for every warm location
class the record names into a fixture repository state under
`testdata/cold-reading/`, materialises and commits that state in a temporary
directory with `HOME` redirected to a planted transcript store, runs
`abcd reading assemble` over it at each reading position, and asserts three
absences over what the assembler wrote: no planted token in the raw
serialisation, no excluded frontmatter key or heading at any depth in a parsed
item, and no item from an excluded record family.

Two assertions sit beside those three and are about the manifest's own claims
rather than about what was selected. Every item's `kind` must name the material
class the record gives its path, in the manifest and in the bundle alike, so the
per-item attestation has a falsifier — that is also the only thing that catches
the include table's basename-SUFFIX row going missing, since the source row
admits a `_test.go` file either way and no path changes. And every family the
oracle refuses must be NAMED in the floor the manifest asserts: the local ledger
tier was excluded by construction and asserted by nothing, so a reader could not
tell that the framing traces had been refused (brief invariant 16 in its
less-than direction).

The oracle is independent of the assembler by construction. The eval invokes
the binary out of process, its exclusion table is transcribed by hand from the
record (itd-183's exclusion list, brief invariants 14 and 15, adr-55) with the
source on every row, and `TestOracleImportsNothingFromTheAssembler` parses every
Go file here to check that nothing imports the assembler's package. An eval that
read the assembler's own include table could only ever confirm the table.

The heading half of the field-absence assertion reads **every spelling a heading
arrives in** — ATX, setext underline, raw HTML tag or heading role, and the
emphasis and code marks a title can carry — not ATX alone. An ATX-only scan
would report an item carrying a setext-underlined `Audit Notes` as clean, which
is the assertion satisfied completely by the leak it exists to catch. That the
assembler currently REFUSES those forms rather than emitting them is a property
of the assembler, not of this oracle: an oracle that can only see what the thing
under test currently emits agrees with it by construction.
`TestFieldAbsenceSeesEveryHeadingForm` feeds the assertion the item the
assembler would have to emit, and its negative rows — prose naming the heading,
a frontmatter close, a thematic break, a table divider — hold the widened scan
to reporting headings rather than everything.

`testdata/cold-reading/baseline/` holds every plant in its canonical home;
`holed/` is the negative control, holding the replacement content for the two
files a relocated plant lands in; `refused/` holds the shapes the exclusion
floor cannot redact and must therefore refuse, one variant directory each;
`home/` is the fixture HOME carrying the planted transcript store, keyed on the
fixture's root-commit sha at materialisation.

### What keeps it from passing vacuously

An absence eval's characteristic failure is asserting nothing while looking
green, so the corpus is adversarial per rule and three separate guards stand
under the assertions:

- `TestEverySentinelIsPlanted` — the corpus keeps its plants, at their declared
  count, in their declared homes, all tracked by git (the assembler walks the
  tracked set, so an untracked plant tests nothing).
- The carrier floor — every plant-bearing file the include list names arrives at
  each position, and its own **cold marker text** is in the bundle's bytes. A
  manifest names what an assembly says it passed; only the bundle says what it
  actually passed.
- The declared table sizes — each oracle table asserts its count rather than
  merely being non-empty, because a `> 0` floor on a table whose size is known
  lets it halve unnoticed.

The plants are chosen so that each rule of the assembler's contract has one that
dies when the rule is removed — including the **positive** half of the field
projection, which needs a section that is neither projected nor on the exclusion
floor, and each excluded heading, which needs a home on a record type that
travels whole (on a projected type the projection keeps the heading out whatever
the floor says, so its exclusion cannot be falsified there).

The exclusion floor's **fail-closed half** needs a corpus of its own, because a
leak cannot reach it: removing a refusal admits nothing new against material with
nothing to refuse, so the whole redaction verifier could be deleted with this
lane green. `refused/` is that corpus — a file the include table admits whole,
carrying an excluded heading in a form the section scan does not report, so the
redactor has no span to delete. `TestTheAssemblerRefusesAnUnredactableShape`
requires the run to be refused and the refusal to name the heading; when the
guard goes, the same binary exits 0 and the test reads the leaked token back out
of the bundle.

`coldreading_coverage_test.go` is the matrix: one row per rule, the mutation that
removes it, and the plants that die. A rule no mutation can falsify carries its
reason in `Gap` rather than being quietly omitted, and
`TestEveryAssemblerRuleHasAFalsifier` fails if a row names a plant or a refusal
that has gone, if either is named by no row, or if the number of declared gaps
changes.

## The amnesia eval

`coldreading_determinism_test.go` and `coldreading_order_test.go` check the other
half of what the assembler owes a reading: the same repository state, assembled
twice, produces the same assembled input. Amnesia is a property of what the
assembler passes rather than an instruction an agent is trusted to follow, so it
is checked here and no case run is spent evidencing it.

The identity relation is **byte-equality of the bundle**, the manifest excluded,
because the manifest carries a run identifier that differs between runs by
construction. The two assemblies run at two **distinct absolute paths** over one
commit — the corpus is committed once and the tree copied, `.git` included — so
an absolute path or a temporary-directory name embedded in the output fails on
the first run. A run-to-run comparison in one directory cannot see that leak, and
it is both a determinism failure and a breach of the rule that no absolute local
path enters an artefact.

The artefacts are also held to a **path detector**, not a list of two names.
Both trees are created under one temporary parent, so that parent is an absolute
local path both runs carry identically — the byte comparison agrees about it and
reports nothing, which is the same blindness the two-path design exists to close,
one level up. The detector runs two mechanisms: every ancestor of the fixture's
root and HOME up to the process temporary directory, and a shape match for any
other absolute path, which is what reaches the machine's own directories that no
list could enumerate. `TestTheAbsolutePathGuardSeesMoreThanTheTwoRoots` falsifies
each half separately, and its negative rows — repo-relative item paths, a URL —
hold the detector to reporting paths rather than every slash. Failure messages
name a leaked path by its last two components alone: a guard against absolute
paths in artefacts must not put one in a CI log itself.

The manifest is not therefore unasserted. Two weaker properties hold on it: no
key and no scalar in it is timestamp-shaped, and its item paths agree with the
eval's **own** lexicographic sort. The scan is confined to the manifest because a
projected record body legitimately quotes dates in prose; the manifest carries
paths, field names and hashes only, so a timestamp-shaped token there is
unambiguously a defect. Two exemptions are declared rather than silent. The run
identifier mints from a clock, so it is exempt from the packed-digit rule alone,
and the manifest is not described as timestamp-free. The exemption is keyed on
the NAME at any depth while the shape assertion reads the top-level field only,
so a nested key spelled the same way would be exempt and unchecked; the closed
manifest structs declare no such field, which is what makes that unreachable
rather than merely unobserved.

The second is the **minted record identifier**, and it is the one the manifest's
own contents force: a path names a record, and every minting family allocates
`<family>-<yymmddHHMMSS><4 random digits>` (adr-45), which is sixteen packed
digits. The premise the scoping decision rests on was therefore already false
for paths — fed a real id the scan fired, and it stayed quiet only because the
fixture corpus names its records shortly (`iss-2608311502474022`). The exemption
is narrow: the family list is closed and hand-transcribed, so a directory named
after a moment (`run-20260831131824`) is not exempted by looking id-shaped; the
elision is token-local, so a packed moment sitting beside a record id in one
path is still reported; and it lifts the packed-digit rule alone, so an id
carrying an ISO date or a clock time still fails.

### What keeps it from passing vacuously

An identity assertion fails the way an absence assertion fails — two artefacts
that agree because both are empty are green and worthless — so four guards stand
under it:

- `testdata/cold-reading/order/` is the order-adversarial corpus: six records
  whose names sort one way by byte, another by case-folded comparison, a third by
  numeric suffix and a fourth by path component, materialised in a creation order
  that is none of the four. `TestFixtureOrderIsAdversarial` asserts those
  disagreements hold, so the order oracle cannot pass by coincidence. The shared
  baseline corpus alone cannot catch a numeric-suffix comparator, nor a
  component-wise one — which is the order a directory walk yields, and so the
  likeliest wrong order of the four; this corpus catches both.
- `nonVacuous` refuses an assembly that has lost that corpus — by path in the
  manifest **and** by its own text in the bundle — rather than a bare "any item"
  floor, which a bundle of empty texts satisfies exactly.
- The two runs are required to report **different run identifiers**, so the
  comparison is over two invocations rather than one artefact read twice. One
  repeat is allowed before that is believed: an identifier is a one-second stamp
  and a uniform four-digit draw, so a single collision is a documented outcome of
  the mint and only a second one is evidence.
- `TestComparatorReportsADifference` feeds the comparator artefacts that differ
  only in item order, only in one item's scalar, and only in the artefact header,
  and demands a reported difference naming the item that differs.

## Running them

Both lanes sit behind the `smoke` build tag so they stay out of the fast
unit-test lane:

```bash
make smoke                       # both lanes
go test -tags smoke ./evals/...  # the same thing

make evals-cold-reading                # the cold-reading evals alone
go test -tags coldreading ./evals/...  # the same thing
```

`make preflight` runs both targets as prerequisites, so a push carries them —
it did not until `iss-2608311632382737`, and a defect in the eval that certifies
the read-block passed every local gate and surfaced only in CI. It names both
even though `smoke` compiles a superset of `evals-cold-reading`'s files: the tag
sets differ, so a cold-reading file that reaches for a smoke-only helper compiles
under one and not the other. Each lane costs about five seconds on a warm cache.

CI runs the smoke harness as the dedicated `smoke` job, and the release workflow
smokes the binary built from the tagged commit before publishing. The
cold-reading evals get their own `cold-reading-evals` job, which carries no
`inert` condition: the diff classifier stands the `smoke` job down on a change
confined to `docs/`, `.abcd/development/`, `.abcd/work/` and the root prose
files, and those are exactly the paths the cold-reading evals read. That job is
a **required status check** on the default branch — an always-run lane whose
check cannot block a merge reports a conclusion and stops nothing — and the
committed ruleset mirror under `.abcd/work/rulesets/` names its context.

A file visible to both lanes carries `//go:build smoke || coldreading`, which is
also how a later cold-reading eval joins the lane — no Makefile or workflow edit.

## `data/` (reserved for v2)

Fixture-driven, per-command scenarios — user-specified and synthetic inputs the
harness auto-discovers to drive richer smokes (e.g. `memory ingest` over a sample
corpus, `capture` round-trips). Deferred; the generalisation into an abcd-managed
eval framework is captured as intent **itd-75**.
