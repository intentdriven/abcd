# SOTA survey and proposal — which `abcd lint` findings block a merge, and which only warn

Dated 2026-09-30. For the technical facilitator, who owes the product thinker
the proposal that ruling M10 (2026-09-23) asked for: "abcd lint becomes
merge-blocking on a narrow, high-precision subset and stays advisory for the
rest; the technical facilitator proposes the subset and the product thinker
confirms it", starting from real home-folder paths in privacy-hygiene. It
serves iss-2608231000561060. One host-run research pass (primary sources read
on 2026-09-30) plus a measurement of the rule set at the lane base, challenged
for fit per [`prefer-sota`](../../principles/prefer-sota.md).

## What the field does

**Block only what is never wrong; do not show what nobody acts on.** The
most-cited practice comes from Google's static-analysis programme:

- Error Prone's criteria for a check that fails the build: "The bug should be
  easy to understand", "The fix should be easy to make", "The bug pattern
  should have no false positives", "The bug should represent a correctness
  issue". A warning may have "very few false positives" — developers "should
  see actual issues at least 90% of the time" (Error Prone wiki, project tier).
- Tricorder, the code-review surface, holds analysers to "less than 10%
  effective false positives", where an issue is an effective false positive
  "if developers did not take some positive action after seeing the issue"
  (*Software Engineering at Google*, chapter 20; Sadowski et al., ICSE 2015
  and CACM 2018, production-practice tier).
- On warnings: "We have found repeatedly that developers ignore compiler
  warnings. We either enable a compiler check as an error (and break the
  build) or don't show it in compiler output" (same chapter). A warning tier
  that is shown on every run and never acted on trains people to ignore the
  tool.

**Gate on severity, chosen per check.** GitHub's code-scanning merge
protection blocks a merge on an alert threshold chosen from "None", "Errors",
"Errors and Warnings" or "All", with a separate security-severity threshold;
there is no default, so an administrator chooses (vendor tier).
golangci-lint lets a configuration assign severity by linter, path or text,
and can report only new findings (`new-from-rev`, `new-from-merge-base`) so a
backlog does not block every change (project tier).

**The shape these share:** a blocking tier chosen check by check against a
measured precision bar, a waiver on the line for the rare deliberate case,
an advisory tier that is either acted on or hidden, and a baseline or
diff-scope when the tree is not yet clean.

## The rule set at the lane base (measured)

`go run ./cmd/abcd lint --json` at 2b9d52fbb returns no findings and no
skips. iss-2609290845269642 (resolved) classified the 102 privacy-hygiene
findings that stood on 2026-09-29 and cleared them: 11 real machine values
(class A), 71 deliberate synthetic values that carried no waiver (B), 3
placeholders (C), and 17 that are not personal identifiers at all (D: cloud
metadata endpoints, a system prefix, RFC section numbers read as IPv4, a hex
fragment read as IPv6, Go selectors read as LAN hostnames).

The seven rules and where each is already gated:

| Rule | Severity today | Already gated elsewhere? |
| --- | --- | --- |
| `three-tier-layout` | error | no |
| `conventions-router` | error | no |
| `decision-durability` | warn | `lint-decisions` covers the log's append-only shape, not its presence |
| `docs-currency` | warn | yes: preflight and CI run `abcd lint docs` |
| `privacy-hygiene` | error (home paths, token shapes, non-reserved IP/MAC); warn (LAN and device hostnames) | partly: CI's gitleaks covers token shapes over full history; the commit-msg hook and record-lint cover session URLs and tool footers |
| `identity-positioning` | warn | no |
| `site-gates` | warn | yes: preflight's `site-render` runs `abcd lint site` |

## Adversary filter

- **Precision is the admission test, not severity.** Class D shows the
  network patterns misfire on correct content (RFC numbers, hex, Go
  selectors). A rule at `error` severity is not thereby fit to block.
- **Double-gating is cost without catch.** `docs-currency` and `site-gates`
  run as their own gates already; `abcd lint` blocking on them adds a second
  failure for one cause.
- **Token shapes are gitleaks' job in CI**, over full history. Blocking on
  them in `abcd lint` too is cheap and harmless (both are high precision),
  and it catches them at preflight, before a push, which gitleaks in CI does
  not.
