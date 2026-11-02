---
id: itd-2609292109475690
slug: every-family-of-abcd-verbs-has-at-least
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
related_issues: [iss-48]
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# Every verb family has a behavioural scenario that drives the built binary end to end

Typed links: `related_issues` [iss-48](../../../work/issues/open/iss-48-e2e-behavioural-scenarios.md) (the record this draft plans).

## Press Release

> Every family of abcd verbs has at least one scenario that drives the built binary through a real round trip, in continuous integration on macOS and Linux. Where the smoke lane proves that help renders and that the verbs writing a committed record land what they claim, the scenario suite follows a person's actual sequence per family: capture an issue, resolve it and list it; install abcd into a scratch repository; ask the memory store a question over a fixture; lint the docs; capture and list a transcript; preview a release. "A change once passed every gate while the verb it touched refused every real input," said Bob, a staff engineer on a platform team. "Now each verb family is driven end to end the way a person uses it, and that kind of break fails the change."

## Why This Matters

The smoke lane (`evals/smoke_test.go`) is structural: every command's help
renders and the read-only verbs run without a panic. The write-path smoke
(`evals/smoke_write_test.go`, the product thinker's ruling M11 of 2026-09-23)
runs the verbs that write a committed record against a scratch repository.
Neither drives a person's sequence through a verb family: iss-48 lists capture,
ahoy, memory, docs, history and launch as families with no behavioural
end-to-end scenario. The product thinker ruled on 2026-09-29 to plan a
behavioural scenario suite per verb family (ruling J24).

## What's In Scope

- **One scenario per verb family** at least, run against the cross-compiled
  binary in a scratch repository and a scratch home, asserting what lands on
  disk and what the verbs print.
- **The families iss-48 names first**: capture, ahoy, memory, docs, history,
  launch; the planning interview decides whether every family in
  `abcd --help` is in scope.
- **Continuous integration on both platforms**, in the tagged lane the smoke
  harness already uses.

## What's Out of Scope

- Anything that reaches a network service or a paid key: a scenario runs
  offline.
- Replacing unit tests: a scenario proves the wiring, not every branch.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

Seeded by the run from iss-48 and the ruling, unconfirmed: the planning
interview walks each one.

- **Given** a scratch repository and home, **when** the capture scenario runs,
  **then** it captures an issue, resolves it and lists it through the built
  binary, and the record is in `resolved/` with its resolution.
- **Given** a scratch repository, **when** the ahoy scenario runs, **then**
  install writes the scaffold and a second install reports nothing to do.
- **Given** a fixture memory store, **when** the memory scenario asks a
  question, **then** the answer cites the fixture's pages.
- **Given** each family in scope, **when** the suite runs in CI on macOS and
  Linux, **then** every family has at least one passing scenario and a family
  with none fails the suite by name.

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [verification-assurance SOTA note](../../research/notes/2026-10-23-verification-assurance-sota.md). The ground is shared with itd-75, whose fixtures and this intent's scenarios are one format.

- **A declarative script a reviewer reads without Go exists and is the Go project's own**: a `testscript` file holds the commands, the expected output (regular-expression match or `cmp` against a file in the same archive) and the fixture tree, runs in a fresh work directory with `HOME=/no-home`, and gates a line to one platform with a condition such as `[unix]` (`testscript` package documentation; `cmd/internal/script/scripttest` runs the `go` command's own `testdata/script/*.txt`).
- **The same engine has a standalone runner**, so a scenario is usable from CI or a non-Go project without `go test` (one practitioner's report of the `testscript` command).
- **BATS** reads to anyone and emits TAP but needs Bash on every runner and keeps the fixture tree outside the test file (bats-core documentation); **godog** binds Gherkin steps to Go functions by regular expression, is pre-1.0 with a deprecated CLI, and reintroduces the step-definition tier itd-109's fit-challenge refused.
- **Every family, by detection**: the smoke harness already walks the Cobra tree in-process, so a detector that fails on a top-level verb with no non-empty `scenarios/<family>/` directory is a few lines over an existing walk, not a hand-kept list.
- **Its own tagged lane** is a question of run time and failure attribution, not of standing down: the CI classifier already stands the smoke harness down on a docs-confined pull request (with the race lane and the workflow audit), so scenarios inside that lane inherit the stand-down either way.
- **Locally**, neither `rogpeppe/go-internal` nor `golang.org/x/tools` is a dependency today.

**Path 1 for the format, shared with itd-75: scenarios are testscript files run from the harness.** This adds a test-only Go dependency on `github.com/rogpeppe/go-internal` and needs explicit sign-off; if refused, Path 2 is itd-75's in-house subset runner over the same files. The detector and the lane are native either way. The independent fit-challenge runs at plan time.

## Open Questions

- **Every family or a named list**: does the suite have to cover every
  top-level verb, with a detector that fails on a family with no scenario?
- **Where it runs**: inside the existing smoke lane or its own tagged lane, and
  whether a pull request confined to docs skips it.
- **Scenario format**: Go tests beside the smoke harness, or a declarative
  script format a reviewer can read without Go.
- (SOTA 2026-10-23) A family whose only real round trip needs the network or
  a paid key (`launch`, parts of `memory`) is out of scope by the intent's own
  rule; does the detector accept a scenario gated by a condition that is never
  true in CI, or must each such family carry an offline round trip?

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
