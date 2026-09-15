---
id: spc-69
slug: a-reading-is-about-something-narrower-than-everything-its
intent: itd-199
origin: researcher-authored
production_mode: hand-written
---
# The scope operand: a reading is commissioned about something, in a closed grammar, from a committed preset

## Summary

spc-69 delivers itd-199. `abcd reading assemble` takes a third operand naming
what the reading is about, and the assembly passes the intersection of what the
include table admits at that position with what the scope selects. A scope can
only narrow.

The presets are committed at `.abcd/config/reading-presets.json` and map
position to scope, so the distinctness of the four readings is a property of the
configuration rather than a hope about it. `warm` is `cold` plus a delta by
construction. The bundle carries the scope so the reading knows what it was
given; the manifest carries the effective scope, its hash, and whether the run
departed from the committed presets.

The comparative position refuses, naming the channel it lacks, rather than
assembling a corpus that is not its object.

This spec rests on adr-58, adopted by the maintainer on 2026-08-31. That ADR
supersedes maintainer ruling M8 and amends brief invariant 15, both of which
forbade a third operand; without it this spec would not be buildable.

## Scope

In: the scope operand and its grammar; the preset file, its schema and its
loader; the intersection; `warm` as `cold` plus a delta; the comparative
refusal; the scope in the bundle; the effective scope, hash and override stamp
in the manifest; the preset file in the dirty gate; the operand pin; the
definition/bundle precedence sentence in each definition; the assembler version.

Out: any route to the local ledger tier; widening beyond a position's admission;
the supply regime, which stays keyed on position; the size report, which is
itd-198's and landed first; a fifth position; the comparative channel itself.

## Approach

### The grammar, and one resolution the intent left open

A scope is one token in a closed grammar of three forms, resolved in this order:

1. **A record id** — `itd-N` or `spc-N`. Selects the items whose path names
   that record. `adr-N` and `iss-N` are deliberately NOT in the grammar: the
   include table names no decisions store and no issue store, so those tokens
   would validate, resolve, select nothing and end in the selects-nothing
   refusal at every position, while the CLI help and the generated reference
   advertised them. An affordance that can never work is worse than an absent
   one, because the operator's first move is to doubt their own invocation.
2. **A family token** — resolved as a **material kind** from `Kinds()`.
3. **A preset name** — a key of the committed preset file.

The second form is a deliberate resolution of something itd-199 left open. The
intent says "a record family"; this spec reads that as the material kind,
because `Kinds()` is already a closed, rendered, version-covered vocabulary and
adding a second family vocabulary beside it would be a third name for the same
distinction. It is also strictly more capable: `source`, `test`, `doc` and
`config` are not record families and are exactly the material a scope most needs
to exclude.

Collisions are refused at load rather than resolved by precedence: a preset
whose name is a kind token or a record-id shape is a configuration error, named
as one. Ordering the resolution is then belt and braces rather than the rule.

### The preset file

```json
{
  "schema_version": 1,
  "presets": {
    "cold": {
      "positions": {
        "widening":   { "kinds": ["brief-section", "glossary-term"], "records": [], "paths": [] },
        "entailment": { "kinds": ["intent-projection", "spec"],      "records": [], "paths": [] },
        "detection":  { "kinds": ["discipline", "spec"],             "records": [], "paths": [] }
      }
    },
    "warm": {
      "extends": "cold",
      "positions": {
        "widening":   { "kinds": ["doc"], "records": [], "paths": ["internal/core/lint"] }
      }
    }
  }
}
```

**`extends` is a union, never a replacement.** A preset naming `extends` takes
the parent's selectors at every position and adds its own. So `warm` cannot be
narrower than `cold` by construction rather than by review, and a scope added to
`cold` appears in `warm` without anyone remembering to add it twice. One level
of `extends` only; a chain is refused, because the guarantee is checkable at one
level and an argument at two.

**`paths` is the one place a repository path may be named**, per adr-58. The
loader refuses a path that is absolute, escapes the repository, names anything
the structural deny covers, carries a control character or surrounding
whitespace, uses backslash separators, or is `.` — a scope that selects the
whole repository narrows nothing, and a path carrying an escape sequence reaches
a rendered terminal line, which brief invariant 13 forbids.