- **Host-agnostic, no new dependency.** The gate must be `go run ./cmd/abcd`
  on the declared toolchain; no `jq`, no Python, no third-party action.
- **Script-first MVP.** The subset should be expressible without a new
  verb first: a Makefile target over existing output, promoted to a flag
  only once the subset has held for a release.
- **Principle [`enforcement-claims-are-facts`](../../principles/enforcement-claims-are-facts.md).** Whatever is proposed, the
  surface index must say exactly what blocks, and nothing more.

## Proposal (for the product thinker to confirm, amend or refuse)

**Block (tier 1), at preflight and in CI's check job:**

1. `privacy-hygiene`, **absolute home-folder path** (the M10 starting
   point). Error Prone-grade: easy to understand, easy to fix (a persona
   home or `~`), a waiver exists (`abcd-lint:allow` on the line), zero
   findings at the base, and it is the case that bit on 2026-08-23 (two
   sessions wrote a real home path into ledger bodies).
2. `privacy-hygiene`, **token shapes and private-key blocks**. Zero
   findings, gitleaks-equivalent precision, earlier catch.
3. `three-tier-layout`, **local tier not gitignored** and **local-tier
   artefact in a committed tier**. Structural, zero false positives by
   construction, and a leak of ephemera when wrong.
4. `conventions-router`, **no AGENTS.md at the root**. Structural and
   binary.

**Advisory (tier 2), shown by `abcd lint`, never failing a gate:**
non-reserved IPv4, IPv6 and MAC addresses and LAN or device hostnames (the
class-D misfires; the iss-305 family of boundary fixes is resolved, so
revisit once a release cycle passes with no misfire on a correct line), `decision-durability`,
`identity-positioning`.

**Not re-gated through `abcd lint`:** `docs-currency` and `site-gates`,
which keep their own gates.

**Admission and demotion rule for later changes:** a check joins tier 1 only
with zero findings on `main`, a documented waiver, and a fix a newcomer can
make from the message alone. Any false positive observed in tier 1 demotes
the check to tier 2 in the change that captures it — the block is never
worked around with a waiver added to unblock one merge.

**Mechanics, script-first:** a `lint-blocking` Makefile target that runs
`go run ./cmd/abcd lint --json` and fails when any finding matches the tier-1
list (rule id plus finding kind), added to preflight and the CI check job.
The tier-1 list lives in one place. A native selector flag (for example
`--block <rule[:kind]>`) is the tool rung, taken only if the target proves
the subset. If the tree is ever not clean when a check is admitted,
golangci-lint's diff-scope (`new-from-merge-base`) is the precedent to copy,
not a baseline file.

What would show this wrong: a tier-1 finding on a correct line within the
first release cycle, or a class of leak landing on `main` that tier 2 flagged
and nobody acted on.

## Review record

2026-09-30: authored in one pass by an implementer in autonomous run A (lane
drainResearch), with the fit-challenge run in-pass by the author. No
independent adversarial reviewer has read this note yet; per the
[research protocol](2026-08-22-sota-research-protocol.md) that review is owed
before the ruling adopts it.

## Sources (all accessed 2026-09-30)

- [Error Prone — Criteria for new checks](https://github.com/google/error-prone/wiki/Criteria-for-new-checks)
- [Software Engineering at Google, chapter 20: Static Analysis](https://abseil.io/resources/swe-book/html/ch20.html)
- [Sadowski et al., Tricorder: Building a Program Analysis Ecosystem (ICSE 2015)](https://research.google/pubs/tricorder-building-a-program-analysis-ecosystem/)
- [Sadowski et al., Lessons from Building Static Analysis Tools at Google (CACM 2018)](https://research.google/pubs/lessons-from-building-static-analysis-tools-at-google/)
- [GitHub Docs — Set code scanning merge protection](https://docs.github.com/en/code-security/code-scanning/managing-your-code-scanning-configuration/set-code-scanning-merge-protection)
- [golangci-lint — configuration file (issues and severity)](https://golangci-lint.run/docs/configuration/file/)
