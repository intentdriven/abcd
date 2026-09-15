---
id: spc-65
slug: amnesia-is-a-repository-property-proven-by-an-eval-the-same
intent: itd-187
---
# Amnesia is a repository property, proven by an eval

## Summary

spc-65 delivers [itd-187](../../intents/shipped/itd-187-amnesia-is-a-repository-property-proven-by-an-eval-the-same.md):
a repository eval that assembles one reading definition twice over an unchanged
repository state and asserts the two assembled inputs are byte-identical, the
manifest excluded from the comparison. Amnesia becomes a property of what the
assembler passes, checkable by anyone with the repository, rather than an
instruction to an agent that only a case run could evidence. The decision log
entry of 2026-08-28, ruling (3), is explicit: amnesia is proven by a repository
eval, never by a case run, which is why the closing run of the cycle carries
only purpose durability and convergence.

The eval also pins the two determinism preconditions itd-187 places on the
assembler ([itd-183](../../intents/shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md),
spec [spc-61](../closed/spc-61-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md)):
hash-only manifests with no timestamps, and a lexicographic walk order. It lands
in the existing smoke harness package ([`evals/`](../../../../evals/README.md))
behind the `smoke` build tag, so `make smoke` and CI's `smoke` job run it with
no change to [`Makefile`](../../../../Makefile) or
[`.github/workflows/ci.yml`](../../../../.github/workflows/ci.yml). Passing in
CI is a cycle exit criterion.

spc-61 is being written concurrently. Where it is still a stub, this spec
designs against itd-183's own text and the assembler row of the cycle build
plan, and inherits the four interface demands
[spc-64](../closed/spc-64-the-read-block-eval-falsifies-the-firewall-planted-warm-cont.md)
places on it rather than adding a fifth.

## Scope

In: the double-assembly comparison and the identity relation it asserts; the
exclusion of the manifest from that comparison and the separate, weaker
assertions made about the manifest itself; the independent lexicographic order
oracle; the order-adversarial fixture that gives the oracle something to catch;
the comparator meta-test that proves the comparison can fail.

