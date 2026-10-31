---
id: itd-75
slug: cli-eval-harness
spec_id: null
kind: standalone
suggested_kind: null
reclassification_history: []
severity: minor
---

# abcd Proves Your CLI Actually Runs — From Fixtures You Drop In

## Press Release

> **Every command a repo exposes gets exercised, and you extend the coverage by dropping a folder — never by writing test plumbing.** abcd builds the binary, discovers the whole command surface by walking it (no hand-kept list), and smokes each command: help renders, nothing panics, read-only verbs run. Then it goes further — a fixtures folder of user-specified and synthetic inputs is auto-discovered and replayed against the matching commands, asserting the *shape* of what comes back. Adding a scenario is dropping an input and an expected-shape file into `evals/data/<command>/`; the harness picks it up on the next run, in local dev, in CI, and in the release gate against the very binary about to ship.
>
> "I used to find out a command crashed only when a user hit it," said Kira, a maintainer. "Now every command is smoked on every push, and when I want a real scenario I drop a sample corpus in a folder and abcd runs my tool against it. My confidence in a release stopped being a feeling."

## Why This Matters

Unit tests prove functions behave; they do not prove the *assembled binary* runs. A command can compile, pass its unit tests, and still panic the moment it is invoked — a broken wiring, a nil dependency, a flag that explodes on parse. The only honest check is to run the real binary through its real command surface. Doing that by hand rots: someone must remember to add each new command to the list. Discovering the surface from the binary itself removes that decay, and a fixtures-folder convention lets coverage grow by contribution rather than by editing a harness.

This is also the natural home for the "wired or it isn't done" rule: a command that is not reachable and runnable from the built binary is not done, and the eval harness is what makes that enforceable rather than aspirational.

## What It Looks Like

- **Self-discovering smoke (shipped as the v1 dogfood).** Walk the command tree, run every command's `--help` and the read-only verbs against the built binary, assert no panic and graceful exit. New commands are covered automatically.
- **Fixture-driven scenarios.** `evals/data/<command>/` holds inputs (user-specified real samples and synthetic generators) plus an expected-shape assertion. The harness matches folders to commands and replays them — `memory ingest` over a corpus, `capture` round-trips, `launch --dry-run` over a scratch tree — with no Go edit per scenario.
- **Any abcd-managed repo inherits it.** `abcd ahoy` scaffolds the harness and an empty `evals/data/` with instructions, and wires the smoke into the repo's CI and release gate. A repo becomes self-smoking by being abcd-managed.
- **Release-artefact smoke.** The gate runs the harness against the binary built from the tagged commit, so the thing that ships is the thing that was exercised.

## Dogfood (v1 running on this repo)

`evals/` in abcd-cli is the working prototype: a `smoke`-tagged Go harness that builds `abcd`, discovers the command tree via the exported root command, and smokes every command; wired into a `smoke` CI job, the release verify gate, and `make smoke`. This intent lifts that from a repo-local harness into an abcd capability every managed repo inherits, and adds the fixtures layer. Relates to the install surface (`ahoy`) and the "wired or it isn't done" principle.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [verification-assurance SOTA note](../../research/notes/2026-10-23-verification-assurance-sota.md).

- **The Go project's own answer to "fixtures you drop in" is one text file per scenario**: commands at the top, the fixture tree below `-- name --` markers (txtar), run in a fresh `$WORK` with `HOME=/no-home`; `exec`, `stdout`, `stderr`, `cmp` and `!` are the assertions and bracketed conditions gate a line to a platform (`testscript` package documentation; the same engine tests the `go` command through `cmd/internal/script`, and `rsc.io/script` is a published copy of it with a `scripttest` subpackage, shown at v0.0.2 from 2023 and unsupported by its own README, so the go command's engine can be imported after all; `go-internal/testscript`, v1.16.0, is the maintained fork).
- **txtar is built to be hand-edited and to diff in review**, and excludes binary data, file modes and symlinks by design (`golang.org/x/tools/txtar` documentation), which settles "committed samples" over seeded generators for determinism.
- **Rust's `trycmd` is the same shape with an explicit filesystem sandbox** (`*.in/` as cwd, `*.out/` compared after) and a dump-then-overwrite update loop; its own advice is blunt snapshot tests for the many and hand-written tests for the peculiar (trycmd documentation). cram is the language-agnostic ancestor and is unmaintained.
- **Gherkin with step definitions was already refused at itd-109's fit-challenge**; this pass found nothing new in its favour, and the practitioner case against it (a reported threefold maintenance cost) is anecdote from an author selling the alternative.
- **Snapshot updates get rubber-stamped**; the practitioner consensus is small, focused expectations, dynamic values excluded, and snapshot diffs read in review. The harness should make an update a visible act, never a flag that passes silently.
- **A fixture is marked safe by the repository's existing rule** ([secret-shaped-fixtures-at-runtime](../../principles/secret-shaped-fixtures-at-runtime.md)), carried in the archive's comment block, rather than by a new marker.
- **Locally**, `evals/` already builds the binary once and walks the Cobra tree; `evals/data/` is empty and reserved; neither `rogpeppe/go-internal` nor `golang.org/x/tools` is in `go.mod`, so either route to txtar is a new dependency.

**Path 1 for the format: scenarios are testscript files, adopted as is.** This adds a Go dependency and needs explicit sign-off, which weighs two routes: `github.com/rogpeppe/go-internal` (testscript v1.16.0, maintained) against `rsc.io/script` (a published copy of the go command's engine, v0.0.2 and unsupported); test-only if the harness runs it, runtime if the binary runs it for managed repositories. If refused, Path 2: a small in-house runner over the txtar grammar and a six-command subset, written so every file it accepts is a valid testscript file, which is the seam. The independent fit-challenge runs at plan time.

## Open Questions

- Fixture format: one folder per command with an input corpus + an expected-shape file, or a single manifest describing scenarios?
- How synthetic data is generated and kept deterministic (seeded generators vs committed samples), and how a repo marks a fixture as containing only safe, non-sensitive data.
- Which commands are safe to run for real by default versus opt-in, and how a mutating command declares a scratch sandbox.
- (SOTA 2026-10-23) If the runner must ship inside the abcd binary so a managed repository in another language inherits it, is a runtime dependency on `go-internal` acceptable, or does that case force the in-house subset runner?
- (SOTA 2026-10-23) Which of the existing Go write-path smokes (`capture`, `capture resolve`, `decide`) migrate to script scenarios and which stay in Go, so the repository does not hold two ways of doing the same thing.
