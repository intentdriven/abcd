---
schema_version: 1
id: "iss-2609120452071388"
slug: "the-slug-generator-can-emit-a-trailing-hyphen-that-its-own-s"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "peer session report from a downstream repo, 2026-09-12"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/issueschema/issueschema.go"
deferred_after: "v0.8.0"
deferral_reason: "The headline defect does not reproduce on this tree, and what remains of the record is not a bug fix. Every site that truncates a slug into a filename already trims the separator it cut against: capture roots.go, intent create.go and decide decide.go each wrap the truncation in a trim, and a sweep finds no fourth truncation site in the tree. Each deriver was run over 200000 adversarial inputs on a scratch copy, mixing separator runs, punctuation, non-ASCII and lengths either side of the 60-character budget, and none emitted a slug its own validator refuses. The three trims have stood since those functions were written, so the downstream record that prompted this was produced by something other than the current generator, which is consistent with its siblings carrying the source value autonomous-hunt that abcd capture would itself have refused. The three items that remain are real and none is contained: naming the members of the source and category flags in the help text, and making the status board say which layer refused a record it skipped, are user-facing surface changes that must not ship without a record, and deciding autonomous-hunt is a closed-vocabulary question this record routes to the product thinker rather than to an implementer. Waits on that vocabulary ruling, which is what the other two should land beside."
---

Reported from a downstream repository using abcd, where `abcd capture` (bare,
the read-only status render) printed three "skipped … malformed frontmatter"
lines for records **abcd itself had written**. The status board's counts then
silently exclude records that exist, and nothing tells the reader whether the
writer or the validator is the side that is wrong.

Two distinct causes, and the first is the serious one.

## 1. The slug generator produces a slug its own validator rejects

Reported record: `iss-190`, refused with

```
slug "guard-notebook-coverage-reads-notebooks-yml-as-one-flat-" is not kebab-case
```

The slug was truncated to its length budget mid-word, leaving a trailing hyphen.
`SlugRe` (`internal/core/issueschema/issueschema.go:150`) is
`^[a-z0-9]+(-[a-z0-9]+)*$`, which refuses a trailing hyphen — correctly. The
defect is that the generator can emit one: the truncation does not trim the
separator it cuts against, so writer and validator disagree about what a legal
slug is.

This is the sharpest shape of a record defect: **the tool wrote a record its own
gate refuses to read.** It is the same class as `resolveShipImpact`'s stated
purpose — that abcd must not "produce, out of its own verbs alone, a record its
own record-lint refuses" (iss-126) — arriving on a different path.

The failure is also silent in the direction that matters. The record is skipped
rather than reported as a fault, so a ledger quietly under-counts and the only
signal is a line the reader may not connect to the count.

## 2. `source: autonomous-hunt` is not in the accepted set

Reported records: `iss-299`, `iss-300`, `iss-301`, refused with `invalid source
"autonomous-hunt"`. They were written by an autonomous bug-hunt loop, so a value
one abcd workflow writes is not one the schema accepts.

`Sources` (`issueschema.go:140-144`) is `plan-review`, `impl-review`,
`manual-test`, `review-followup`, `agent-finding`, `agent-observation`,
`user-observation`, `drift-detection`, `memory-curation`.

**Corroborated independently in this repository on the same day.** Capturing
findings during the v0.8.0 release, `--source release-followup` and
`--category reliability` were both refused, each on the first attempt, each
plausible-sounding and neither in the list. The vocabulary is closed, which is
right, but it is discoverable only by being refused: `capture --help` renders
`--source string   surfacing channel (default user-observation)` and names no
member, and the same is true of `--category`.

So the refusal is correct and the surface around it is not. Two sessions
independently guessed values that do not exist, which is evidence about the help
text rather than about the guessers.

## What is owed

- **Trim the separator when truncating a slug**, so the generator cannot emit
  what `SlugRe` refuses. Then sweep: any other writer that truncates into a
  bounded filename has the same exposure.
- **Decide `autonomous-hunt`.** Either the bug-hunt loop uses an existing member
  (`agent-finding` is the obvious fit) or the value joins `Sources`. This is a
  vocabulary decision, not a bug fix, and it belongs to the maintainer.
- **Name the members in `--help` for `--source` and `--category`**, the way
  `--severity` already does ("severity: nitpick | minor | major | critical").
  Two independent wrong guesses in one day is the measurement.
- **Say which layer is wrong when a record is skipped.** A record that fails
  validation should report that it does, distinguishably from a record that is
  absent, rather than being dropped from a count with a line above it.

## Acceptance

- **Given** any text a caller captures, **when** the slug is derived and
  truncated, **then** the result satisfies `SlugRe` — no trailing or doubled
  separator — and a property test asserts this over adversarial inputs.
- **Given** a record whose frontmatter the schema refuses, **when** the status
  board renders, **then** the count and the diagnostic agree about what was
  excluded and why.
- **Given** `capture --help`, **when** a caller reads it, **then** the legal
  values for `--source` and `--category` are named there.