**The file must be known to git and must not be a symlink.** adr-58 admits a
preset name at the invocation on the ground that it is committed and reviewed,
and the amended invariant 15 says so — so that has to be checked rather than
assumed. The dirty gate cannot supply it, and adversarial review demonstrated
both bypasses rather than inferring them: git reports nothing for a file it
ignores, so a repository gitignoring `.abcd/` (which brief invariant 5 does for
public visibility) ran against an untracked preset and stamped the run
`overridden: false`; and a committed symlink is tracked while git reports
nothing when its target changes, leaving the effective configuration
permanently unreviewed and freely mutable with every gate green. Both stamped
"ran as reviewed" on an examination establishing only "git reported no
modification", which is brief invariant 16's forbidden shape. The remedy is to
make the code hold what the invariant claims, not to soften the claim.

Precisely: `trackedSet` reads the git INDEX, not HEAD, so on its own it
establishes "known to git" rather than "committed". The committed property is
delivered by that check TOGETHER with the dirty gate, which refuses an
added-but-uncommitted preset as an `A ` entry. Both halves are needed — the
dirty gate alone missed an ignored file, and the tracked check alone would admit
a staged one — and saying "tracked at HEAD" would credit one line with what the
pair delivers.

**The shipped preset file is itself validated.** Every other test overwrites it
with a fixture, so the file the whole committed-and-reviewed argument rests on
was checked by nothing and could have shipped malformed, or with `warm` no
longer extending `cold`, or with `cold` degenerate at a position, with every
gate green. Three tests now load the committed file: that it parses, that the
shipped warm contains the shipped cold at every assembling position, and that
cold scopes the three positions distinctly.

**Duplicate preset keys are refused.** Go's decoder takes the last silently, so
a second block low in the file would replace the reviewed one while a reviewer
reading top-down sees the first. Against the one file whose whole safety
argument is that a human read it, that is a review-evasion vector.

**The comparative position has no entry**, and the loader refuses one. That
position refuses before a scope is ever resolved, so a preset carrying a scope
for it would describe a run that cannot happen.

### The intersection

A scope narrows and never widens. `Assemble` collects candidates exactly as it
does today, then filters:

```go
kept := cands[:0]
for _, c := range cands {
    if scope.selects(c) {
        kept = append(kept, c)
    }
}
```

`selects` is a union over the scope's selectors: a candidate is kept if its kind
is named, or its path names a selected record, or its path lies under a selected
path prefix. An empty scope at a position selects nothing, which is a refusal
rather than a silently empty bundle.

Because the filter runs AFTER collection, every structural property the
assembler already holds is untouched: the deny, the exclusion floor, the
tracked-set intersection and the dirty gate all run over the unfiltered walk. A
scope cannot admit a row the table denies, and the manifest's exclusion
assertions continue to describe the position rather than the scope.

**That ordering is pinned by a test, and was not.** Mutation review moved the
filter ahead of both gates and the entire package stayed green, which made the
paragraph above a claim nothing could falsify — the shape itd-195 says to make
executable or stop making. Neither gate could be made to notice by ordinary
means: the dirty gate's predicate is a pure function of the position, so it
refuses under either order, and the exclusion floor's own paths are structurally
denied, so no candidate can breach one to begin with. The property was real,
load-bearing and untestable at once. `assertExclusionsHook` is the seam that
makes it observable, and `TestGatesSeeTheUnfilteredWalk` fails when the filter
moves — verified by performing that move, not by asserting about it.

### What the artefacts carry

The bundle gains `scope`, as the reading's own fact: a reader told its object is
the shipped tree and handed a tenth of it will report the missing nine tenths as
a finding, so the bundle has to say what it was given.

**It is a projection, not the resolved scope, and the difference is brief
invariant 15.** The manifest may name repository paths — that is its job, and it
is why the bundle can be pathless and still checkable. The bundle may not: the
assembled input is the reading's entire working set and no repository path
enters its context. A scope's `paths` selectors ARE repository paths, so writing
one `Scope` type into both artefacts put a path into the reading's working set.
It did, and the probe that found it is now a permanent guard
(`iss-2608312058244357`). The bundle therefore carries the kinds and the record
ids it was scoped to, plus a COUNT of the narrowings by location and never the
locations — enough for a reading to know it holds a subset, carrying nothing
about where the subset came from.