Out: everything under [Out of scope](#out-of-scope). In particular this eval
asserts nothing about *what* the assembler included, which is spc-64's property.

## Approach

### The identity relation

Two assemblies of the same definition over the same repository state at the same
commit must produce byte-identical assembled input. The comparison is a byte
comparison of the whole artefact, not a semantic one: a semantic comparison
would tolerate exactly the reordering and re-serialisation this eval exists to
refuse.

The manifest sits outside the comparison, because it legitimately carries a run
identifier that differs between runs. It is not, however, unasserted. Two
weaker properties are checked on each run's manifest: no key or scalar value in
it is timestamp-shaped, and its item paths appear in lexicographic order. That
is the whole of itd-187's "determinism preconditions it enforces on the
assembler", made concrete.

### Assembling twice, from two paths

The eval materialises the shared fixture corpus
(`evals/testdata/cold-reading/baseline/`, owned by spc-64), initialises it as a
git repository, and makes one commit. It then copies the whole tree, `.git`
included, to a second temporary directory, so the second assembly runs at a
different absolute path over an identical commit.

Assembling from two different paths is a deliberate strengthening of itd-187's
criterion, decided here: a run-to-run comparison in one directory cannot see an
absolute path or a temporary directory name embedded in the output, and that
leak is both a determinism failure and a breach of the repository's rule that no
absolute local path enters an artefact. Comparing across paths catches a path
the two runs DIFFER in, on the first run.

It cannot catch one they SHARE. Both trees are created under one temporary
parent, so a parent embedded in both outputs leaves the comparison agreeing and
reporting nothing — the same blindness one level up. The artefacts are therefore
also held to a path detector, on two mechanisms: every ancestor of the fixture's
repository root and HOME up to the process temporary directory, which is where
this harness's own directories stop and the machine's begin, and a shape match
for any other absolute path, which reaches the machine's own directories that no
list of known paths could enumerate. The comparison and the detector reach
different halves of one rule.

### The order oracle

Byte-identity alone is satisfied by any order the assembler picks consistently,
including a filesystem-dependent one that changes on another machine. So the
eval carries its own oracle: it collects the item paths from the assembled
input, sorts a copy with the eval's own lexicographic sort, and asserts the two
agree. The oracle is the eval's sort, never the assembler's, which is what makes
this a check rather than a restatement.

`evals/testdata/cold-reading/order/` gives the oracle something to catch: a set
of records whose names sort differently under byte order, case-insensitive
order, and numeric-suffix order, and whose creation order is deliberately not
their sorted order. The eval owns this directory; spc-64 shares the corpus it
sits in.

### Decisions this spec makes that the record did not

- **The timestamp scan is confined to the manifest.** Projected record bodies
  legitimately quote dates in prose, so a timestamp scan over the assembled
  input would fire on the record's own text. The manifest carries paths, field
  names, and hashes only, so a timestamp-shaped token there is unambiguously a
  defect.
- **Identity is asserted across two paths**, per the reasoning above, rather
  than twice in one directory.
- **The comparison is over the assembled-input artefact as a whole**, taken as
  a file from the assembler's dry-run output directory rather than from stdout,
  so that interleaved diagnostics can never be mistaken for a difference.
- **The eval binds to the verb, not the package**, matching spc-64: it invokes
  the built binary through the harness's existing `TestMain` build and `run`
  helper in [`evals/smoke_test.go`](../../../../evals/smoke_test.go). Here the
  reason is not independence from the include table but that a repository
  property must be checkable by running the shipped binary.
- **One definition, not four.** itd-187's claim is about the assembler's
  determinism, which is position-independent, so the eval asserts identity at
  one position and, for cheapness rather than principle, repeats the order
  oracle at each of the others.

### Wiring

Go test files in package `evals` with `//go:build smoke || coldreading`, so the
same files run in the smoke harness and in the cold-reading lane. `make smoke`
runs `go test -tags smoke ./evals/...`; `make evals-cold-reading` runs
`go test -tags coldreading ./evals/...`, and the `cold-reading-evals` CI job runs
that target. The target and the job were landed by spc-64, and selection is by
build tag rather than by test name, so this spec's tests joined both lanes with
no Makefile and no workflow edit of their own. The same
caveat spc-64 records applies: on a pull request confined to `docs/`,
`.abcd/development/`, `.abcd/work/`, and the root prose files, the diff
classifier stands the `smoke` job down, and the merge-queue entry, which runs
the full set on the would-be merge result, is what gates the merge.

**Ruled by the maintainer, 2026-08-30 — the evals get their own always-run CI
job.** The `smoke` job stands down on a pull request confined to `docs/`,
`.abcd/development/`, `.abcd/work/` and the root prose files, and those are
precisely the paths these evals read: a record-only change is the diff most able
to introduce warm content into material the assembler includes, so standing the
eval down there is anti-correlated with the risk. A stood-down job still reports
its check context green, which is a green for work that did not happen.

The remedy follows the reasoning `ci.yml` already documents for the ubuntu unit
lane, which never stands down because its tests read the live tree under the
allowlist. So: a small CI job carrying no `inert` condition, running these two
evals alone behind their own make target, while the rest of the smoke harness
keeps standing down. The workflow edit lands with this spec's build, reviewed
alongside the evals it serves; the merge-queue entry continues to run everything
regardless.

## Acceptance criteria mapping

The criteria were split on 2026-08-31, before this spec was built, because the
second criterion as written named a precondition no eval may establish for
itself. The numbering below is the positional authority ac-1..ac-4.

| itd-187 criterion | How spc-65 satisfies it | Pinned by |
|---|---|---|
| ac-1 — two assemblies from two distinct paths over one commit are byte-identical, manifest excluded | Two assemblies over one commit from two temporary directories, compared as whole artefacts taken from the dry-run output directory | `TestAssembledInputIsByteIdenticalAcrossRuns` |
| ac-2 — item paths agree with the eval's own lexicographic sort | The order oracle is the eval's sort, never the assembler's, run against the order-adversarial fixture, so a consistent-but-not-lexicographic order fails where byte identity alone would accept it | `TestWalkOrderIsLexicographic`, `TestFixtureOrderIsAdversarial` |
| ac-3 — a timestamp-shaped key or scalar in the manifest fails | The timestamp scan is confined to the manifest, which carries paths, field names and hashes only, so a timestamp-shaped token there is unambiguously a defect | `TestManifestCarriesNoTimestamp` |
| ac-4 — the comparator reports a difference naming the differing item | Two synthetic pairs, one differing in item order and one in a single scalar; the anti-vacuity guard without which a comparator that compared nothing would pass everything else here | `TestComparatorReportsADifference` |

The criterion these three replace read "given a nondeterminism introduced into
the assembler, the eval fails", and its Given is unestablishable by any shipped
artefact, since an eval must not patch the code under test. itd-187 discloses
the remainder as a recorded hand-run: the walk sort removed by a one-line local
patch, the test watched red, the patch reverted before the branch is pushed,
and the run recorded in the pull-request body.


## Tests

Every test below is watched red before the change and green after.

- `TestAssembledInputIsByteIdenticalAcrossRuns` is watched red against an
  assembler whose candidate slice is rebuilt from a map, so the order varies
  between runs. That is the mutation that proves the comparison can fail against
  real nondeterminism. Removing the walk sort does not: it yields a different
  order rather than an unstable one, so both assemblies still agree and the byte
  comparison stays green. The run is recorded in the pull request body and the
  patch is reverted before the branch is pushed.
- `TestWalkOrderIsLexicographic` is watched red against the removed walk sort,
  which is the mutation that suits it, and stays the standing guard afterwards:
  it fails on any consistent-but-not-lexicographic order, which the byte
  comparison alone would accept.
- `TestManifestCarriesNoTimestamp` is watched red against a manifest with a
  generation time stamped into it.
- `TestComparatorReportsADifference` feeds the comparator two synthetic
  artefacts differing only in item order, and a second pair differing only in
  one scalar, and demands a reported difference naming the differing item. It is
  the anti-vacuity guard: without it, a comparator that silently compared
  nothing would pass every other test here.
- `TestFixtureOrderIsAdversarial` asserts the `order/` fixture's creation order
  is not its sorted order, so the order oracle cannot pass by coincidence.

## Out of scope

- The assembler, its include table, its projection, and its manifest format:
  spc-61.
- What the assembler included or excluded, and the sentinel corpus that tests
  it: spc-64.
- The reading definitions
  ([spc-62](../closed/spc-62-four-cold-reading-definitions-one-blindness-core-each-positi.md))
  and the output contract
  ([spc-63](spc-63-one-ingest-verb-validates-every-cold-reading-output-includin.md)).
- Determinism of anything downstream of the assembler: a reading's own output is
  not required to be reproducible, and nothing here suggests it is.
- Cross-commit stability. The claim is identity at one commit; a changed
  repository state is expected to change the assembled input, and that is what
  makes the manifest's commit reference meaningful.