The manifest gains `scope`, `scope_hash`, and `scope_overridden`. Naming a
committed preset is not an override; naming a record id or a kind directly is,
because that is a departure from what was reviewed and is the thing worth
counting.

Both shapes move, so `SchemaVersion` goes 2 to 3 and `AssemblerVersionCore`
goes 1.1.0 to 1.2.0.

### The definition/bundle precedence

Each definition's Object section gains one sentence: the definition states what
the position MAY read, the bundle states what THIS run was given, and where they
disagree the bundle governs. Without it a reading holds two accounts of its
object with no rule between them, and reports the difference as a finding.

The comparative definition carries the precedence sentence too, and then a
second paragraph recording that the position is not currently assembled and
why. An earlier draft gave it only the second, which left ac-12's "each
definition" true of three of four.

### The dirty gate and the operand pin

`PresetConfigPath` joins `LintConfigPath` in the dirty set, by the same argument
already written beside it. `readingOperands` gains `scope`, which is itd-184's
pin working as designed: the addition turns it red until someone states what the
operand does, and adr-58 is that statement.

## Acceptance criteria mapping

| ac | delivered by |
|---|---|
| 1 | required flag; refusal names the three token forms |
| 2 | record-id selector; intersection with the position's admission |
| 3 | filter runs after collection, so the deny and the floor are untouched |
| 4 | scope intersects admission; drafts stay denied at widening |
| 5 | `Bundle.Scope` |
| 6 | `Manifest.Scope`, `Manifest.ScopeHash` |
| 7 | `Manifest.ScopeOverridden`, false for a preset name |
| 8 | `PresetConfigPath` in the dirty set |
| 9 | per-position preset scopes; pinned across the three assembling positions |
| 10 | comparative refuses, naming the channel |
| 11 | `extends` unions, so warm ⊇ cold; pinned by a test over every position |
| 12 | precedence sentence in each definition's Object section |

## Tests

Watched fail before, pass after; each guard proved by mutation.

- `TestScopeIsRequired` — no scope refuses and names the grammar.
- `TestScopeNarrowsNeverWidens` — for every preset and position, the scoped item
  set is a SUBSET of the unscoped one. Mutation: a selector naming a denied path
  adds nothing.
- `TestScopeCannotReachTheLedgerTier` — an operator-written scope naming a local
  tier path yields no item from it.
- `TestDraftsStayDeniedAtWidening` — a scope naming the intent kind at widening
  admits no draft or planned item.
- `TestWarmContainsCold` — for every position, warm's selector set contains
  cold's. Mutation: making `extends` replace rather than union turns it red.
- `TestPresetNameIsNotAnOverrideAndADirectScopeIs`.
- `TestGatesSeeTheUnfilteredWalk` — the pipeline's order, proved by moving the
  filter ahead of the gates and watching it go red.
- `TestBundleScopeCarriesNoRepositoryPath` and
  `TestNoBundleFieldIsAScopeSelector` — brief invariant 15 held against the
  scope, after the first implementation broke it.
- `TestThreePositionsCarryDistinctItemSets`.
- `TestComparativeRefuses` — and names the channel.
- `TestPresetCollisionIsRefused` — a preset named for a kind or a record id.
- `TestUncommittedPresetRefuses` — the dirty gate.
- `TestBundleCarriesTheScope`, `TestManifestCarriesScopeAndHash`.

Both eval lanes are run explicitly and under `TMPDIR=/tmp`. The read-block eval
is run at a preset that keeps every carrier: three of its eleven carriers are
shipped-tree files, and `fence.go` is the sole corpus behind the body-redaction
row, so a scope dropping source would turn a real assertion into an undeclared
gap.

## Out of scope

- The comparative channel. Its absence is why that position refuses.
- Widening a scope beyond the table. A scope intersects and only narrows.
- Any scope reaching `.abcd/.work.local/`. Brief invariant 14, unconditional.
